package restorations

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type restorePresetsFile struct {
	AdminGroup string              `yaml:"admin_group"`
	Presets    []restorePresetYAML `yaml:"presets"`
}

type restorePresetYAML struct {
	ID          string              `yaml:"id"`
	Title       string              `yaml:"title"`
	Description string              `yaml:"description,omitempty"`
	RBAC        RestorePresetRBAC   `yaml:"rbac"`
	Source      restoreEndpointYAML `yaml:"source"`
	Targets     []restoreTargetYAML `yaml:"targets"`
}

type restoreEndpointYAML struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Owner    string `yaml:"owner,omitempty"`
}

type restoreTargetYAML struct {
	Environment string `yaml:"environment"`
	restoreEndpointYAML `yaml:",inline"`
}

var (
	// restorePresetsMu guards restorePresets and restoreAdminGroup:
	// they are hot-reloaded via LoadPresetsFromBytes while HTTP handlers read them.
	restorePresetsMu     sync.RWMutex
	restorePresets       []RestorePreset
	restoreAdminGroup    string
	// restorePresetAliases maps legacy preset IDs to current ones for backwards compatibility.
	restorePresetAliases = map[string]string{}
)

// LoadPresetsFromBytes parses and hot-reloads restore presets from raw YAML bytes.
func LoadPresetsFromBytes(data []byte) error {
	presets, adminGroup, err := parsePresetsYAML(data)
	if err != nil {
		return err
	}
	setRestorePresets(presets, adminGroup)
	return nil
}

// LoadPresets reads restore presets from a YAML file.
func LoadPresets(path string) error {
	if strings.TrimSpace(path) == "" {
		path = "configs/restore-presets.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read restore presets %q: %w", path, err)
	}

	presets, adminGroup, err := parsePresetsYAML(data)
	if err != nil {
		return fmt.Errorf("parse restore presets %q: %w", path, err)
	}

	setRestorePresets(presets, adminGroup)
	return nil
}

func setRestorePresets(presets []RestorePreset, adminGroup string) {
	restorePresetsMu.Lock()
	defer restorePresetsMu.Unlock()
	restorePresets = presets
	restoreAdminGroup = adminGroup
}

func parsePresetsYAML(data []byte) ([]RestorePreset, string, error) {
	var file restorePresetsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, "", err
	}

	if len(file.Presets) == 0 {
		return nil, "", fmt.Errorf("presets list is empty")
	}

	seenPresetIDs := map[string]struct{}{}
	presets := make([]RestorePreset, 0, len(file.Presets))

	for i, item := range file.Presets {
		preset, err := normalizePresetYAML(item, i)
		if err != nil {
			return nil, "", err
		}
		if _, ok := seenPresetIDs[preset.ID]; ok {
			return nil, "", fmt.Errorf("presets[%d]: duplicate id %q", i, preset.ID)
		}
		seenPresetIDs[preset.ID] = struct{}{}
		presets = append(presets, preset)
	}

	return presets, strings.TrimSpace(file.AdminGroup), nil
}

// ValidatePresetsFromBytes parses restore presets YAML without applying it.
func ValidatePresetsFromBytes(data []byte) error {
	_, _, err := parsePresetsYAML(data)
	return err
}

func normalizePresetYAML(item restorePresetYAML, index int) (RestorePreset, error) {
	prefix := fmt.Sprintf("presets[%d]", index)

	if strings.TrimSpace(item.ID) == "" {
		return RestorePreset{}, fmt.Errorf("%s: id is required", prefix)
	}
	if strings.TrimSpace(item.Title) == "" {
		return RestorePreset{}, fmt.Errorf("%s: title is required", prefix)
	}

	source, err := normalizeEndpointYAML(item.Source, prefix+".source")
	if err != nil {
		return RestorePreset{}, err
	}
	if len(item.Targets) == 0 {
		return RestorePreset{}, fmt.Errorf("%s: at least one target is required", prefix)
	}

	seenEnvironments := map[string]struct{}{}
	targets := make([]RestoreTarget, 0, len(item.Targets))
	for i, target := range item.Targets {
		endpoint, err := normalizeTargetYAML(
			target, fmt.Sprintf("%s.targets[%d]", prefix, i), seenEnvironments,
		)
		if err != nil {
			return RestorePreset{}, err
		}
		targets = append(targets, endpoint)
	}

	return RestorePreset{
		ID:          strings.TrimSpace(item.ID),
		Title:       strings.TrimSpace(item.Title),
		Description: strings.TrimSpace(item.Description),
		RBAC:        item.RBAC,
		Source:      source,
		Targets:     targets,
	}, nil
}

func normalizeTargetYAML(
	item restoreTargetYAML, prefix string, seenEnvironments map[string]struct{},
) (RestoreTarget, error) {
	environment := normalizeEnvironment(item.Environment)
	if environment == "" {
		return RestoreTarget{}, fmt.Errorf("%s: environment is required", prefix)
	}
	if _, ok := seenEnvironments[environment]; ok {
		return RestoreTarget{}, fmt.Errorf("%s: duplicate environment %q", prefix, environment)
	}
	seenEnvironments[environment] = struct{}{}

	endpoint, err := normalizeEndpointYAML(item.restoreEndpointYAML, prefix)
	if err != nil {
		return RestoreTarget{}, err
	}

	return RestoreTarget{
		Environment:     environment,
		RestoreEndpoint: endpoint,
	}, nil
}

func normalizeEndpointYAML(
	item restoreEndpointYAML, prefix string,
) (RestoreEndpoint, error) {
	host := strings.TrimSpace(item.Host)
	database := strings.TrimSpace(item.Database)
	owner := strings.TrimSpace(item.Owner)

	if host == "" {
		return RestoreEndpoint{}, fmt.Errorf("%s: host is required", prefix)
	}
	if item.Port < 1 || item.Port > 65535 {
		return RestoreEndpoint{}, fmt.Errorf("%s: port must be between 1 and 65535", prefix)
	}
	if database == "" {
		return RestoreEndpoint{}, fmt.Errorf("%s: database is required", prefix)
	}

	return RestoreEndpoint{
		Host:     host,
		Port:     item.Port,
		Database: database,
		Owner:    owner,
		PbwName:  pbwName(host, item.Port, database),
	}, nil
}

func getRestorePresets() []RestorePreset {
	restorePresetsMu.RLock()
	defer restorePresetsMu.RUnlock()
	return restorePresets
}

// GetPresets returns loaded restore presets.
func GetPresets() []RestorePreset {
	return getRestorePresets()
}

// GetAdminGroup returns the global admin group from presets YAML.
func GetAdminGroup() string {
	restorePresetsMu.RLock()
	defer restorePresetsMu.RUnlock()
	return restoreAdminGroup
}
