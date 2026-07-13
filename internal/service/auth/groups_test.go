package auth

import "testing"

func TestGroupsFromClaims(t *testing.T) {
	t.Helper()

	groups := GroupsFromClaims(map[string]any{
		"groups": []any{"pgwarden_myapp_r", "/groups/pgwarden_admins"},
	})
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %v", groups)
	}
}
