package scenraview

import (
	"os"

	"github.com/nsgo-dev/nsgo/base/camera"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// Thumbnail draws a Scenra file still and whole, fitted to Size, as a
// QuickLook thumbnail: like Preview it reads Path and its project's
// components when it can, but without tools. A file that is broken, empty
// or unreadable draws nothing, so Finder keeps the file's icon.
type Thumbnail struct {
	Path   string
	Size   core.Size
	TestID string
}

// Render implements core.Node.
func (t Thumbnail) Render(cx *core.Scope) core.Node {
	data, err := os.ReadFile(t.Path)
	if err != nil {
		return core.View{TestID: t.TestID}
	}
	source := string(data)
	f, err := flowfile.Parse(source)
	if err != nil || (f.Kind == flowfile.Chart && !hasFlowNodes(f)) || (f.Kind != flowfile.Chart && len(f.Screens) == 0) {
		return core.View{TestID: t.TestID}
	}
	colors := ui.ColorsOf(cx)
	bg := colors.Inset
	if f.Kind == flowfile.Chart {
		bg = colors.Background
	}
	// A sixteenth of the side keeps small icons from being mostly margin.
	margin := min(t.Size.Width, t.Size.Height) / 16
	return core.View{
		TestID:    t.TestID,
		Style:     ui.FillRest.Merge(core.Style{Overflow: core.Hidden}),
		StyleFunc: ui.Filled(bg),
		NoFocus:   true,
		Children: []core.Node{camera.View{
			State:     &camera.State{},
			FitMargin: margin,
			Content:   Still{Source: source, Project: ProjectOf(t.Path), TestID: "thumbnail-still"},
		}},
	}
}
