package component

import (
	nodx "github.com/nodxdev/nodxgo"
)

func SkeletonTr(rows int) nodx.Node {
	rs := make([]nodx.Node, rows)
	for i := range rs {
		rs[i] = nodx.Tr(
			nodx.Td(
				nodx.Colspan("100%"),
				nodx.Div(
					nodx.Class("animate-pulse h-4 w-full bg-base-300 rounded-badge"),
				),
			),
		)
	}

	return nodx.Group(rs...)

}

func SkeletonCard(cards int) nodx.Node {
	cs := make([]nodx.Node, cards)
	for i := range cs {
		cs[i] = nodx.Div(
			nodx.Class("animate-pulse h-16 w-full bg-base-300 rounded-lg mb-2"),
		)
	}

	return nodx.Group(cs...)
}
