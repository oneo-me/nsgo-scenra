package scenraview

import (
	"maps"
	"slices"

	"github.com/nsgo-dev/nsgo/core"
	uiflow "github.com/nsgo-dev/nsgo/ui/flow"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// stillName is the name a Still's file takes among its project's designs,
// one no file has.
const stillName = "\x00still"

// Still draws a Scenra file's Source still, as the editor does with
// nothing selected. Several designs pack as the editor packs them, under
// their captions, a component beside its variants, their uses drawn from
// the components and their images from the pictures of the project at
// Project, the folder holding the file's project; a design filling an
// outlet there draws alone, as the editor draws it, at the outlet's size
// and in its look. Flows
// draw as the editor lays them out, without its tools, selection or
// edits. It lets apps show a version of a file, such as one Git keeps,
// beside another without opening an Editor, at the size it takes on a
// canvas of their own. Source that does not parse draws nothing, which
// Drawable tells.
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
		read  bool
	}
	read := core.NewSignalFunc(cx, project{}, func(a, b project) bool {
		return a.read == b.read && sameComponents(a.parts, b.parts) && sameAssets(a.pics, b.pics) && sameDesigns(a.docs, b.docs)
	})
	if s.Project != "" {
		go func() {
			_, parts, pics, _, docs := scan(s.Project, nil)
			cx.Post(func() { read.Set(project{parts, pics, docs, true}) })
		}()
	}
	// measured are the sizes the designs as large as they lay out laid
	// out at, by index.
	measured := core.NewSignalFunc(cx, make([]core.Size, len(f.Screens)), slices.Equal[[]core.Size])
	var pending map[int]core.Size
	// report keeps the size a design laid out at, those of one layout
	// together, so they pack once rather than once each.
	report := func(i int, size core.Size) {
		if pending == nil {
			pending = map[int]core.Size{}
			cx.Post(func() {
				ms := slices.Clone(measured.GetUntracked())
				for i, size := range pending {
					ms[i] = size
				}
				pending = nil
				measured.Set(ms)
			})
		}
		pending[i] = size
	}
	// hosted are the outlets the designs filling them lay out at, by the
	// designs' index, as the editor measures them.
	hosted := core.NewSignalFunc(cx, map[int]hosting(nil), maps.Equal[map[int]hosting])
	return core.Dynamic(func(cx *core.Scope) core.Node {
		p := read.Get()
		sel := stillSelection(cx, f.Screens, p.parts, p.pics)
		sel.bare = func() bool { return true }
		// The file is the project's under a name of its own, as this
		// version of it may not be the one the project holds.
		ds := p.docs.with(stillName, f)
		hosts := unseen("still-hosts", outletHosts(cx, ds, stillName, f, p.parts, p.pics, hosted))
		// hosting tells the designs that fill outlets, which wait for the
		// project to be read to tell whether their hosts are there.
		hosting := func(i int) bool {
			if _, in := f.HostOf(i); !in {
				return false
			}
			if s.Project != "" && !p.read {
				return true
			}
			hops, msg := ds.hosts(place{stillName, i})
			return msg == "" && len(hops) > 0
		}
		// waiting reports whether a design filling an outlet has yet to
		// learn its outlet's size. Until it does, it draws nothing and
		// the designs stay unseen, as in the editor, rather than showing
		// a moment at its device's size.
		waiting := func() bool {
			hs := hosted.Get()
			for i := range f.Screens {
				if _, ok := hs[i]; !ok && hosting(i) {
					return true
				}
			}
			return false
		}
		hide := func() core.Style {
			if waiting() {
				return core.Style{Opacity: core.Alpha(0)}
			}
			return core.Style{}
		}
		// draw is the design at i and the size it draws at, 0 where it is
		// as large as it lays out, as a component beside its variants is
		// and a design filling an outlet is once the outlet lays out.
		draw := func(i int) (core.Node, int, int) {
			if hosting(i) {
				return core.Dynamic(func(cx *core.Scope) core.Node {
					s := sel
					s.screen, s.hosted = i, hosted.Get()
					if _, ok := s.hosted[i]; !ok {
						return core.View{}
					}
					return screenView(cx, f, i, s)
				}), 0, 0
			}
			s := sel
			s.screen = i
			w, h := f.Size(i)
			if len(f.Screens[i].Variants) > 0 {
				w, h = 0, 0
			}
			return withVariants(cx, f, i, screenView(cx, f, i, s), s), w, h
		}
		if !f.Several() {
			drawing, _, _ := draw(0)
			return core.View{TestID: s.TestID, StyleFunc: func() core.Style {
				return core.Style{AlignItems: core.AlignStart}.Merge(s.Style).Merge(hide())
			}, Children: []core.Node{drawing, hosts}}
		}
		// Several designs pack as the editor packs them, under their
		// captions.
		drawings := make([]core.Node, len(f.Screens))
		fixed := make([]core.Size, len(f.Screens))
		for i := range f.Screens {
			var w, h int
			drawings[i], w, h = draw(i)
			fixed[i] = core.Size{Width: core.Points(w), Height: core.Points(h)}
		}
		type layout struct {
			at    []core.Point
			sizes []core.Size
			total core.Size
		}
		placed := core.NewMemoFunc(cx, func() layout {
			ms := measured.Get()
			sizes := slices.Clone(fixed)
			for i, size := range sizes {
				if size.Width == 0 {
					sizes[i].Width = ms[i].Width
				}
				if size.Height == 0 {
					sizes[i].Height = ms[i].Height
				}
			}
			at, total := packDesigns(sizes, f.Spacing())
			return layout{at, sizes, total}
		}, func(a, b layout) bool {
			return slices.Equal(a.at, b.at) && slices.Equal(a.sizes, b.sizes) && a.total == b.total
		})
		nodes := []core.Node{hosts}
		for i, screen := range f.Screens {
			name := screen.Name
			if name == "" {
				name = screen.ID
			}
			drawing := drawings[i]
			if fixed[i].Width == 0 || fixed[i].Height == 0 {
				drawing = core.View{OnLayout: func(b core.Bounds) { report(i, b.Size) }, Children: []core.Node{drawing}}
			}
			nodes = append(nodes,
				core.Dynamic(func(cx *core.Scope) core.Node {
					l := placed.Get()
					return caption(cx, name, l.at[i], int(l.sizes[i].Width), captionRoom+core.Points(f.Spacing()), func() float64 { return 1 })
				}),
				core.View{TestID: "screen-" + screen.ID, StyleFunc: func() core.Style {
					at := placed.Get().at[i]
					return core.Style{Position: core.Absolute, Left: core.Pt(at.X), Top: core.Pt(at.Y)}
				}, Children: []core.Node{drawing}})
		}
		return core.View{TestID: s.TestID, StyleFunc: func() core.Style {
			total := placed.Get().total
			return core.Style{Width: core.Pt(total.Width), Height: core.Pt(total.Height)}.Merge(s.Style).Merge(hide())
		}, Children: nodes}
	})
}
