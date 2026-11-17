package scenraview

import (
	"github.com/nsgo-dev/nsgo/core"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// stillGap is the room between the designs a Still draws.
const stillGap = 32

// Still draws the designs or the component of a Scenra file's Source
// still, as the editor does with nothing selected: one under another, each
// at its size, a component beside its variants, its uses drawn from the
// components and its images from the pictures of the project at Project,
// the folder holding the file's project. It lets apps show a version of a
// file, such as one Git keeps, beside another without opening an Editor;
// a design filling an outlet draws alone, at its own size. A flow, or
// Source that does not parse, draws nothing, which Drawable tells.
type Still struct {
	Source  string
	Project string
	TestID  string
	Style   core.Style
}

// Drawable reports whether Still draws source: designs or a component
// that parse.
func Drawable(source string) bool {
	f, err := flowfile.Parse(source)
	return err == nil && f.Kind != flowfile.Chart && len(f.Screens) > 0
}

// Render implements core.Node.
func (s Still) Render(cx *core.Scope) core.Node {
	f, err := flowfile.Parse(s.Source)
	if err != nil || f.Kind == flowfile.Chart {
		return core.View{TestID: s.TestID, Style: s.Style}
	}
	type project struct {
		parts components
		pics  assets
	}
	read := core.NewSignalFunc(cx, project{}, func(a, b project) bool {
		return sameComponents(a.parts, b.parts) && sameAssets(a.pics, b.pics)
	})
	if s.Project != "" {
		go func() {
			_, parts, pics, _, _ := scan(s.Project, nil)
			cx.Post(func() { read.Set(project{parts, pics}) })
		}()
	}
	return core.Dynamic(func(cx *core.Scope) core.Node {
		p := read.Get()
		sel := stillSelection(cx, f.Screens, p.parts, p.pics)
		sel.bare = func() bool { return true }
		var screens []core.Node
		for i := range f.Screens {
			sel.screen = i
			screens = append(screens, withVariants(cx, f, i, screenView(cx, f, i, sel), sel))
		}
		return core.View{TestID: s.TestID, Style: core.Style{Gap: stillGap, AlignItems: core.AlignStart}.Merge(s.Style), Children: screens}
	})
}
