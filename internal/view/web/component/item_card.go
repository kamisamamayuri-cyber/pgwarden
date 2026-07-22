package component

import nodx "github.com/nodxdev/nodxgo"

func Stat(label string, value nodx.Node) nodx.Node {
	return nodx.SpanEl(
		nodx.Class("inline-flex items-center text-sm"),
		nodx.SpanEl(nodx.Class("text-base-content/60"), nodx.Text(label+": ")),
		nodx.SpanEl(nodx.Class("font-medium inline-flex items-center"), value),
	)
}

func ItemCard(extraAttrs []nodx.Node, header []nodx.Node, stats []nodx.Node) nodx.Node {
	attrs := append([]nodx.Node{nodx.Class("border border-base-300 rounded-lg p-3 mb-2")}, extraAttrs...)
	attrs = append(attrs,
		nodx.Div(
			nodx.Class("flex items-center gap-2 mb-2"),
			nodx.Group(header...),
		),
		nodx.Div(
			nodx.Class("flex flex-wrap items-center gap-x-6 gap-y-1"),
			nodx.Group(stats...),
		),
	)
	return nodx.Div(attrs...)
}
