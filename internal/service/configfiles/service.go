package configfiles

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/discovery"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
)

const (
	NameRestorePresets = "restore-presets"
	NameDiscovery      = "discovery"
)

type Service struct {
	dbgen              *dbgen.Queries
	restorePresetsPath string
	discoveryPath      string

	mu               sync.Mutex
	discoveryService *discovery.Service
	lastDiscoveryCfg discovery.Config
	// appliedAt tracks the DB updated_at of the config version currently
	// loaded in this process; Watch reloads when the DB has a newer one.
	appliedAt map[string]time.Time
}

func New(
	db *dbgen.Queries,
	restorePresetsPath string,
	discoveryPath string,
) *Service {
	return &Service{
		dbgen:              db,
		restorePresetsPath: restorePresetsPath,
		discoveryPath:      discoveryPath,
		appliedAt:          map[string]time.Time{},
	}
}

func (s *Service) markApplied(name string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appliedAt[name] = at
}

func (s *Service) appliedVersion(name string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appliedAt[name]
}

// SetDiscoveryService wires the discovery service for hot-reload after save.
func (s *Service) SetDiscoveryService(svc *discovery.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discoveryService = svc
}

// LastDiscoveryConfig returns the discovery config loaded during LoadAll.
func (s *Service) LastDiscoveryConfig() discovery.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastDiscoveryCfg
}

// LoadAll loads both configs from DB at startup.
// Falls back to the on-disk file if DB content is empty (first boot).
func (s *Service) LoadAll(ctx context.Context) error {
	if err := s.loadOne(ctx, NameRestorePresets, s.restorePresetsPath, s.reloadRestorePresets); err != nil {
		return fmt.Errorf("load restore-presets: %w", err)
	}
	if err := s.loadOne(ctx, NameDiscovery, s.discoveryPath, s.reloadDiscovery); err != nil {
		return fmt.Errorf("load discovery: %w", err)
	}
	return nil
}

func (s *Service) loadOne(
	ctx context.Context,
	name, fallbackPath string,
	reload func([]byte) error,
) error {
	row, err := s.dbgen.ConfigFilesServiceGetConfigFile(ctx, name)
	if err != nil {
		return err
	}

	content := strings.TrimSpace(row.Content)
	if content == "" {
		// First boot: seed from disk file.
		data, err := os.ReadFile(fallbackPath)
		if err != nil {
			return fmt.Errorf("read fallback %q: %w", fallbackPath, err)
		}
		if err := reload(data); err != nil {
			return fmt.Errorf("validate fallback %q: %w", fallbackPath, err)
		}
		seeded, err := s.dbgen.ConfigFilesServiceUpdateConfigFile(ctx, dbgen.ConfigFilesServiceUpdateConfigFileParams{
			Name:    name,
			Content: string(data),
		})
		if err != nil {
			return fmt.Errorf("seed %q to db: %w", name, err)
		}
		s.markApplied(name, seeded.UpdatedAt)
		logger.Info("seeded config from file", logger.KV{"config": name, "path": fallbackPath})
		return nil
	}

	if err := reload([]byte(content)); err != nil {
		return fmt.Errorf("reload %q from db: %w", name, err)
	}
	s.markApplied(name, row.UpdatedAt)
	return nil
}

// GetConfig returns the current raw YAML content for a named config.
func (s *Service) GetConfig(ctx context.Context, name string) (dbgen.ConfigFile, error) {
	return s.dbgen.ConfigFilesServiceGetConfigFile(ctx, name)
}

// ValidateOnly parses YAML without saving or applying. Returns an error if invalid.
func (s *Service) ValidateOnly(_ context.Context, name, content string) error {
	validate, _, err := s.configFuncs(name)
	if err != nil {
		return err
	}
	return validate([]byte(content))
}

// configFuncs returns a parse-only validator and an apply (hot-reload) function
// for a named config. Validation must not mutate any live state.
func (s *Service) configFuncs(name string) (func([]byte) error, func([]byte) error, error) {
	switch name {
	case NameRestorePresets:
		return s.validateRestorePresets, s.reloadRestorePresets, nil
	case NameDiscovery:
		return s.validateDiscovery, s.reloadDiscovery, nil
	}
	return nil, nil, fmt.Errorf("unknown config: %s", name)
}

// ValidateAndSave validates YAML, backs up current content, saves new content,
// and only then hot-reloads. Apply happens after a successful save so the
// in-memory state never diverges from what is persisted.
func (s *Service) ValidateAndSave(ctx context.Context, name, content string) error {
	data := []byte(content)

	validate, apply, err := s.configFuncs(name)
	if err != nil {
		return err
	}

	if err := validate(data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Backup current content if non-empty.
	current, err := s.dbgen.ConfigFilesServiceGetConfigFile(ctx, name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(current.Content) != "" {
		if _, err := s.dbgen.ConfigFilesServiceCreateConfigFileBackup(ctx, dbgen.ConfigFilesServiceCreateConfigFileBackupParams{
			ConfigName: name,
			Content:    current.Content,
		}); err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
		if err := s.pruneBackups(ctx, name); err != nil {
			logger.Error("prune config backups", logger.KV{"config": name, "error": err})
		}
	}

	saved, err := s.dbgen.ConfigFilesServiceUpdateConfigFile(ctx, dbgen.ConfigFilesServiceUpdateConfigFileParams{
		Name:    name,
		Content: content,
	})
	if err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	if err := apply(data); err != nil {
		return fmt.Errorf("apply saved config: %w", err)
	}
	s.markApplied(name, saved.UpdatedAt)

	return nil
}

// Watch polls config_files and hot-reloads configs saved by other pods.
// The pod that handles a save applies it immediately (ValidateAndSave);
// every other pod picks the change up here within one interval.
func (s *Service) Watch(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reloadChanged(ctx)
		}
	}
}

func (s *Service) reloadChanged(ctx context.Context) {
	for _, name := range []string{NameRestorePresets, NameDiscovery} {
		row, err := s.dbgen.ConfigFilesServiceGetConfigFile(ctx, name)
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("config watch: fetch failed", logger.KV{
					"config": name, "error": err.Error(),
				})
			}
			continue
		}

		if !row.UpdatedAt.After(s.appliedVersion(name)) {
			continue
		}

		_, apply, err := s.configFuncs(name)
		if err != nil {
			continue
		}
		if err := apply([]byte(row.Content)); err != nil {
			// Content was validated on save; if it still fails here, record
			// the version anyway so the error is logged once, not every tick.
			logger.Error("config watch: apply failed", logger.KV{
				"config": name, "error": err.Error(),
			})
			s.markApplied(name, row.UpdatedAt)
			continue
		}
		s.markApplied(name, row.UpdatedAt)
		logger.Info("config hot-reloaded from db", logger.KV{
			"config": name, "updated_at": row.UpdatedAt,
		})
	}
}

// ListBackups returns the last 10 backups for a config.
func (s *Service) ListBackups(ctx context.Context, name string) ([]dbgen.ConfigFileBackup, error) {
	return s.dbgen.ConfigFilesServiceListConfigFileBackups(ctx, name)
}

// RestoreBackup restores config from a backup: validates, saves current as new backup, then applies.
func (s *Service) RestoreBackup(ctx context.Context, backupID string) error {
	id, err := parseUUID(backupID)
	if err != nil {
		return fmt.Errorf("invalid backup id: %w", err)
	}

	backup, err := s.dbgen.ConfigFilesServiceGetConfigFileBackup(ctx, id)
	if err != nil {
		return err
	}

	return s.ValidateAndSave(ctx, backup.ConfigName, backup.Content)
}

func (s *Service) pruneBackups(ctx context.Context, name string) error {
	backups, err := s.dbgen.ConfigFilesServiceListConfigFileBackups(ctx, name)
	if err != nil {
		return err
	}
	// Keep the 10 newest; ListConfigFileBackups already returns DESC order.
	for i := 10; i < len(backups); i++ {
		if err := s.dbgen.ConfigFilesServiceDeleteConfigFileBackup(ctx, backups[i].ID); err != nil {
			logger.Error("delete old config backup", logger.KV{"id": backups[i].ID, "error": err})
		}
	}
	return nil
}

func (s *Service) validateRestorePresets(data []byte) error {
	return restorations.ValidatePresetsFromBytes(data)
}

func (s *Service) reloadRestorePresets(data []byte) error {
	return restorations.LoadPresetsFromBytes(data)
}

func (s *Service) validateDiscovery(data []byte) error {
	_, err := discovery.LoadConfigFromBytes(data)
	return err
}

func (s *Service) reloadDiscovery(data []byte) error {
	cfg, err := discovery.LoadConfigFromBytes(data)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.lastDiscoveryCfg = cfg
	svc := s.discoveryService
	s.mu.Unlock()

	if svc != nil {
		svc.ReloadConfig(cfg)
	}
	return nil
}
