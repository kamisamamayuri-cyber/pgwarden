package discovery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/cron"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/backups"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/databases"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
)

var ErrDiscoveryAlreadyRunning = errors.New("discovery is already running")

type Service struct {
	cfg              Config
	cfgMu            sync.RWMutex
	dbgen            *dbgen.Queries
	databasesService *databases.Service
	backupsService   *backups.Service
	pgUser           string
	pgPassword       string
	cr               *cron.Cron
	scheduledEnabled bool
	cronExpression   string
	timeZone         string
	runMu            sync.Mutex
	runActive        bool
}

// ReloadConfig hot-reloads the discovery configuration without a pod restart.
func (s *Service) ReloadConfig(cfg Config) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.cfg = cfg
}

func (s *Service) getConfig() Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

// Active-run detection window is defined in discovery.sql
// (DiscoveryServiceHasActiveRun, INTERVAL '30 minutes').
const (
	discoveryTCPWorkers           = 128
	discoveryTCPDialTimeout       = 150 * time.Millisecond
	discoveryPostgresProbes       = 32
	discoveryPostgresProbeTimeout = 2 * time.Second
	discoveryProbeFailureLogLimit = 50
)

func discoveryProgressStep(total int) int {
	if total <= 500 {
		return 100
	}
	step := total / 50
	if step < 500 {
		return 500
	}
	return step
}

type Result struct {
	RunID            uuid.UUID
	ClustersScanned  int
	DatabasesSeen    int
	DatabasesCreated int
	BackupsCreated   int
	SkippedExisting  int
	Errors           int
}

type remoteDatabase struct {
	Name      string
	Version   string
	SizeBytes int64
}

type Event struct {
	RunID        uuid.UUID
	Level        string
	Event        string
	Host         string
	Port         int
	DatabaseName string
	Message      string
}

type PaginateEventsParams struct {
	Page           int
	Limit          int
	LevelFilter    sql.NullString
	EventFilter    sql.NullString
	HostFilter     sql.NullString
	PortFilter     sql.NullInt32
	DatabaseFilter sql.NullString
}

type PaginateRunsParams = PaginateEventsParams

func New(
	cfg Config,
	dbgen *dbgen.Queries,
	databasesService *databases.Service,
	backupsService *backups.Service,
	pgUser string,
	pgPassword string,
	cr *cron.Cron,
	scheduledEnabled bool,
	cronExpression string,
	timeZone string,
) *Service {
	return &Service{
		cfg:              cfg,
		dbgen:            dbgen,
		databasesService: databasesService,
		backupsService:   backupsService,
		pgUser:           pgUser,
		pgPassword:       pgPassword,
		cr:               cr,
		scheduledEnabled: scheduledEnabled,
		cronExpression:   cronExpression,
		timeZone:         timeZone,
	}
}

func (s *Service) Schedule() error {
	jobID := uuid.MustParse("7da02632-6c1d-4b31-a5de-170bfbdbf9e4")
	if !s.scheduledEnabled {
		logger.Info("discovery schedule disabled (PBW_DISCOVERY_SCHEDULED_ENABLED=false)")
		return s.cr.RemoveJob(jobID)
	}

	return s.cr.UpsertJob(jobID, s.timeZone, s.cronExpression, func() {
		result, err := s.Run(context.Background())
		if err != nil {
			if errors.Is(err, ErrDiscoveryAlreadyRunning) {
				logger.Info("scheduled discovery skipped: already running")
				return
			}
			logger.Error("scheduled discovery failed", logger.KV{"error": err})
			return
		}
		logger.Info("scheduled discovery completed", logger.KV{
			"run_id":            result.RunID.String(),
			"clusters_scanned":  result.ClustersScanned,
			"databases_seen":    result.DatabasesSeen,
			"databases_created": result.DatabasesCreated,
			"backups_created":   result.BackupsCreated,
			"skipped_existing":  result.SkippedExisting,
			"errors":            result.Errors,
		})
	})
}

func (s *Service) Running(ctx context.Context) (bool, error) {
	s.runMu.Lock()
	inMemory := s.runActive
	s.runMu.Unlock()
	if !inMemory {
		return false, nil
	}

	dbActive, err := s.hasActiveRunInDB(ctx)
	if err != nil {
		return inMemory, err
	}
	if dbActive {
		return true, nil
	}

	s.runMu.Lock()
	s.runActive = false
	s.runMu.Unlock()
	logger.Warn("discovery in-memory lock recovered: no active run in DB")
	return false, nil
}

func (s *Service) Run(ctx context.Context) (Result, error) {
	if s.pgPassword == "" {
		return Result{}, fmt.Errorf("PBW_ENCRYPTION_KEY is required as pgwbackup password")
	}

	if err := s.beginRun(ctx); err != nil {
		return Result{}, err
	}
	defer s.endRun()

	return s.runLocked(ctx)
}

func (s *Service) beginRun(ctx context.Context) error {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	if s.runActive {
		dbActive, err := s.hasActiveRunInDB(ctx)
		if err != nil {
			return err
		}
		if dbActive {
			return ErrDiscoveryAlreadyRunning
		}
		logger.Warn("discovery in-memory lock recovered before new run")
		s.runActive = false
	}

	dbActive, err := s.hasActiveRunInDB(ctx)
	if err != nil {
		return err
	}
	if dbActive {
		return ErrDiscoveryAlreadyRunning
	}

	s.runActive = true
	return nil
}

func (s *Service) endRun() {
	s.runMu.Lock()
	s.runActive = false
	s.runMu.Unlock()
}

func (s *Service) hasActiveRunInDB(ctx context.Context) (bool, error) {
	active, err := s.dbgen.DiscoveryServiceHasActiveRun(ctx)
	if err != nil {
		return false, err
	}
	return active, nil
}

func (s *Service) runLocked(ctx context.Context) (Result, error) {
	cfg := s.getConfig()
	result := Result{RunID: uuid.New()}
	s.logEvent(ctx, Event{
		RunID:   result.RunID,
		Level:   "info",
		Event:   "scan_started",
		Host:    "*",
		Message: "Discovery scan started",
	})
	type hostEntry struct {
		host          HostConfig
		noReplica     bool
	}
	var allHosts []hostEntry
	for _, h := range cfg.Hosts {
		allHosts = append(allHosts, hostEntry{host: h, noReplica: false})
	}
	for _, h := range cfg.HostsNoReplica {
		allHosts = append(allHosts, hostEntry{host: h, noReplica: true})
	}

	for _, entry := range allHosts {
		host := entry.host
		clusters, err := s.discoverClusters(ctx, result.RunID, host)
		if err != nil {
			result.Errors++
			s.logEvent(ctx, Event{
				RunID:   result.RunID,
				Level:   "error",
				Event:   "error",
				Host:    host.Name,
				Message: fmt.Sprintf("Discover ports failed: %v", err),
			})
			continue
		}
		for _, cluster := range clusters {
			result.ClustersScanned++
			s.logEvent(ctx, Event{
				RunID:   result.RunID,
				Level:   "info",
				Event:   "cluster_list_started",
				Host:    host.Name,
				Port:    cluster.Port,
				Message: "Listing databases in PostgreSQL cluster",
			})
			dbs, err := s.listDatabases(ctx, host, cluster)
			if err != nil {
				result.Errors++
				s.logEvent(ctx, Event{
					RunID:   result.RunID,
					Level:   "error",
					Event:   "error",
					Host:    host.Name,
					Port:    cluster.Port,
					Message: fmt.Sprintf("List databases failed: %v", err),
				})
				continue
			}
			s.logEvent(ctx, Event{
				RunID: result.RunID,
				Level: "info",
				Event: "cluster_list_finished",
				Host:  host.Name,
				Port:  cluster.Port,
				Message: fmt.Sprintf(
					"Database list received: %d databases before exclusions",
					len(dbs),
				),
			})
			maxSizeBytes, _ := cfg.Defaults.MaxDBSizeBytes()
			for _, db := range dbs {
				if s.isExcluded(db.Name, host, cluster) {
					continue
				}
				if maxSizeBytes > 0 && db.SizeBytes > maxSizeBytes {
					s.logEvent(ctx, Event{
						RunID:        result.RunID,
						Level:        "info",
						Event:        "database_skipped_size",
						Host:         host.Name,
						Port:         cluster.Port,
						DatabaseName: db.Name,
						Message: fmt.Sprintf(
							"Database skipped: size %d MB exceeds max_db_size limit %d MB",
							db.SizeBytes/1024/1024,
							maxSizeBytes/1024/1024,
						),
					})
					continue
				}
				result.DatabasesSeen++
				s.logEvent(ctx, Event{
					RunID:        result.RunID,
					Level:        "info",
					Event:        "database_found",
					Host:         host.Name,
					Port:         cluster.Port,
					DatabaseName: db.Name,
					Message:      "Database found",
				})
				s.logEvent(ctx, Event{
					RunID:        result.RunID,
					Level:        "info",
					Event:        "database_register_started",
					Host:         host.Name,
					Port:         cluster.Port,
					DatabaseName: db.Name,
					Message:      "Checking metadata DB and creating missing records",
				})
				createdDB, createdBackup, skipped, err := s.ensureBackup(ctx, host, cluster, db, entry.noReplica)
				if err != nil {
					result.Errors++
					s.logEvent(ctx, Event{
						RunID:        result.RunID,
						Level:        "error",
						Event:        "error",
						Host:         host.Name,
						Port:         cluster.Port,
						DatabaseName: db.Name,
						Message:      fmt.Sprintf("Add database/backup failed: %v", err),
					})
					continue
				}
				if createdDB {
					result.DatabasesCreated++
					s.logEvent(ctx, Event{
						RunID:        result.RunID,
						Level:        "info",
						Event:        "database_created",
						Host:         host.Name,
						Port:         cluster.Port,
						DatabaseName: db.Name,
						Message:      "Database registered",
					})
				}
				if createdBackup {
					result.BackupsCreated++
					s.logEvent(ctx, Event{
						RunID:        result.RunID,
						Level:        "info",
						Event:        "backup_created",
						Host:         host.Name,
						Port:         cluster.Port,
						DatabaseName: db.Name,
						Message:      "Backup job created",
					})
				}
				if skipped {
					result.SkippedExisting++
					s.logEvent(ctx, Event{
						RunID:        result.RunID,
						Level:        "info",
						Event:        "skipped_existing",
						Host:         host.Name,
						Port:         cluster.Port,
						DatabaseName: db.Name,
						Message:      "Backup job already exists",
					})
				}
			}
		}
	}
	s.logEvent(ctx, Event{
		RunID: result.RunID,
		Level: "info",
		Event: "scan_finished",
		Host:  "*",
		Message: fmt.Sprintf(
			"Discovery scan finished: ports=%d databases=%d created_databases=%d created_backups=%d skipped=%d errors=%d",
			result.ClustersScanned,
			result.DatabasesSeen,
			result.DatabasesCreated,
			result.BackupsCreated,
			result.SkippedExisting,
			result.Errors,
		),
	})

	return result, nil
}

func (s *Service) PaginateEvents(
	ctx context.Context, params PaginateEventsParams,
) (paginateutil.PaginateResponse, []dbgen.DiscoveryEvent, error) {
	page := max(params.Page, 1)
	limit := min(max(params.Limit, 1), 100)

	count, err := s.dbgen.DiscoveryServicePaginateEventsCount(
		ctx, dbgen.DiscoveryServicePaginateEventsCountParams{
			Level:        params.LevelFilter,
			Event:        params.EventFilter,
			Host:         params.HostFilter,
			Port:         params.PortFilter,
			DatabaseName: params.DatabaseFilter,
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	paginateParams := paginateutil.PaginateParams{Page: page, Limit: limit}
	offset := paginateutil.CreateOffsetFromParams(paginateParams)
	events, err := s.dbgen.DiscoveryServicePaginateEvents(
		ctx, dbgen.DiscoveryServicePaginateEventsParams{
			Level:        params.LevelFilter,
			Event:        params.EventFilter,
			Host:         params.HostFilter,
			Port:         params.PortFilter,
			DatabaseName: params.DatabaseFilter,
			Limit:        int32(limit),
			Offset:       int32(offset),
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	return paginateutil.CreatePaginateResponse(paginateParams, int(count)), events, nil
}

func (s *Service) PaginateRuns(
	ctx context.Context, params PaginateRunsParams,
) (paginateutil.PaginateResponse, []dbgen.DiscoveryServicePaginateRunsRow, error) {
	page := max(params.Page, 1)
	limit := min(max(params.Limit, 1), 100)

	count, err := s.dbgen.DiscoveryServicePaginateRunsCount(
		ctx, dbgen.DiscoveryServicePaginateRunsCountParams{
			Level:        params.LevelFilter,
			Event:        params.EventFilter,
			Host:         params.HostFilter,
			Port:         params.PortFilter,
			DatabaseName: params.DatabaseFilter,
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	paginateParams := paginateutil.PaginateParams{Page: page, Limit: limit}
	offset := paginateutil.CreateOffsetFromParams(paginateParams)
	runs, err := s.dbgen.DiscoveryServicePaginateRuns(
		ctx, dbgen.DiscoveryServicePaginateRunsParams{
			Level:        params.LevelFilter,
			Event:        params.EventFilter,
			Host:         params.HostFilter,
			Port:         params.PortFilter,
			DatabaseName: params.DatabaseFilter,
			Limit:        int32(limit),
			Offset:       int32(offset),
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	return paginateutil.CreatePaginateResponse(paginateParams, int(count)), runs, nil
}

func (s *Service) ListRunEvents(
	ctx context.Context,
	runID uuid.UUID,
	reportOnly bool,
) ([]dbgen.DiscoveryEvent, error) {
	return s.dbgen.DiscoveryServiceListRunEvents(
		ctx,
		dbgen.DiscoveryServiceListRunEventsParams{
			RunID:      runID,
			ReportOnly: reportOnly,
		},
	)
}

func (s *Service) logEvent(ctx context.Context, event Event) {
	var port sql.NullInt32
	if event.Port > 0 {
		port = sql.NullInt32{Int32: int32(event.Port), Valid: true}
	}
	var databaseName sql.NullString
	if event.DatabaseName != "" {
		databaseName = sql.NullString{String: event.DatabaseName, Valid: true}
	}

	_ = s.dbgen.DiscoveryServiceCreateEvent(ctx, dbgen.DiscoveryServiceCreateEventParams{
		RunID:        event.RunID,
		Level:        event.Level,
		Event:        event.Event,
		Host:         event.Host,
		Port:         port,
		DatabaseName: databaseName,
		Message:      event.Message,
	})
}

func (s *Service) discoverClusters(
	ctx context.Context,
	runID uuid.UUID,
	host HostConfig,
) ([]ClusterConfig, error) {
	if len(host.Clusters) > 0 {
		s.logEvent(ctx, Event{
			RunID:   runID,
			Level:   "info",
			Event:   "host_scan_started",
			Host:    host.Name,
			Message: fmt.Sprintf("Using %d configured PostgreSQL ports", len(host.Clusters)),
		})
		return host.Clusters, nil
	}

	ports, err := s.getConfig().scanPorts()
	if err != nil {
		return nil, err
	}
	progressStep := discoveryProgressStep(len(ports))
	s.logEvent(ctx, Event{
		RunID:   runID,
		Level:   "info",
		Event:   "host_scan_started",
		Host:    host.Name,
		Message: fmt.Sprintf(
			"Phase 1: fast TCP scan of %d ports on %s",
			len(ports),
			host.connectionAddress(),
		),
	})

	openPorts := s.scanOpenTCPPorts(ctx, runID, host, ports, progressStep)
	s.logEvent(ctx, Event{
		RunID: runID,
		Level: "info",
		Event: "tcp_scan_finished",
		Host:  host.Name,
		Message: fmt.Sprintf(
			"TCP scan finished: checked=%d open=%d ports=%s",
			len(ports),
			len(openPorts),
			formatPortList(openPorts),
		),
	})
	if len(openPorts) == 0 {
		s.logEvent(ctx, Event{
			RunID:   runID,
			Level:   "error",
			Event:   "host_scan_finished",
			Host:    host.Name,
			Message: fmt.Sprintf("No open TCP ports in range (%d checked)", len(ports)),
		})
		return nil, fmt.Errorf("no open TCP ports on %s", host.Name)
	}

	s.logEvent(ctx, Event{
		RunID: runID,
		Level: "info",
		Event: "postgres_probe_started",
		Host:  host.Name,
		Message: fmt.Sprintf(
			"Phase 2: PostgreSQL probe on %d open ports",
			len(openPorts),
		),
	})

	clusters, probeFailures := s.probePostgresPorts(ctx, runID, host, openPorts)
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Port < clusters[j].Port
	})
	s.logEvent(ctx, Event{
		RunID: runID,
		Level: "info",
		Event: "postgres_probe_finished",
		Host:  host.Name,
		Message: fmt.Sprintf(
			"PostgreSQL probe finished: open=%d postgres=%d not_postgres=%d",
			len(openPorts),
			len(clusters),
			len(probeFailures),
		),
	})
	if len(clusters) == 0 {
		s.logEvent(ctx, Event{
			RunID:   runID,
			Level:   "error",
			Event:   "host_scan_finished",
			Host:    host.Name,
			Message: fmt.Sprintf(
				"No PostgreSQL ports among %d open TCP ports",
				len(openPorts),
			),
		})
		return nil, fmt.Errorf("no PostgreSQL ports discovered on %s", host.Name)
	}
	s.logEvent(ctx, Event{
		RunID: runID,
		Level: "info",
		Event: "host_scan_finished",
		Host:  host.Name,
		Message: fmt.Sprintf(
			"Host scan finished: tcp_checked=%d tcp_open=%d postgres=%d",
			len(ports),
			len(openPorts),
			len(clusters),
		),
	})

	return clusters, nil
}

func (s *Service) scanOpenTCPPorts(
	ctx context.Context,
	runID uuid.UUID,
	host HostConfig,
	ports []int,
	progressStep int,
) []int {
	workerCount := discoveryTCPWorkers
	if len(ports) < workerCount {
		workerCount = len(ports)
	}

	jobs := make(chan int, workerCount)
	type tcpResult struct {
		port  int
		open  bool
	}
	results := make(chan tcpResult, workerCount)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				results <- tcpResult{
					port: port,
					open: s.isTCPPortOpen(ctx, host, port),
				}
			}
		}()
	}

	go func() {
		for _, port := range ports {
			jobs <- port
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	openPorts := make([]int, 0, 32)
	checked := 0
	for result := range results {
		checked++
		if result.open {
			openPorts = append(openPorts, result.port)
		}
		if checked%progressStep == 0 || checked == len(ports) {
			s.logEvent(ctx, Event{
				RunID: runID,
				Level: "info",
				Event: "scan_progress",
				Host:  host.Name,
				Message: fmt.Sprintf(
					"TCP scan: checked %d/%d, open=%d",
					checked,
					len(ports),
					len(openPorts),
				),
			})
		}
	}
	sort.Ints(openPorts)
	return openPorts
}

type postgresProbeFailure struct {
	port    int
	message string
}

func (s *Service) probePostgresPorts(
	ctx context.Context,
	runID uuid.UUID,
	host HostConfig,
	openPorts []int,
) ([]ClusterConfig, []postgresProbeFailure) {
	type probeResult struct {
		cluster  ClusterConfig
		ok       bool
		port     int
		error    string
		rawError error
	}

	jobs := make(chan int, len(openPorts))
	results := make(chan probeResult, len(openPorts))
	postgresSem := make(chan struct{}, discoveryPostgresProbes)

	workerCount := discoveryPostgresProbes
	if len(openPorts) < workerCount {
		workerCount = len(openPorts)
	}

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				version, err := s.probePostgresOnPort(ctx, host, port, postgresSem)
				results <- probeResult{
					port:     port,
					ok:       err == nil,
					cluster:  ClusterConfig{Port: port, PGVersion: version},
					error:    probeErrText(err),
					rawError: err,
				}
			}
		}()
	}

	go func() {
		for _, port := range openPorts {
			jobs <- port
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	clusters := make([]ClusterConfig, 0, len(openPorts))
	failures := make([]postgresProbeFailure, 0)
	checked := 0
	progressStep := discoveryProgressStep(len(openPorts))
	logEachFailure := len(openPorts) <= discoveryProbeFailureLogLimit
	for result := range results {
		checked++
		if result.ok {
			clusters = append(clusters, result.cluster)
			s.logEvent(ctx, Event{
				RunID:   runID,
				Level:   "info",
				Event:   "port_found",
				Host:    host.Name,
				Port:    result.port,
				Message: fmt.Sprintf("PostgreSQL probe succeeded, version %s", result.cluster.PGVersion),
			})
			continue
		}
		failures = append(failures, postgresProbeFailure{
			port:    result.port,
			message: result.error,
		})
		if logEachFailure {
			if isPostgresAuthError(result.rawError) {
				s.logEvent(ctx, Event{
					RunID:   runID,
					Level:   "warn",
					Event:   "postgres_auth_failed",
					Host:    host.Name,
					Port:    result.port,
					Message: fmt.Sprintf(
						"PostgreSQL detected but probe user has no access (grant SELECT to probe user): %s",
						result.error,
					),
				})
			} else {
				s.logEvent(ctx, Event{
					RunID:   runID,
					Level:   "info",
					Event:   "port_not_postgres",
					Host:    host.Name,
					Port:    result.port,
					Message: fmt.Sprintf(
						"Open TCP port is not PostgreSQL (expected during scan): %s",
						result.error,
					),
				})
			}
		}
		if checked%progressStep == 0 || checked == len(openPorts) {
			s.logEvent(ctx, Event{
				RunID: runID,
				Level: "info",
				Event: "scan_progress",
				Host:  host.Name,
				Message: fmt.Sprintf(
					"PostgreSQL probe: checked %d/%d, postgres=%d",
					checked,
					len(openPorts),
					len(clusters),
				),
			})
		}
	}
	if !logEachFailure && len(failures) > 0 {
		s.logEvent(ctx, Event{
			RunID: runID,
			Level: "info",
			Event: "scan_progress",
			Host:  host.Name,
			Message: fmt.Sprintf(
				"Skipped per-port logs for %d non-PostgreSQL open ports (too many to list)",
				len(failures),
			),
		})
	}
	return clusters, failures
}

func probeErrText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// isPostgresAuthError returns true when the probe error indicates the port IS
// running PostgreSQL but rejected our credentials (pq codes 28000 / 28P01).
func isPostgresAuthError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "28000" || pqErr.Code == "28P01"
	}
	return false
}

func (s *Service) isTCPPortOpen(ctx context.Context, host HostConfig, port int) bool {
	address := net.JoinHostPort(host.connectionAddress(), strconv.Itoa(port))
	dialer := net.Dialer{Timeout: discoveryTCPDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (s *Service) probePostgresOnPort(
	ctx context.Context,
	host HostConfig,
	port int,
	postgresSem chan struct{},
) (string, error) {
	select {
	case postgresSem <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-postgresSem }()

	probeCtx, cancel := context.WithTimeout(ctx, discoveryPostgresProbeTimeout+500*time.Millisecond)
	defer cancel()

	type probeResult struct {
		version string
		err     error
	}
	ch := make(chan probeResult, 1)
	go func() {
		version, err := s.probePostgresOnPortInner(probeCtx, host, port)
		ch <- probeResult{version: version, err: err}
	}()

	select {
	case res := <-ch:
		return res.version, res.err
	case <-probeCtx.Done():
		return "", fmt.Errorf(
			"postgres probe timed out on port %d after %s",
			port,
			discoveryPostgresProbeTimeout+500*time.Millisecond,
		)
	}
}

func (s *Service) probePostgresOnPortInner(
	ctx context.Context,
	host HostConfig,
	port int,
) (string, error) {
	db, err := sql.Open("postgres", s.probeConnectionString(host.connectionAddress(), port))
	if err != nil {
		return "", err
	}
	defer db.Close()
	db.SetConnMaxLifetime(5 * time.Second)
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		return "", err
	}

	var versionNum string
	err = db.QueryRowContext(
		ctx,
		"SELECT current_setting('server_version_num')",
	).Scan(&versionNum)
	if err != nil {
		return "", err
	}
	return majorVersion(versionNum)
}

func (s *Service) listDatabases(
	ctx context.Context,
	host HostConfig,
	cluster ClusterConfig,
) ([]remoteDatabase, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", s.connectionString(host.connectionAddress(), cluster.Port, "postgres"))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(time.Minute)

	rows, err := db.QueryContext(queryCtx, `
SELECT datname, current_setting('server_version_num'), pg_database_size(datname)
FROM pg_database
WHERE datistemplate = false
ORDER BY datname`)
	if err != nil {
		return nil, fmt.Errorf("list databases on %s:%d: %w", host.Name, cluster.Port, err)
	}
	defer rows.Close()

	var res []remoteDatabase
	for rows.Next() {
		var db remoteDatabase
		var versionNum string
		if err := rows.Scan(&db.Name, &versionNum, &db.SizeBytes); err != nil {
			return nil, err
		}
		if cluster.PGVersion != "" {
			db.Version = cluster.PGVersion
		} else {
			db.Version, err = majorVersion(versionNum)
			if err != nil {
				return nil, fmt.Errorf(
					"database %s on %s:%d: %w", db.Name, host.Name, cluster.Port, err,
				)
			}
		}
		res = append(res, db)
	}
	return res, rows.Err()
}

func (s *Service) ensureBackup(
	ctx context.Context,
	host HostConfig,
	cluster ClusterConfig,
	db remoteDatabase,
	serializableDeferrable bool,
) (bool, bool, bool, error) {
	opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	name := fmt.Sprintf("%s:%d-%s", host.Name, cluster.Port, db.Name)
	destDir := fmt.Sprintf("/%s/%d/%s", host.Name, cluster.Port, db.Name)

	_, err := s.dbgen.DiscoveryServiceGetBackupByDestDir(opCtx, destDir)
	if err == nil {
		return false, false, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, false, false, err
	}

	databaseID, dbCreated, err := s.ensureDatabase(opCtx, host, cluster, db, name)
	if err != nil {
		return false, false, false, err
	}

	destinationID, err := uuid.Parse(s.getConfig().Defaults.DestinationID)
	if err != nil {
		return false, false, false, err
	}
	_, err = s.backupsService.CreateBackup(opCtx, dbgen.BackupsServiceCreateBackupParams{
		DatabaseID:                 databaseID,
		DestinationID:              uuid.NullUUID{UUID: destinationID, Valid: true},
		IsLocal:                    false,
		Name:                       name,
		CronExpression:             s.randomCron(),
		TimeZone:                   s.getConfig().Defaults.TimeZone,
		IsActive:                   *s.getConfig().Defaults.IsActive,
		DestDir:                    destDir,
		RetentionDays:              s.getConfig().Defaults.RetentionDays,
		OptDataOnly:                false,
		OptSchemaOnly:              false,
		OptClean:                   false,
		OptIfExists:                false,
		OptCreate:                  false,
		OptNoComments:              false,
		OptSerializableDeferrable:  serializableDeferrable,
	})
	if err != nil {
		return dbCreated, false, false, err
	}

	return dbCreated, true, false, nil
}

func (s *Service) ensureDatabase(
	ctx context.Context,
	host HostConfig,
	cluster ClusterConfig,
	db remoteDatabase,
	name string,
) (uuid.UUID, bool, error) {
	existing, err := s.dbgen.DiscoveryServiceGetDatabaseByName(ctx, name)
	if err == nil {
		return existing.ID, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, false, err
	}

	created, err := s.databasesService.CreateDatabase(ctx, dbgen.DatabasesServiceCreateDatabaseParams{
		Name:             name,
		ConnectionString: s.connectionString(host.connectionAddress(), cluster.Port, db.Name),
		PgVersion:        db.Version,
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	return created.ID, true, nil
}

func (s *Service) connectionString(host string, port int, dbName string) string {
	return s.buildConnectionString(host, port, dbName, 0)
}

func (s *Service) probeConnectionString(host string, port int) string {
	connectTimeoutSec := int(discoveryPostgresProbeTimeout / time.Second)
	if connectTimeoutSec < 1 {
		connectTimeoutSec = 1
	}
	return s.buildConnectionString(host, port, "postgres", connectTimeoutSec)
}

func (s *Service) buildConnectionString(
	host string, port int, dbName string, connectTimeoutSec int,
) string {
	u := url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(s.pgUser, s.pgPassword),
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   dbName,
	}
	q := u.Query()
	q.Set("sslmode", s.getConfig().Defaults.ConnectionSSLMode)
	if connectTimeoutSec > 0 {
		q.Set("connect_timeout", strconv.Itoa(connectTimeoutSec))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func formatPortList(ports []int) string {
	if len(ports) == 0 {
		return "[]"
	}
	parts := make([]string, len(ports))
	for i, port := range ports {
		parts[i] = strconv.Itoa(port)
	}
	return strings.Join(parts, ", ")
}

func (s *Service) isExcluded(dbName string, host HostConfig, cluster ClusterConfig) bool {
	excludedDatabases := append([]string{}, s.getConfig().Defaults.ExcludeDatabases...)
	excludedDatabases = append(excludedDatabases, host.ExcludeDatabases...)
	excludedDatabases = append(excludedDatabases, cluster.ExcludeDatabases...)
	for _, excluded := range excludedDatabases {
		if dbName == excluded {
			return true
		}
	}
	return false
}

func (s *Service) randomCron() string {
	cfg := s.getConfig().Defaults

	step := cfg.CronMinuteStep
	if step <= 0 || step > 60 {
		step = 5
	}
	minute := rand.Intn(60/step) * step

	hourRange := cfg.CronHourTo - cfg.CronHourFrom + 1
	if hourRange <= 0 {
		hourRange = 1
	}
	hour := cfg.CronHourFrom + rand.Intn(hourRange)

	return fmt.Sprintf("%d %d * * *", minute, hour)
}

func (h HostConfig) connectionAddress() string {
	if h.ConnectionHost != "" {
		return h.ConnectionHost
	}
	return h.Name
}

// majorVersion maps server_version_num to a pg_dump major version. Servers
// older than 13 are clamped to "13": pg_dump can dump older servers. Garbage
// input is an error instead of a silent fallback that would surface later as
// a confusing dump failure.
func majorVersion(serverVersionNum string) (string, error) {
	n, err := strconv.Atoi(strings.TrimSpace(serverVersionNum))
	if err != nil || n <= 0 {
		return "", fmt.Errorf("unexpected server_version_num %q", serverVersionNum)
	}
	major := n / 10000
	if major < 13 {
		return "13", nil
	}
	return strconv.Itoa(major), nil
}
