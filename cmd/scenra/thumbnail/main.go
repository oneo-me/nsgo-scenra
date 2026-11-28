// Command thumbnail is Scenra's macOS QuickLook Thumbnail Extension, drawing
// .scenra designs, components and flow charts as Finder's icons via
// scenraview.Thumbnail.
package main

import (
	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"

	"github.com/oneo-me/nsgo-scenra/scenraview"
)

func main() {
	nsgo.RunQuickLookThumbnail(func(cx *core.Scope, path string, size core.Size) core.Node {
		return scenraview.Thumbnail{Path: path, Size: size}
	})
}
