package component

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	nodx "github.com/nodxdev/nodxgo"
)

type LogotypeParams struct {
	Compact bool
	Size    size
}

func Logotype(params ...LogotypeParams) nodx.Node {
	p := LogotypeParams{}
	if len(params) > 0 {
		p = params[0]
	}

	markClass := "w-[44px] h-auto shrink-0 brand-mark"
	titleClass := "brand-wordmark text-primary"
	subtitleClass := "brand-product-name text-base-content"

	switch p.Size {
	case SizeSm:
		markClass = "w-[32px] h-auto shrink-0 brand-mark"
		titleClass += " text-sm"
		subtitleClass += " text-[10px]"
	case SizeLg:
		markClass = "w-[56px] h-auto shrink-0 brand-mark"
		titleClass += " text-2xl"
		subtitleClass += " text-sm"
	default:
		titleClass += " text-lg"
		subtitleClass += " text-xs"
	}

	textBlock := nodx.Div(
		nodx.Class("min-w-0"),
		nodx.Div(
			nodx.Class(titleClass),
			nodx.Text(AppShortName),
		),
	)
	if !p.Compact {
		textBlock = nodx.Div(
			nodx.Class("min-w-0"),
			nodx.Div(
				nodx.Class(titleClass),
				nodx.Text(AppShortName),
			),
			nodx.Div(
				nodx.Class(subtitleClass),
				nodx.Text(AppProductName),
			),
		)
	}

	return nodx.Div(
		nodx.ClassMap{
			"inline-flex items-center gap-3 select-none": true,
		},
		nodx.Img(
			nodx.Class(markClass),
			nodx.Src(pathutil.BuildPath("/images/pgwarden-mark.svg")),
			nodx.Alt(AppShortName),
		),
		textBlock,
	)
}
