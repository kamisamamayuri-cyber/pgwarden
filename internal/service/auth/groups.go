package auth

import (
	"strings"
)

// GroupsFromClaims reads Keycloak group membership from OIDC/JWT claims.
func GroupsFromClaims(raw map[string]any) []string {
	if raw == nil {
		return nil
	}

	seen := map[string]struct{}{}
	add := func(values ...string) {
		for _, value := range values {
			normalized := normalizeGroupName(value)
			if normalized == "" {
				continue
			}
			seen[normalized] = struct{}{}
		}
	}

	switch groups := raw["groups"].(type) {
	case []any:
		for _, item := range groups {
			if value, ok := item.(string); ok {
				add(value)
			}
		}
	case []string:
		add(groups...)
	case string:
		add(groups)
	}

	switch groups := raw["group"].(type) {
	case []any:
		for _, item := range groups {
			if value, ok := item.(string); ok {
				add(value)
			}
		}
	case []string:
		add(groups...)
	case string:
		add(groups)
	}

	result := make([]string, 0, len(seen))
	for group := range seen {
		result = append(result, group)
	}
	return result
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

