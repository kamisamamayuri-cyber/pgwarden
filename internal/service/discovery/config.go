package discovery

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Defaults       DefaultsConfig `yaml:"defaults"`
	Hosts          []HostConfig   `yaml:"hosts"`
	HostsNoReplica []HostConfig   `yaml:"hosts_no_replica"`
}

type DefaultsConfig struct {
	DestinationID     string   `yaml:"destination_id"`
	RetentionDays     int16    `yaml:"retention_days"`
	TimeZone          string   `yaml:"time_zone"`
	IsActive          *bool    `yaml:"is_active"`
	ExcludeDatabases  []string `yaml:"exclude_databases"`
	CronMinuteStep    int      `yaml:"cron_minute_step"`
	CronHourFrom      int      `yaml:"cron_hour_from"`
	CronHourTo        int      `yaml:"cron_hour_to"`
	ConnectionSSLMode string   `yaml:"connection_sslmode"`
	ScanPorts         []string `yaml:"scan_ports"`
	// MaxDBSize is a human-readable size limit (e.g. "10GB", "500MB"). Databases
	// larger than this are skipped by discovery — no record is created in the
	// metadata DB and no backup job is scheduled. Empty or "0" means no limit.
	MaxDBSize string `yaml:"max_db_size"`
}

// MaxDBSizeBytes parses MaxDBSize and returns the limit in bytes.
// Returns 0 when no limit is configured.
func (d DefaultsConfig) MaxDBSizeBytes() (int64, error) {
	s := strings.TrimSpace(d.MaxDBSize)
	if s == "" || s == "0" {
		return 0, nil
	}
	units := []struct {
		suffix string
		mult   int64
	}{
		{"TB", 1 << 40},
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	}
	upper := strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	for _, u := range units {
		if strings.HasSuffix(upper, u.suffix) {
			numStr := upper[:len(upper)-len(u.suffix)]
			n, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil || n < 0 {
				return 0, fmt.Errorf("invalid max_db_size %q", s)
			}
			return n * u.mult, nil
		}
	}
	return 0, fmt.Errorf("invalid max_db_size %q: unrecognized unit (use KB/MB/GB/TB)", s)
}

type HostConfig struct {
	Name             string          `yaml:"name"`
	ConnectionHost   string          `yaml:"connection_host"`
	ExcludeDatabases []string        `yaml:"exclude_databases"`
	Clusters         []ClusterConfig `yaml:"clusters,omitempty"`
}

type ClusterConfig struct {
	Port             int      `yaml:"port"`
	PGVersion        string   `yaml:"pg_version"`
	ExcludeDatabases []string `yaml:"exclude_databases"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.setDefaults()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// LoadConfigFromBytes parses a discovery config from raw YAML bytes.
func LoadConfigFromBytes(data []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg.setDefaults()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) setDefaults() {
	if c.Defaults.TimeZone == "" {
		c.Defaults.TimeZone = "Asia/Yekaterinburg"
	}
	if c.Defaults.RetentionDays == 0 {
		c.Defaults.RetentionDays = 30
	}
	if c.Defaults.IsActive == nil {
		v := true
		c.Defaults.IsActive = &v
	}
	if c.Defaults.CronMinuteStep == 0 {
		c.Defaults.CronMinuteStep = 5
	}
	if c.Defaults.CronHourTo == 0 {
		c.Defaults.CronHourTo = 5
	}
	if c.Defaults.ConnectionSSLMode == "" {
		c.Defaults.ConnectionSSLMode = "disable"
	}
	if len(c.Defaults.ScanPorts) == 0 {
		c.Defaults.ScanPorts = []string{"5432", "10000-20000"}
	}
}

func (c Config) validate() error {
	if c.Defaults.DestinationID == "" {
		return fmt.Errorf("defaults.destination_id is required")
	}
	if _, err := uuid.Parse(c.Defaults.DestinationID); err != nil {
		return fmt.Errorf("invalid defaults.destination_id: %w", err)
	}
	if c.Defaults.CronMinuteStep < 1 || c.Defaults.CronMinuteStep > 59 {
		return fmt.Errorf("defaults.cron_minute_step must be between 1 and 59")
	}
	if c.Defaults.CronHourFrom < 0 || c.Defaults.CronHourFrom > 23 {
		return fmt.Errorf("defaults.cron_hour_from must be between 0 and 23")
	}
	if c.Defaults.CronHourTo < c.Defaults.CronHourFrom || c.Defaults.CronHourTo > 23 {
		return fmt.Errorf("defaults.cron_hour_to must be between cron_hour_from and 23")
	}
	if _, err := c.scanPorts(); err != nil {
		return err
	}
	if _, err := c.Defaults.MaxDBSizeBytes(); err != nil {
		return err
	}
	if len(c.Hosts) == 0 && len(c.HostsNoReplica) == 0 {
		return fmt.Errorf("hosts or hosts_no_replica is required")
	}
	for _, host := range append(c.Hosts, c.HostsNoReplica...) {
		if host.Name == "" {
			return fmt.Errorf("host.name is required")
		}
		for _, cluster := range host.Clusters {
			if cluster.Port < 1 || cluster.Port > 65535 {
				return fmt.Errorf("host %s has invalid cluster port %d", host.Name, cluster.Port)
			}
		}
	}
	return nil
}

func (c Config) scanPorts() ([]int, error) {
	seen := map[int]bool{}
	var ports []int
	for _, item := range c.Defaults.ScanPorts {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "-") {
			parts := strings.SplitN(item, "-", 2)
			from, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid scan_ports entry %q", item)
			}
			to, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid scan_ports entry %q", item)
			}
			if from < 1 || to > 65535 || to < from {
				return nil, fmt.Errorf("invalid scan_ports range %q", item)
			}
			for port := from; port <= to; port++ {
				if !seen[port] {
					seen[port] = true
					ports = append(ports, port)
				}
			}
			continue
		}
		port, err := strconv.Atoi(item)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid scan_ports entry %q", item)
		}
		if !seen[port] {
			seen[port] = true
			ports = append(ports, port)
		}
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("defaults.scan_ports must not be empty")
	}
	return ports, nil
}
