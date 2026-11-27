// Command quicklook is Scenra's macOS QuickLook Preview Extension, showing
// .scenra designs, components and flow charts in Finder via scenraview.Preview.
package main

import (
	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"

	"github.com/oneo-me/nsgo-scenra/scenraview"
)

func main() {
	nsgo.RunQuickLook(func(cx *core.Scope, path string) core.Node {
		return scenraview.Preview{Path: path}
	}, nsgo.WindowSize(960, 680))
}
