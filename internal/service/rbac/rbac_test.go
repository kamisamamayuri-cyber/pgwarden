package rbac

import (
	"testing"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
)

func TestAccessMyappGroups(t *testing.T) {
	t.Helper()

	presets := []restorations.RestorePreset{{
		ID: "myapp",
		Source: restorations.RestoreEndpoint{
			PbwName: "db-prod-01:5432-myapp",
		},
		RBAC: restorations.RestorePresetRBAC{
			ViewGroup:    "pgwarden_myapp_r",
			ExecuteGroup: "pgwarden_myapp_rwx",
		},
	}}

	svc := New("pgwarden_admins", presets)

	viewer := svc.Access([]string{"pgwarden_myapp_r"}, false)
	if !viewer.CanViewPreset("myapp") {
		t.Fatal("viewer should see preset")
	}
	if viewer.CanExecutePreset("myapp") {
		t.Fatal("viewer must not execute preset")
	}
	if !viewer.CanViewPbwName("db-prod-01:5432-myapp") {
		t.Fatal("viewer should see pbw name")
	}
	if viewer.CanExecutePbwName("db-prod-01:5432-myapp") {
		t.Fatal("viewer must not execute pbw name")
	}
	if viewer.CanManageApp() {
		t.Fatal("viewer must not manage app")
	}

	executor := svc.Access([]string{"pgwarden_myapp_rwx"}, false)
	if !executor.CanExecutePreset("myapp") {
		t.Fatal("executor should run preset")
	}
	if !executor.CanExecutePbwName("db-prod-01:5432-myapp") {
		t.Fatal("executor should run backup")
	}
	if executor.CanManageApp() {
		t.Fatal("executor must not manage app")
	}

	admin := svc.Access([]string{"pgwarden_admins"}, false)
	if !admin.CanManageApp() {
		t.Fatal("admin should manage app")
	}

	full := svc.Access(nil, true)
	if !full.CanManageApp() {
		t.Fatal("full access user should manage app")
	}
}

func TestAccessKeycloakGroupPath(t *testing.T) {
	t.Helper()

	presets := []restorations.RestorePreset{{
		ID: "myapp",
		Source: restorations.RestoreEndpoint{PbwName: "db-prod-01:5432-myapp"},
		RBAC: restorations.RestorePresetRBAC{
			ViewGroup: "pgwarden_myapp_r",
		},
	}}

	svc := New("", presets)
	access := svc.Access([]string{"/groups/pgwarden_myapp_r"}, false)
	if !access.CanViewPreset("myapp") {
		t.Fatal("expected path-style group to match")
	}
}
