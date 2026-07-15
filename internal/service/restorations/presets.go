package restorations

import (
	"fmt"
	"strings"
)

// RestoreEndpoint describes a PostgreSQL instance used in a restore preset.
type RestoreEndpoint struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Owner    string `json:"owner,omitempty"`
	PbwName  string `json:"pbw_name"`
}

// RestoreTarget is a restore destination for a named environment.
type RestoreTarget struct {
	Environment string `json:"environment"`
	RestoreEndpoint
}

// RestorePresetRBAC maps Keycloak/AD groups to preset permissions.
type RestorePresetRBAC struct {
	ViewGroup    string `json:"view_group,omitempty" yaml:"view_group"`
	ExecuteGroup string `json:"execute_group,omitempty" yaml:"execute_group"`
}

// RestorePreset is a restore scenario exposed via API (loaded from YAML).
type RestorePreset struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	RBAC        RestorePresetRBAC   `json:"rbac,omitempty" yaml:"rbac"`
	Source      RestoreEndpoint   `json:"source"`
	Targets     []RestoreTarget   `json:"targets"`
}

func pbwName(host string, port int, database string) string {
	return fmt.Sprintf("%s:%d-%s", host, port, database)
}

func findRestorePreset(id string) (RestorePreset, bool) {
	if canonicalID, ok := restorePresetAliases[id]; ok {
		id = canonicalID
	}

	for _, preset := range getRestorePresets() {
		if preset.ID == id {
			return preset, true
		}
	}
	return RestorePreset{}, false
}

func findRestoreTargetByEnvironment(
	preset RestorePreset, environment string,
) (RestoreTarget, bool) {
	environment = normalizeEnvironment(environment)
	for _, target := range preset.Targets {
		if target.Environment == environment {
			return target, true
		}
	}
	return RestoreTarget{}, false
}

func normalizeEnvironment(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
