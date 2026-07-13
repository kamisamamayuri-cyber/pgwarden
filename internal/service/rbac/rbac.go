package rbac

import (
	"strings"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
)

// Access describes effective permissions for one authenticated subject.
type Access struct {
	enabled      bool
	fullAccess   bool
	admin        bool
	viewPbwNames map[string]struct{}
	execPbwNames map[string]struct{}
	viewPresets  map[string]struct{}
	execPresets  map[string]struct{}
}

// Service resolves permissions from restore presets. Preset and admin-group
// sources are functions so hot-reloaded YAML takes effect without a restart.
type Service struct {
	adminGroupFn func() string
	presetsFn    func() []restorations.RestorePreset
}

// New builds a service over a static preset snapshot (tests, tooling).
func New(adminGroup string, presets []restorations.RestorePreset) *Service {
	return &Service{
		adminGroupFn: func() string { return adminGroup },
		presetsFn:    func() []restorations.RestorePreset { return presets },
	}
}

// NewLive reads presets and admin group from the restorations package on every
// call, so RBAC follows hot-reloaded preset YAML. envAdminGroup, when set,
// overrides admin_group from the YAML.
func NewLive(envAdminGroup string) *Service {
	return &Service{
		adminGroupFn: func() string {
			if envAdminGroup != "" {
				return envAdminGroup
			}
			return restorations.GetAdminGroup()
		},
		presetsFn: restorations.GetPresets,
	}
}

func (s *Service) Enabled() bool {
	if s.adminGroupFn() != "" {
		return true
	}
	for _, preset := range s.presetsFn() {
		if preset.RBAC.ViewGroup != "" || preset.RBAC.ExecuteGroup != "" {
			return true
		}
	}
	return false
}

func (s *Service) Access(groups []string, fullAccess bool) Access {
	if !s.Enabled() || fullAccess {
		return Access{fullAccess: true}
	}

	if adminGroup := s.adminGroupFn(); adminGroup != "" && hasGroup(groups, adminGroup) {
		return Access{admin: true}
	}

	access := Access{
		enabled:      true,
		viewPbwNames: map[string]struct{}{},
		execPbwNames: map[string]struct{}{},
		viewPresets:  map[string]struct{}{},
		execPresets:  map[string]struct{}{},
	}

	for _, preset := range s.presetsFn() {
		pbwName := preset.Source.PbwName
		view := preset.RBAC.ViewGroup != "" && hasGroup(groups, preset.RBAC.ViewGroup)
		exec := preset.RBAC.ExecuteGroup != "" && hasGroup(groups, preset.RBAC.ExecuteGroup)

		if exec {
			access.execPresets[preset.ID] = struct{}{}
			access.execPbwNames[pbwName] = struct{}{}
			access.viewPresets[preset.ID] = struct{}{}
			access.viewPbwNames[pbwName] = struct{}{}
			continue
		}
		if view {
			access.viewPresets[preset.ID] = struct{}{}
			access.viewPbwNames[pbwName] = struct{}{}
		}
	}

	return access
}

func (a Access) Unrestricted() bool {
	return !a.enabled || a.fullAccess || a.admin
}

func (a Access) CanManageApp() bool {
	return a.Unrestricted()
}

func (a Access) CanSeeConnectionSecrets() bool {
	return a.Unrestricted()
}

func (a Access) CanViewPreset(id string) bool {
	if a.Unrestricted() {
		return true
	}
	_, ok := a.viewPresets[id]
	return ok
}

func (a Access) CanExecutePreset(id string) bool {
	if a.Unrestricted() {
		return true
	}
	_, ok := a.execPresets[id]
	return ok
}

func (a Access) CanViewPbwName(name string) bool {
	if a.Unrestricted() {
		return true
	}
	_, ok := a.viewPbwNames[name]
	return ok
}

func (a Access) CanExecutePbwName(name string) bool {
	if a.Unrestricted() {
		return true
	}
	_, ok := a.execPbwNames[name]
	return ok
}

func (a Access) NamesFilter() []string {
	if a.Unrestricted() {
		return nil
	}

	names := make([]string, 0, len(a.viewPbwNames))
	for name := range a.viewPbwNames {
		names = append(names, name)
	}
	return names
}

func hasGroup(groups []string, want string) bool {
	want = normalizeGroupName(want)
	if want == "" {
		return false
	}
	for _, group := range groups {
		if normalizeGroupName(group) == want {
			return true
		}
	}
	return false
}

func normalizeGroupName(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, "/")
	if value == "" {
		return ""
	}
	if idx := strings.LastIndex(value, "/"); idx >= 0 {
		value = value[idx+1:]
	}
	return value
}
