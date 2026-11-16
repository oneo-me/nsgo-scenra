package scenraview

import (
	"math"
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo/base/camera"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// componentEntries are the files of the components folder among files,
// by path, those that do not parse as a component too, which the
// showcase marks.
func componentEntries(files []entry) []entry {
	var out []entry
	for _, f := range files {
		if isComponentFile(f.path) {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(a, b entry) int { return strings.Compare(a.path, b.path) })
	return out
}

// isComponentFile says the file at path names a component, which the
// showcase shows instead of the tree.
func isComponentFile(path string) bool {
	_, ok := flowfile.ComponentName(path)
	return ok
}

// The showcase gives each component at least the larger of
// showcaseMinArea and showcaseMinShare of the largest one's area, scaling
// smaller ones up, at most showcaseMaxScale times, so a button beside a
// screen-wide card stays as legible as the card when the whole fits.
const (
	showcaseMinArea  = 200 * 120
	showcaseMinShare = 0.3
	showcaseMaxScale = 3.0
)

// showcaseScales are the scales the components of sizes show at, as
// their areas give them: the smallest that reaches the least area, never
// below 1 or above showcaseMaxScale.
func showcaseScales(sizes []core.Size) []float64 {
	largest := 0.0
	for _, sz := range sizes {
		largest = max(largest, float64(sz.Width*sz.Height))
	}
	least := max(showcaseMinArea, largest*showcaseMinShare)
	out := make([]float64, len(sizes))
	for i, sz := range sizes {
		out[i] = 1
		if area := float64(sz.Width * sz.Height); area > 0 && area < least {
			out[i] = min(math.Sqrt(least/area), showcaseMaxScale)
		}
	}
	return out
}

// showcaseGap is the room between the showcase's components, closer
// than a file's designs, as they show together.
const showcaseGap = 24

// brokenHeight is how high the showcase draws a component that does not
// parse.
const brokenHeight = 96

// showcaseFile is the components of entries as one file of components,
// each a design drawing its elements of parts at its size, named as the
// tree names it.
func showcaseFile(entries []entry, parts components) flowfile.File {
	f := flowfile.File{Kind: flowfile.Component}
	for _, en := range entries {
		name, _ := flowfile.ComponentName(en.path)
		title := en.name
		if title == "" {
			title = keyName(en.path)
		}
		f.Screens = append(f.Screens, flowfile.Screen{ID: name, Name: title, Device: en.device, Width: en.width, Height: en.height, UI: parts[name]})
	}
	return f
}

// showcasePane is every component of the open folder on one canvas,
// packed tight under their captions, in whatever order wastes the least
// room, once each has laid out at the height it needs, and scaled up to the least area
// showcaseScales gives it. It only shows them: nothing opens, plays or
// selects here.
func showcasePane(cx *core.Scope, e *Editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	c := &camera.State{}
	entries := core.NewMemoFunc(cx, func() []entry { return componentEntries(e.files.Get()) }, slices.Equal[[]entry])
	look := core.NewMemoFunc(cx, func() lookOf { return lookOf{parts: e.parts.Get(), assets: e.assets.Get()} },
		func(a, b lookOf) bool { return sameComponents(a.parts, b.parts) && sameAssets(a.assets, b.assets) })
	// heights are the heights the components laid out at, by index.
	heights := core.NewSignalFunc(cx, []core.Points(nil), slices.Equal[[]core.Points])
	// layout is where each component is in the plane, the size it shows
	// at and the size of the whole; one not laid out yet counts as high
	// as it is wide.
	layout := func(f flowfile.File) ([]core.Point, []core.Size, core.Size) {
		hs := heights.Get()
		sizes := make([]core.Size, len(f.Screens))
		for i := range f.Screens {
			w, h := f.Size(i)
			height := core.Points(h)
			if i < len(hs) && hs[i] > 0 {
				height = hs[i]
			} else if h == 0 {
				height = core.Points(w)
			}
			sizes[i] = core.Size{Width: core.Points(w), Height: height}
		}
		boxes := make([]core.Size, len(sizes))
		for i, k := range showcaseScales(sizes) {
			sizes[i].Width, sizes[i].Height = sizes[i].Width*core.Points(k), sizes[i].Height*core.Points(k)
			boxes[i] = core.Size{Width: sizes[i].Width, Height: sizes[i].Height + captionRoom}
		}
		at, total := packTight(boxes, showcaseGap)
		for i := range at {
			at[i].Y += captionRoom
		}
		return at, sizes, total
	}
	file := func() flowfile.File { return showcaseFile(entries.Get(), look.Get().parts) }
	picture := core.Dynamic(func(cx *core.Scope) core.Node {
		f := file()
		if len(f.Screens) == 0 {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0),
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 10}, Children: []core.Node{
				ui.Icon(lucide.Component, 28, colors.Muted),
				core.Text{Content: tr.ComponentsEmpty(), Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(320)},
					StyleFunc: ui.Colored(colors.Muted)}}}
		}
		if len(heights.GetUntracked()) != len(f.Screens) {
			heights.Set(make([]core.Points, len(f.Screens)))
		}
		play := newPlayer(cx, func() string { return "" })
		look := look.GetUntracked()
		play.scan(f.Screens, look.parts)
		sel := selection{selected: func() string { return "" }, frames: map[string]frame{}, laidOut: core.NewSignal(cx, 0), play: play,
			parts: look.parts, assets: look.assets, choose: func(string) {}, tap: func(flowfile.Tap, int) {}}
		// Each draws as its own file does: the component is around its
		// elements, which cannot use it again.
		// A component that does not parse shows where it is.
		list := entries.GetUntracked()
		var nodes []core.Node
		for i, screen := range f.Screens {
			w, _ := f.Size(i)
			s := sel
			s.using = []string{screen.ID}
			drawing := screenView(cx, f, i, s)
			if list[i].kind != flowfile.Component {
				drawing = core.View{TestID: "showcase-broken-" + screen.ID,
					Style: core.Style{Width: core.Pt(core.Points(w)), Height: core.Pt(brokenHeight), CornerRadius: 12, BorderWidth: 1,
						BorderStyle: core.BorderDashed, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
					StyleFunc: func() core.Style { return core.Style{BorderColor: colors.Danger()} },
					Children:  []core.Node{ui.Icon(lucide.FileX, 24, colors.Danger)}}
			}
			nodes = append(nodes,
				core.Dynamic(func(cx *core.Scope) core.Node {
					at, sizes, _ := layout(f)
					return caption(cx, screen.Name, at[i], int(sizes[i].Width), c.Zoom)
				}),
				// The drawing lays out at its own size, which the scale
				// grows from the top left corner of its box.
				core.View{TestID: "showcase-" + screen.ID,
					StyleFunc: func() core.Style {
						at, sizes, _ := layout(f)
						return core.Style{Position: core.Absolute, Left: core.Pt(at[i].X), Top: core.Pt(at[i].Y),
							Width: core.Pt(sizes[i].Width), Height: core.Pt(sizes[i].Height)}
					},
					Children: []core.Node{core.View{TestID: "showcase-drawing-" + screen.ID,
						StyleFunc: func() core.Style {
							_, sizes, _ := layout(f)
							return core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0),
								Transform:       core.Transform{Scale: float32(sizes[i].Width / core.Points(w))},
								TransformOrigin: core.ObjectPosition{X: core.PositionAxis{Offset: core.Pt(0)}, Y: core.PositionAxis{Offset: core.Pt(0)}}}
						},
						OnLayout: func(b core.Bounds) {
							cx.Post(func() {
								hs := slices.Clone(heights.GetUntracked())
								if i < len(hs) && hs[i] != b.Size.Height {
									hs[i] = b.Size.Height
									heights.Set(hs)
								}
							})
						},
						Children: []core.Node{drawing}}}})
		}
		return core.View{TestID: "showcase",
			StyleFunc: func() core.Style {
				_, _, total := layout(f)
				return core.Style{Width: core.Pt(total.Width), Height: core.Pt(total.Height)}
			},
			Children: nodes}
	})
	tool := func(icon func() *core.SVG, label string, press func(), testID string) core.Node {
		return ui.IconButton{Icon: icon, Label: label, OnPress: press, TestID: testID}
	}
	fin := ui.FinOf(cx)
	toolbar := core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(12), Right: core.Pt(12), Direction: core.Row, Gap: 2,
		Padding: core.All(4), CornerRadius: ui.RadiusControl + 4, BorderWidth: 1},
		StyleFunc: func() core.Style {
			return fin.Raised().Over(core.Style{Background: colors.Surface(), BorderColor: colors.Border()})
		},
		Children: []core.Node{
			tool(lucide.ZoomOut, tr.ZoomOut(), c.ZoomOut, "showcase-zoom-out"),
			tool(lucide.ZoomIn, tr.ZoomIn(), c.ZoomIn, "showcase-zoom-in"),
			tool(lucide.Scan, tr.Fit(), c.Fit, "showcase-fit"),
		}}
	canvas := camera.View{TestID: "showcase-camera", State: c, Content: picture, FitMargin: designFitMargin,
		Children: []core.Node{toolbar}}
	return core.View{Style: ui.FillRest.Merge(core.Style{Overflow: core.Hidden}), StyleFunc: ui.Filled(colors.Inset), NoFocus: true,
		Children: []core.Node{canvas}}
}
