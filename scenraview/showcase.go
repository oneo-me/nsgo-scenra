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
// those that do not parse as a component too, which the showcase marks,
// those in no group first and then by their groups' names, and by path
// in each.
func componentEntries(files []entry) []entry {
	var out []entry
	for _, f := range files {
		if isComponentFile(f.path) {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(a, b entry) int {
		ga, _ := groupOf(a.title())
		gb, _ := groupOf(b.title())
		if c := strings.Compare(strings.ToLower(ga), strings.ToLower(gb)); c != 0 {
			return c
		}
		if c := strings.Compare(ga, gb); c != 0 {
			return c
		}
		return strings.Compare(a.path, b.path)
	})
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
// than a file's designs, as they show together; groupGap is the room
// between their groups, and groupRoom the room above a group that its
// name takes at full size.
const (
	showcaseGap = 24
	groupGap    = 96
	groupRoom   = 48
)

// brokenHeight is how high the showcase draws a component that does not
// parse, and brokenWidth how wide, unless it sets its width.
const (
	brokenHeight = 96
	brokenWidth  = 200
)

// variantGap is the room between a component and its variants.
const variantGap = 32

// stillSelection draws screens, drawn from parts and assets, only to
// show them: nothing selects or plays.
func stillSelection(cx *core.Scope, screens []flowfile.Screen, parts components, pics assets) selection {
	play := newPlayer(cx, func() string { return "" })
	sel := selection{selected: func() string { return "" }, frames: map[string]frame{}, laidOut: core.NewSignal(cx, 0), play: play,
		parts: parts, assets: pics, choose: func(string) {}, tap: func(flowfile.Tap, int, string) {}}
	play.scan(screens, parts, sel.top)
	return sel
}

// withVariants is drawing, of the component that is f's design at i,
// with its variants beside it, each over its ID and drawn from still,
// as only the component itself selects; drawing alone without any.
func withVariants(cx *core.Scope, f flowfile.File, i int, drawing core.Node, still selection) core.Node {
	variants := f.Screens[i].Variants
	if len(variants) == 0 {
		return drawing
	}
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	state := func(name string, d core.Node, testID string) core.Node {
		return core.View{TestID: testID, Style: core.Style{Gap: 8, AlignItems: core.AlignStart}, Children: []core.Node{d,
			core.Text{Content: name, MaxLines: 1, Style: core.Style{FontSize: 12, LineHeight: 16, FontWeight: ui.Medium}, StyleFunc: ui.Colored(colors.Muted)}}}
	}
	states := []core.Node{state(tr.ComponentDefault(), drawing, "variant")}
	for _, v := range variants {
		vf := f
		vf.Screens = slices.Clone(f.Screens)
		vf.Screens[i].UI = flowfile.Instance(v.Use(f.Screens[i].ID), flowfile.Part{UI: f.Screens[i].UI, Variants: variants})
		states = append(states, state(v.ID, screenView(cx, vf, i, still), "variant-"+v.ID))
	}
	return core.View{Style: core.Style{Direction: core.Row, Gap: variantGap, AlignItems: core.AlignStart}, Children: states}
}

// showcaseFile is the components of entries as one file of components,
// each a design drawing its elements of parts at its size, named as its
// file names it without its group.
func showcaseFile(entries []entry, parts components) flowfile.File {
	f := flowfile.File{Kind: flowfile.Component}
	for _, en := range entries {
		name, _ := flowfile.ComponentName(en.path)
		_, title := groupOf(en.title())
		s := en.screen()
		s.ID, s.Name, s.UI = name, title, parts[name].UI
		f.Screens = append(f.Screens, s)
	}
	return f
}

// showcaseGroups are the groups of entries, in order: each one's name,
// "" for the components in none, and the indexes of its components.
func showcaseGroups(entries []entry) []componentGroup {
	var out []componentGroup
	for i, en := range entries {
		g, _ := groupOf(en.title())
		if n := len(out); n == 0 || out[n-1].name != g {
			out = append(out, componentGroup{name: g})
		}
		out[len(out)-1].members = append(out[len(out)-1].members, i)
	}
	return out
}

// componentGroup is a group of the showcase: its name and the indexes of
// its components.
type componentGroup struct {
	name    string
	members []int
}

// placement is where the showcase's components are in the plane, by
// index, the sizes they laid out at, the scales they show at, where each
// group's name is and how wide the group is, and the size of the whole.
type placement struct {
	at     []core.Point
	sizes  []core.Size
	scales []float64
	heads  []core.Point
	widths []core.Points
	total  core.Size
}

func samePlacement(a, b placement) bool {
	return slices.Equal(a.at, b.at) && slices.Equal(a.sizes, b.sizes) && slices.Equal(a.scales, b.scales) &&
		slices.Equal(a.heads, b.heads) && slices.Equal(a.widths, b.widths) && a.total == b.total
}

// placeGroups packs boxes, of sizes, tight in each of groups, and the
// groups in their order, each under room for its name when it has one,
// so the components of a group show together. It returns each box's top
// left corner, each group's, its width and the whole's size.
func placeGroups(boxes []core.Size, groups []componentGroup) (at, heads []core.Point, widths []core.Points, total core.Size) {
	at = make([]core.Point, len(boxes))
	blocks := make([]core.Size, len(groups))
	inner := make([][]core.Point, len(groups))
	for g, group := range groups {
		sizes := make([]core.Size, len(group.members))
		for k, i := range group.members {
			sizes[k] = boxes[i]
		}
		var size core.Size
		inner[g], size = packTight(sizes, showcaseGap)
		if group.name != "" {
			size.Height += groupRoom
		}
		blocks[g] = size
	}
	heads, total = pack(blocks, groupGap)
	for g, group := range groups {
		top := heads[g].Y
		if group.name != "" {
			top += groupRoom
		}
		for k, i := range group.members {
			at[i] = core.Point{X: heads[g].X + inner[g][k].X, Y: top + inner[g][k].Y}
		}
		widths = append(widths, blocks[g].Width)
	}
	return at, heads, widths, total
}

// groupHeading names a group of the showcase over its top left corner,
// its size on screen the same at any zoom.
func groupHeading(cx *core.Scope, text string, at core.Point, width core.Points, zoom func() float64, testID string) core.Node {
	colors := ui.ColorsOf(cx)
	const size, height = 16, 16 * 1.5
	return core.Text{Content: text, MaxLines: 1, PointerEvents: core.PointerEventsNone, TestID: testID,
		StyleFunc: func() core.Style {
			z := zoom()
			return core.Style{Position: core.Absolute, Left: core.Pt(at.X), Top: core.Pt(at.Y + groupRoom - 8/core.Points(z) - height),
				Width: core.Pt(width * core.Points(z)), Height: core.Pt(height), FontSize: size, LineHeight: height,
				FontWeight: ui.Semibold, Color: colors.Text(), Transform: core.Transform{Scale: float32(1 / z)},
				TransformOrigin: core.ObjectPosition{X: core.PositionAxis{Offset: core.Pt(0)}, Y: core.PositionAxis{Offset: core.Pct(100)}}}
		}}
}

// showcasePane is every component of the open folder on one canvas,
// with its variants, under their captions, each group's packed tight
// together under its name in whatever order wastes the least room, once each has laid out at the size it
// needs, and scaled up to the least area showcaseScales gives it. It
// only shows them: nothing opens, plays or selects here.
func showcasePane(cx *core.Scope, e *Editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	c := &camera.State{}
	pane := core.NewSignal(cx, core.Size{})
	(&paneCamera{state: c, size: pane.GetUntracked}).mount(cx, &e.showcaseCanvas)
	entries := core.NewMemoFunc(cx, func() []entry { return componentEntries(e.files.Get()) }, slices.Equal[[]entry])
	look := core.NewMemoFunc(cx, func() lookOf { return lookOf{parts: e.parts.Get(), assets: e.assets.Get()} },
		func(a, b lookOf) bool { return sameComponents(a.parts, b.parts) && sameAssets(a.assets, b.assets) })
	// measured are the sizes the components laid out at, by index.
	measured := core.NewSignalFunc(cx, []core.Size(nil), slices.Equal[[]core.Size])
	// layout is where each component is in the plane, the size it laid
	// out at, the scale it shows at and the size of the whole; one not
	// laid out yet counts as wide as it sets, or brokenWidth, and as high
	// as wide.
	layout := func(f flowfile.File, groups []componentGroup) placement {
		ms := measured.Get()
		sizes := make([]core.Size, len(f.Screens))
		for i := range f.Screens {
			w, h := f.Size(i)
			if w == 0 {
				w = brokenWidth
			}
			if h == 0 {
				h = w
			}
			sizes[i] = core.Size{Width: core.Points(w), Height: core.Points(h)}
			if i < len(ms) && ms[i].Width > 0 && ms[i].Height > 0 {
				sizes[i] = ms[i]
			}
		}
		scales := showcaseScales(sizes)
		boxes := make([]core.Size, len(sizes))
		for i, k := range scales {
			boxes[i] = core.Size{Width: sizes[i].Width * core.Points(k), Height: sizes[i].Height*core.Points(k) + captionRoom}
		}
		at, heads, widths, total := placeGroups(boxes, groups)
		for i := range at {
			at[i].Y += captionRoom
		}
		return placement{at, sizes, scales, heads, widths, total}
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
		if len(measured.GetUntracked()) != len(f.Screens) {
			measured.Set(make([]core.Size, len(f.Screens)))
		}
		// Every component's place reads the one packing, packed again only
		// when a size changes.
		groups := showcaseGroups(entries.GetUntracked())
		placed := core.NewMemoFunc(cx, func() placement { return layout(f, groups) }, samePlacement)
		// The sizes the components lay out at in one layout land
		// together, so they pack once rather than once each.
		var pending map[int]core.Size
		report := func(i int, size core.Size) {
			if pending == nil {
				pending = map[int]core.Size{}
				cx.Post(func() {
					ms := slices.Clone(measured.GetUntracked())
					for i, size := range pending {
						if i < len(ms) {
							ms[i] = size
						}
					}
					pending = nil
					measured.Set(ms)
				})
			}
			pending[i] = size
		}
		look := look.GetUntracked()
		sel := stillSelection(cx, f.Screens, look.parts, look.assets)
		sel.bare = e.OutlinesHidden
		// Each draws as its own file does: the component is around its
		// elements, which cannot use it again.
		// A component that does not parse shows where it is.
		list := entries.GetUntracked()
		var nodes []core.Node
		for i, screen := range f.Screens {
			s := sel
			s.using = []string{screen.ID}
			drawing := withVariants(cx, f, i, screenView(cx, f, i, s), s)
			if list[i].kind != flowfile.Component {
				w, _ := f.Size(i)
				if w == 0 {
					w = brokenWidth
				}
				drawing = core.View{TestID: "showcase-broken-" + screen.ID,
					Style: core.Style{Width: core.Pt(core.Points(w)), Height: core.Pt(brokenHeight), CornerRadius: 12, BorderWidth: 1,
						BorderStyle: core.BorderDashed, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
					StyleFunc: func() core.Style { return core.Style{BorderColor: colors.Danger()} },
					Children:  []core.Node{ui.Icon(lucide.FileX, 24, colors.Danger)}}
			}
			nodes = append(nodes,
				core.Dynamic(func(cx *core.Scope) core.Node {
					p := placed.Get()
					return caption(cx, screen.Name, p.at[i], int(float64(p.sizes[i].Width)*p.scales[i]), c.Zoom)
				}),
				// The drawing lays out at its own size, in room wide
				// enough for any, which the scale grows from the top left
				// corner of its box.
				core.View{TestID: "showcase-" + screen.ID,
					StyleFunc: func() core.Style {
						p := placed.Get()
						k := core.Points(p.scales[i])
						return core.Style{Position: core.Absolute, Left: core.Pt(p.at[i].X), Top: core.Pt(p.at[i].Y),
							Width: core.Pt(p.sizes[i].Width * k), Height: core.Pt(p.sizes[i].Height * k)}
					},
					Children: []core.Node{core.View{PointerEvents: core.PointerEventsNone,
						StyleFunc: func() core.Style {
							return core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Width: core.Pt(flowfile.MaxSize),
								AlignItems:      core.AlignStart,
								Transform:       core.Transform{Scale: float32(placed.Get().scales[i])},
								TransformOrigin: core.ObjectPosition{X: core.PositionAxis{Offset: core.Pt(0)}, Y: core.PositionAxis{Offset: core.Pt(0)}}}
						},
						Children: []core.Node{core.View{TestID: "showcase-drawing-" + screen.ID,
							OnLayout: func(b core.Bounds) { report(i, b.Size) },
							Children: []core.Node{drawing}}}}}})
		}
		for g, group := range groups {
			if group.name == "" {
				continue
			}
			nodes = append(nodes, core.Dynamic(func(cx *core.Scope) core.Node {
				p := placed.Get()
				return groupHeading(cx, group.name, p.heads[g], p.widths[g], c.Zoom, "showcase-group-"+group.name)
			}))
		}
		return core.View{TestID: "showcase",
			StyleFunc: func() core.Style {
				total := placed.Get().total
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
		OnLayout: func(b core.Bounds) { pane.Set(b.Size) }, Children: []core.Node{canvas}}
}
