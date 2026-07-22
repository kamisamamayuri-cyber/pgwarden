package component

import nodx "github.com/nodxdev/nodxgo"

func ToggleBadge(label string, enabled bool) nodx.Node {
	class := "badge-ghost text-base-content/50"
	if enabled {
		class = "badge-info"
	}

	return nodx.SpanEl(
		nodx.ClassMap{
			"badge":    true,
			"badge-sm": true,
			class:      true,
		},
		nodx.Text(label),
	)
}
