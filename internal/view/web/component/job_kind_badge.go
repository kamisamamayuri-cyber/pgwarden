package component

import nodx "github.com/nodxdev/nodxgo"

func JobKindBadge(kind string) nodx.Node {
	label := "Backup"
	class := "badge-neutral"
	switch kind {
	case "fix_owner":
		label = "Fix owner"
		class = "badge-warning"
	case "restore":
		label = "Restore"
		class = "badge-error"
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
