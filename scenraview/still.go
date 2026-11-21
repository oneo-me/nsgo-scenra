package scenraview

import (
	"github.com/nsgo-dev/nsgo/core"
	uiflow "github.com/nsgo-dev/nsgo/ui/flow"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// stillGap is the room between the designs a Still draws.
const stillGap = 32

// stillName is the name a Still's file takes among its project's designs,
// one no file has.
const stillName = "\x00still"

// Still draws a Scenra file's Source still, as the editor does with
// nothing selected. Designs draw one under another, each at its size, a
// component beside its variants, their uses drawn from the components and
// their images from the pictures of the project at Project, the folder
// holding the file's project; a design filling an outlet draws in its
// hosts there, or else alone at its own size. Flows draw as the editor
// lays them out, without its tools, selection or edits. It lets apps show
// a version of a file, such as one Git keeps, beside another without
// opening an Editor, at the size it takes on a canvas of their own.
// Source that does not parse draws nothing, which Drawable tells.
type Still struct {
	Source  string
	Project string
	TestID  string
	Style   core.Style
}

// Drawable reports whether Still draws source: designs, a component or
// flows that parse.
func Drawable(source string) bool {
	f, err := flowfile.Parse(source)
	return err == nil && (f.Kind == flowfile.Chart || len(f.Screens) > 0)
}

// Render implements core.Node.
func (s Still) Render(cx *core.Scope) core.Node {
	f, err := flowfile.Parse(s.Source)
	if err != nil {
		return core.View{TestID: s.TestID, Style: s.Style}
	}
	if f.Kind == flowfile.Chart {
		var charts []*uiflow.State
		chart := func(i int) *uiflow.State {
			for len(charts) <= i {
				charts = append(charts, &uiflow.State{})
			}
			return charts[i]
		}
		board, _ := flowBoard(cx, func() flowfile.File { return f }, chart, func() float64 { return 1 }, nil, "flows")
		return core.View{TestID: s.TestID, Style: core.Style{AlignItems: core.AlignStart}.Merge(s.Style), Children: []core.Node{board}}
	}
	type project struct {
		parts components
		pics  assets
		docs  designs
	}
	read := core.NewSignalFunc(cx, project{}, func(a, b project) bool {
		return sameComponents(a.parts, b.parts) && sameAssets(a.pics, b.pics) && sameDesigns(a.docs, b.docs)
	})
	if s.Project != "" {
		go func() {
			_, parts, pics, _, docs := scan(s.Project, nil)
			cx.Post(func() { read.Set(project{parts, pics, docs}) })
		}()
	}
	return core.Dynamic(func(cx *core.Scope) core.Node {
		p := read.Get()
		sel := stillSelection(cx, f.Screens, p.parts, p.pics)
		sel.bare = func() bool { return true }
		// The file is the project's under a name of its own, as this
		// version of it may not be the one the project holds.
		ds := p.docs.with(stillName, f)
		var screens []core.Node
		for i := range f.Screens {
			if hops, msg := ds.hosts(place{stillName, i}); msg == "" && len(hops) > 0 {
				one, shown, _ := composed(ds, hops, place{stillName, i})
				host := stillSelection(cx, one.Screens, p.parts, p.pics)
				host.bare = sel.bare
				host.fills = func(k string) ([]flowfile.Element, bool) { return fillOf(shown, ds, k) }
				screens = append(screens, screenView(cx, one, 0, host))
				continue
			}
			sel.screen = i
			screens = append(screens, withVariants(cx, f, i, screenView(cx, f, i, sel), sel))
		}
		return core.View{TestID: s.TestID, Style: core.Style{Gap: stillGap, AlignItems: core.AlignStart}.Merge(s.Style), Children: screens}
	})
}
