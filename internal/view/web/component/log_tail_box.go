package component

import (
	"strings"

	nodx "github.com/nodxdev/nodxgo"
)

func LogTailBox(lines []string, extra ...nodx.Node) nodx.Node {
	nodes := []nodx.Node{
		nodx.Class(
			"font-mono text-xs whitespace-pre-wrap break-all " +
				"max-h-56 overflow-y-auto w-full " +
				"bg-neutral text-neutral-content rounded-box p-3",
		),
	}
	nodes = append(nodes, extra...)
	nodes = append(nodes, nodx.Text(strings.Join(lines, "\n")))
	return nodx.Pre(nodes...)
}
