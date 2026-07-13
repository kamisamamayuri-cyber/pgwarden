package component

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	nodx "github.com/nodxdev/nodxgo"
)

func StatusBadge(status string) nodx.Node {
	class := ""
	switch status {
	case "queued":
		class = "badge-ghost"
	case "running":
		class = "badge-info"
	case "success":
		class = "badge-success"
	case "failed":
		class = "badge-error"
	case "deleted":
		class = "badge-warning"
	default:
		class = "badge-neutral"
	}

	return nodx.SpanEl(
		nodx.ClassMap{
			"badge": true,
			class:   true,
		},
		nodx.Text(i18n.StatusLabel(status)),
	)
}
