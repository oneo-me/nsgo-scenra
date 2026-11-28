package scenraview

import (
	"maps"
	"slices"
	"time"

	"github.com/nsgo-dev/nsgo/base/camera"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// designFitMargin keeps a fitted design from the pane's edges.
const designFitMargin = 32

// captionRoom is the room above each design of a file of several that
// its caption takes at full size, and captionSize the caption's font
// size on screen, the same at any zoom.
const (
	captionRoom = 36
	captionSize = 13
)

// screenLayout is where each design of f is in the plane, packed
// f.Spacing() apart under their captions, and the size of the whole, 0
// for a file of one, which is as large as it lays out; designs filling
// outlets are as large as hosted says.
func screenLayout(f flowfile.File, hosted map[int]hosting) ([]core.Point, core.Size) {
	if !f.Several() {
		return []core.Point{{}}, core.Size{}
	}
	sizes := make([]core.Size, len(f.Screens))
	for i := range f.Screens {
		w, h := sizeOf(f, i, hosted)
		sizes[i] = core.Size{Width: core.Points(w), Height: core.Points(h)}
	}
	return packDesigns(sizes, f.Spacing())
}

// packDesigns is where designs of sizes are in the plane, packed spacing
// apart under their captions, and the size of the whole.
func packDesigns(sizes []core.Size, spacing int) ([]core.Point, core.Size) {
	boxes := make([]core.Size, len(sizes))
	for i, s := range sizes {
		boxes[i] = core.Size{Width: s.Width, Height: s.Height + captionRoom}
	}
	at, total := pack(boxes, core.Points(spacing))
	for i := range at {
		at[i].Y += captionRoom
	}
	return at, total
}

// screenBounds is the frame of the design of f at index i in the plane,
// laid out at origin; a component of no width or height is as large as
// it laid out, in measured, and a design filling an outlet as hosted
// says.
func screenBounds(f flowfile.File, i int, origin core.Point, measured map[int]core.Size, hosted map[int]hosting) core.Bounds {
	w, h := sizeOf(f, i, hosted)
	size := core.Size{Width: core.Points(w), Height: core.Points(h)}
	if w == 0 {
		size.Width = measured[i].Width
	}
	if h == 0 {
		size.Height = measured[i].Height
	}
	return core.Bounds{Origin: origin, Size: size}
}

// caption names a design of a file of several over its top left corner,
// its size on screen the same at any zoom unless it would then take more
// than room, the room above at in the plane, and cover what is above.
func caption(cx *core.Scope, text string, at core.Point, width int, room core.Points, zoom func() float64) core.Node {
	colors := ui.ColorsOf(cx)
	return core.Text{Content: text, MaxLines: 1, PointerEvents: core.PointerEventsNone,
		StyleFunc: func() core.Style {
			k := captionScale(zoom(), room)
			return core.Style{Position: core.Absolute, Left: core.Pt(at.X), Top: core.Pt(at.Y - core.Points(captionGap*k) - captionHeight),
				Width: core.Pt(core.Points(float64(width) / k)), Height: core.Pt(captionHeight), FontSize: captionSize, LineHeight: captionHeight,
				FontWeight: ui.Medium, Color: colors.Muted(), Transform: core.Transform{Scale: float32(k)},
				TransformOrigin: core.ObjectPosition{X: core.PositionAxis{Offset: core.Pt(0)}, Y: core.PositionAxis{Offset: core.Pct(100)}}}
		}}
}

// captionGap is the room between a caption and its design on screen, and
// captionHeight the caption's height there.
const (
	captionGap    = 6
	captionHeight = captionSize * 1.6
)

// captionScale is how much a caption grows in the plane at zoom: as much
// as keeps its size on screen, but no more than fits it and its gap in
// room.
func captionScale(zoom float64, room core.Points) float64 {
	return min(1/zoom, float64(room)/(captionGap+captionHeight))
}

// lookOf is what a design is drawn from: its text, and the components,
// pictures and the designs it may use or fill.
type lookOf struct {
	text    string
	parts   components
	assets  assets
	designs designs
}

// peekDelay is how long the pointer rests on an element before its card
// shows, or off it before the card goes.
const peekDelay = 600 * time.Millisecond

// designPane is the designs of the file shown on the canvas, packed under
// their captions when there are several, with its zoom
// tools, played as the app would be. A layer over the designs takes the
// pointer and finds the element under it itself, as the camera's
// transform moves the design where the platforms' hit testing does not
// follow: an outline marks the element the pointer is over, pressing an
// element with a tap plays it, as do those it holds, and pressing another
// selects it, as Alt does any, and pressing around it clears the
// selection. What can be tapped glows a moment as a design shows or is
// read again, and again from the toolbar, and with it the tags of the
// elements the user noted. Pins mark the elements the user commented on,
// and a card edits the comment and note of the element the pointer rests on
// or selected, beside where it pressed or else beside the element; ask,
// when set, sends the assistant a prompt about them.
func designPane(cx *core.Scope, e *Editor, ask func(prompt string)) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	fin := ui.FinOf(cx)
	c := &camera.State{}
	// measured are the sizes the designs laid out at, by index, and pane
	// the pane's.
	measured := map[int]core.Size{}
	pane := core.NewSignal(cx, core.Size{})
	(&paneCamera{state: c, size: pane.GetUntracked}).mount(cx, &e.designCanvas)
	// A design is drawn again as it, a component or a picture changes.
	look := core.NewMemoFunc(cx, func() lookOf {
		return lookOf{e.cur().doc.Get().Format(), e.parts.Get(), e.assets.Get(), e.designs.Get()}
	}, func(a, b lookOf) bool {
		return a.text == b.text && sameComponents(a.parts, b.parts) && sameAssets(a.assets, b.assets) && sameDesigns(a.designs, b.designs)
	})
	play := newPlayer(cx, func() string { return e.designName(e.cur().path) })
	// The outlets play the folder's designs, the one shown as it is here.
	play.designs = func() designs {
		d := e.cur()
		return look.Get().designs.with(e.designName(d.path), d.doc.Get())
	}
	// hosted are the outlets the designs filling them lay out at, by the
	// designs' index.
	hosted := core.NewSignalFunc(cx, map[int]hosting(nil), maps.Equal[map[int]hosting])
	hovered := core.NewSignal(cx, "")
	// tappable says the pointer is over an element pressing which plays a
	// tap.
	tappable := core.NewSignal(cx, false)
	// peek is the element whose card shows, held while the pointer is
	// over the card or the card has changes.
	peek := core.NewSignal(cx, "")
	overCard, held := core.NewSignal(cx, false), core.NewSignal(cx, false)
	// clicked is the element the pointer last selected and where, in the
	// design's coordinates.
	type click struct {
		path string
		at   core.Point
	}
	clicked := core.NewSignal(cx, click{})
	// Another file shows its tabs at their first, and no card.
	shown := core.NewMemo(cx, e.cur)
	cx.Effect(func() {
		shown.Get()
		core.Untracked(cx, func() bool {
			play.tabs.Set(nil)
			play.shown.Set(nil)
			hosted.Set(nil)
			peek.Set("")
			clicked.Set(click{})
			return true
		})
	})
	// It shows fitted, or the design a tap opened in it.
	cx.Effect(func() {
		shown.Get()
		e.focus.Get()
		cx.Post(func() {
			f := e.shown().doc.GetUntracked()
			id := e.focus.GetUntracked().id
			i, ok := f.Screen(id)
			if id == "" || !ok || !f.Several() {
				c.Fit()
				return
			}
			origins, _ := screenLayout(f, hosted.GetUntracked())
			b := screenBounds(f, i, origins[i], measured, hosted.GetUntracked())
			b.Origin.Y -= captionRoom
			b.Size.Height += captionRoom
			c.FitBounds(b)
		})
	})
	// glow is the opacity of the layer marking the elements with a tap.
	glow := core.NewAnimatedValue(cx, 0)
	var glowing *core.AnimationRun
	flash := func() {
		if glowing != nil {
			glowing.Stop()
		}
		glow.Set(0)
		glowing = cx.StartAnimation(core.Sequence(
			core.Timing(glow, 1, core.TimingConfig{Duration: 150 * time.Millisecond, Easing: core.BezierEasing(core.EaseOut)}),
			core.Delay(900*time.Millisecond),
			core.Timing(glow, 0, core.TimingConfig{Duration: 450 * time.Millisecond, Easing: core.BezierEasing(core.EaseIn)}),
		), nil)
	}
	cx.Effect(func() {
		shown.Get()
		e.reloads.Get()
		cx.Post(flash)
	})
	sel := selection{
		bare:     e.OutlinesHidden,
		selected: func() string { return e.cur().element.Get() },
		choose: func(path string) {
			cx.Batch(func() {
				e.shown().element.Set(path)
				peek.Set(path)
			})
		},
		tap: func(t flowfile.Tap, screen int, outlet string) {
			peek.Set("")
			switch t.Action {
			case flowfile.TapOpen:
				if !play.open(t, screen) {
					e.tapOpen(t, screen)
				}
			case flowfile.TapClose:
				if t.Target == "" {
					if !play.close(outlet) {
						e.tapClose()
					}
					break
				}
				// A design it shows in closes: an outlet's goes back, and
				// the one drawn returns to what opened it, its outlets
				// showing their own elements when it opens again.
				key, ok := play.closing(t, screen, outlet)
				switch {
				case !ok:
				case key != "":
					play.close(key)
				case len(e.trail) > 0:
					play.forget(screen)
					e.tapClose()
				}
			case flowfile.TapTab:
				play.showTab(t.Target, screen)
			}
		},
		frames:  map[string]frame{},
		laidOut: core.NewSignal(cx, 0),
		play:    play,
	}
	// The card follows the element the pointer rests on, once it has
	// rested there a while, unless it is held.
	var stop func() bool
	cx.Effect(func() {
		path, over, keep := hovered.Get(), overCard.Get(), held.Get()
		if stop != nil {
			stop()
			stop = nil
		}
		if over || keep {
			return
		}
		stop = cx.AfterFunc(peekDelay, func() {
			if !held.GetUntracked() && !overCard.GetUntracked() {
				peek.Set(path)
			}
		})
	})
	// mark outlines the element at path, its line width points on screen
	// at any zoom.
	mark := func(path func() string, width float64, testID string) core.Node {
		return core.View{TestID: testID, PointerEvents: core.PointerEventsNone,
			StyleFunc: func() core.Style {
				sel.laidOut.Get()
				b, ok := sel.bounds(path())
				if !ok {
					return core.Style{Display: core.DisplayNone}
				}
				w := core.Points(width / c.Zoom())
				return core.Style{Position: core.Absolute, Left: core.Pt(b.Origin.X - w), Top: core.Pt(b.Origin.Y - w),
					Width: core.Pt(b.Size.Width + 2*w), Height: core.Pt(b.Size.Height + 2*w), BorderWidth: w, BorderColor: colors.Accent()}
			}}
	}
	selected := mark(sel.selected, 2, "design-selected")
	hover := mark(func() string {
		if p := hovered.Get(); p != sel.selected() {
			return p
		}
		return ""
	}, 1, "design-hovered")
	// allTaps are the keys of the elements of f's designs with a tap.
	allTaps := func(sel selection, f flowfile.File) []string {
		var out []string
		for i, s := range f.Screens {
			out = append(out, sel.taps(s.UI, sel.top(i), nil)...)
		}
		return out
	}
	// taps mark the elements with a tap that show while glow is up.
	taps := func(s flowfile.File) core.Node {
		return core.View{TestID: "design-taps", PointerEvents: core.PointerEventsNone,
			Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0),
				Animated: &core.AnimatedStyle{Opacity: glow}},
			Children: []core.Node{core.Dynamic(func(*core.Scope) core.Node {
				var out core.Group
				for _, key := range allTaps(sel, s) {
					out = append(out, core.View{TestID: "design-tap-" + key, StyleFunc: func() core.Style {
						sel.laidOut.Get()
						b, ok := sel.bounds(key)
						if !ok {
							return core.Style{Display: core.DisplayNone}
						}
						w := core.Points(1.5 / c.Zoom())
						return core.Style{Position: core.Absolute, Left: core.Pt(b.Origin.X), Top: core.Pt(b.Origin.Y),
							Width: core.Pt(b.Size.Width), Height: core.Pt(b.Size.Height), BorderWidth: w,
							CornerRadius: core.Points(4 / c.Zoom()), Background: colors.Accent().Faded(.18), BorderColor: colors.Accent()}
					}})
				}
				return out
			})}}
	}
	// hasTaps says an element of the designs has a tap, in any panel.
	hasTaps := func() bool {
		return len(allTaps(selection{parts: look.Get().parts, fills: play.fill}, e.cur().doc.Get())) > 0
	}
	// targetAt is the element of the designs at a point of them, the tap
	// pressing there plays and the index of its design.
	targetAt := func(at core.Point) (string, *flowfile.Tap, int, string) {
		s := e.cur().doc.Get()
		for i := range s.Screens {
			if i >= len(sel.origins) {
				break
			}
			b := screenBounds(s, i, sel.origins[i], measured, hosted.Get())
			if at.X >= b.Origin.X && at.Y >= b.Origin.Y && at.X < b.Origin.X+b.Size.Width && at.Y < b.Origin.Y+b.Size.Height {
				path, tap, outlet := target(sel.hits(s.Screens[i].UI, sel.top(i), nil, sel.origins[i], at))
				return path, tap, i, outlet
			}
		}
		return "", nil, 0, ""
	}
	// A changed design is laid out afresh, the pointer over nothing until
	// it moves.
	cx.Effect(func() {
		look.Get()
		hovered.Set("")
		tappable.Set(false)
	})
	picture := core.Dynamic(func(cx *core.Scope) core.Node {
		sel.parts, sel.assets = look.Get().parts, look.Get().assets
		// What the outlets show draws again as it changes, and the
		// designs filling outlets as their outlets lay out.
		shown, ds := play.shown.Get(), play.designs()
		sel.hosted = hosted.Get()
		sel.fills = func(key string) ([]flowfile.Element, bool) { return fillOf(shown, ds, key) }
		s := core.Untracked(cx, func() flowfile.File { return e.cur().doc.Get() })
		clear(sel.frames)
		play.scan(s.Screens, sel.parts, sel.top)
		if !slices.ContainsFunc(s.Screens, func(s flowfile.Screen) bool { return len(s.UI) > 0 }) {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0),
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 10}, Children: []core.Node{
				ui.Icon(lucide.PanelsTopLeft, 28, colors.Muted),
				core.Text{Content: tr.DesignEmpty(), Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(320)},
					StyleFunc: ui.Colored(colors.Muted)}}}
		}
		origins, total := screenLayout(s, sel.hosted)
		sel.origins = origins
		clear(measured)
		// framed records the size a design laid out at.
		framed := func(i int, view core.Node) core.Node {
			return core.View{OnLayout: func(b core.Bounds) { measured[i] = b.Size }, Children: []core.Node{view}}
		}
		var style core.Style
		// The designs, with their marks, stay unseen until the outlets
		// they fill lay out, rather than showing a moment at their
		// devices' sizes.
		file := e.designName(core.Untracked(cx, e.cur).path)
		for i := range s.Screens {
			if _, ok := sel.hosted[i]; ok {
				continue
			}
			if _, in := s.HostOf(i); in {
				if hops, msg := ds.hosts(place{file, i}); msg == "" && len(hops) > 0 {
					style.Opacity = core.Alpha(0)
				}
			}
		}
		var screens []core.Node
		if !s.Several() {
			// A component's variants show beside it.
			still := stillSelection(cx, s.Screens, sel.parts, sel.assets)
			still.bare = sel.bare
			screens = []core.Node{withVariants(cx, s, 0, framed(0, screenView(cx, s, 0, sel)), still)}
		} else {
			style.Width, style.Height = core.Pt(total.Width), core.Pt(total.Height)
			for i, screen := range s.Screens {
				w, _ := sizeOf(s, i, sel.hosted)
				name := screen.Name
				if name == "" {
					name = screen.ID
				}
				screens = append(screens, caption(cx, name, origins[i], w, captionRoom+core.Points(s.Spacing()), c.Zoom),
					core.View{TestID: "screen-" + screen.ID, Style: core.Style{Position: core.Absolute, Left: core.Pt(origins[i].X), Top: core.Pt(origins[i].Y)},
						Children: []core.Node{framed(i, screenView(cx, s, i, sel))}})
			}
		}
		// The camera's pointer layer takes the pointer over the designs.
		return core.View{TestID: "design", Style: style, PointerEvents: core.PointerEventsNone,
			Children: append(append(append(screens, taps(s)), remarkPins(cx, s, sel, c.Zoom, glow)...), hover, selected)}
	})
	// comments counts the designs' comments.
	comments := func() int { return len(e.cur().doc.Get().Comments()) }
	// anchor is the point of the view the card opens beside.
	anchor := func() (core.Point, bool) {
		path := peek.Get()
		if k := clicked.Get(); k.path == path {
			return c.ToView(k.at), true
		}
		sel.laidOut.Get()
		b, ok := sel.bounds(path)
		return c.ToView(core.Point{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y}), ok
	}
	// place keeps a card of size beside the anchor and inside the view.
	place := func(size core.Size) (core.Point, bool) {
		p, ok := anchor()
		if !ok {
			return core.Point{}, false
		}
		const gap, margin = 12, 8
		w, h := pane.Get().Width, pane.Get().Height
		x, y := p.X+gap, p.Y+gap
		if x+size.Width > w-margin {
			x = p.X - gap - size.Width
		}
		if y+size.Height > h-margin {
			y = p.Y - gap - size.Height
		}
		return core.Point{X: max(margin, min(x, w-margin-size.Width)), Y: max(margin, min(y, h-margin-size.Height))}, true
	}
	tool := func(icon func() *core.SVG, label string, press func(), testID string) core.Node {
		return ui.IconButton{Icon: icon, Label: label, OnPress: press, TestID: testID}
	}
	toolbar := core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(12), Right: core.Pt(12), Direction: core.Row, Gap: 2,
		Padding: core.All(4), CornerRadius: ui.RadiusControl + 4, BorderWidth: 1},
		StyleFunc: func() core.Style {
			return fin.Raised().Over(core.Style{Background: colors.Surface(), BorderColor: colors.Border()})
		},
		Children: []core.Node{
			tool(lucide.ZoomOut, tr.ZoomOut(), c.ZoomOut, "design-zoom-out"),
			tool(lucide.ZoomIn, tr.ZoomIn(), c.ZoomIn, "design-zoom-in"),
			tool(lucide.Scan, tr.Fit(), c.Fit, "design-fit"),
			core.Show{When: func() bool { return hasTaps() || len(e.cur().doc.Get().Notes()) > 0 }, Then: func(*core.Scope) core.Node {
				return tool(lucide.MousePointerClick, tr.ShowTaps(), flash, "design-show-taps")
			}},
			core.Show{When: func() bool { return ask != nil && comments() > 0 && !e.Working() }, Then: func(*core.Scope) core.Node {
				return tool(lucide.MessagesSquare, tr.ResolveComments(), func() { ask(tr.ResolveCommentsPrompt(core.Untracked(cx, comments))) }, "resolve-comments")
			}},
		}}
	// The camera hands the pointer on at points of the designs: a press
	// plays a tap, unless Alt is held, or selects, and empty space leaves
	// a double press to zoom.
	press := func(at core.Point, ev core.PressEvent) bool {
		path, tap, screen, outlet := targetAt(at)
		if tap != nil && !ev.Modifiers.Alt {
			sel.tap(*tap, screen, outlet)
			return true
		}
		clicked.Set(click{path, at})
		sel.choose(path)
		return path != ""
	}
	pointAt := func(at core.Point, over bool) {
		path, tap := "", (*flowfile.Tap)(nil)
		if over {
			path, tap, _, _ = targetAt(at)
		}
		cx.Batch(func() {
			hovered.Set(path)
			tappable.Set(tap != nil)
		})
	}
	canvas := camera.View{TestID: "design-camera", State: c, Content: picture, OnPress: press, OnHover: pointAt, FitMargin: designFitMargin, Style: overflowing,
		Cursor: func() core.Cursor {
			if tappable.Get() {
				return core.CursorPointer
			}
			return core.CursorAuto
		},
		Children: []core.Node{toolbar,
			// The user cannot comment while the agent works.
			core.Show{When: func() bool { return peek.Get() != "" && !e.Working() }, Then: func(cx *core.Scope) core.Node {
				return remarkCard(cx, e, remarkOptions{path: peek.Get, ask: ask, place: place, over: overCard.Set, held: held.Set})
			}},
		}}
	return core.View{Style: ui.FillRest, NoFocus: true,
		OnLayout: func(b core.Bounds) { pane.Set(b.Size) }, Children: []core.Node{backdrop(e, colors.Inset), hostsOf(cx, e, look, hosted), withMenu(canvas, elementMenu(e, hovered.GetUntracked))}}
}

// withMenu has a pane's canvas hear secondary clicks with menu, unless
// it is nil. The camera passes them on unmoved, so menu reads what the
// pointer is over from the hovering the camera hands on, which leaves
// what is over the camera's tools or cards.
func withMenu(canvas core.Node, menu func(core.ContextMenuEvent)) core.Node {
	if menu == nil {
		return canvas
	}
	return core.View{Style: ui.FillRest, OnContextMenu: menu, Children: []core.Node{canvas}}
}

// elementMenu tells the editor's host of a secondary click on the element
// at the key hovered says, nil without a host hearing them.
func elementMenu(e *Editor, hovered func() string) func(core.ContextMenuEvent) {
	if e.opts.OnElementMenu == nil {
		return nil
	}
	return func(ev core.ContextMenuEvent) {
		if screen, path, ok := splitKey(hovered()); ok {
			e.opts.OnElementMenu(ElementMenuEvent{Design: screen, Path: path, At: ev.PagePosition, Modifiers: ev.Modifiers})
		}
	}
}

// unseen holds views laid out but neither seen, pressed nor heard by
// assistive technology.
func unseen(testID string, kids []core.Node) core.Node {
	return core.View{TestID: testID, PointerEvents: core.PointerEventsNone, AccessibilityElementsHidden: func() bool { return true },
		Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Opacity: core.Alpha(0)}, Children: kids}
}

// hostsOf lays out, unseen, the designs holding those of the file shown
// that fill outlets, and keeps how each outlet lays out in hosted, which
// the designs filling them take.
func hostsOf(cx *core.Scope, e *Editor, look *core.Memo[lookOf], hosted *core.Signal[map[int]hosting]) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		lk := look.Get()
		d := core.Untracked(cx, e.cur)
		name, f := e.designName(d.path), d.doc.GetUntracked()
		ds := lk.designs.with(name, f)
		return unseen("design-hosts", outletHosts(cx, ds, name, f, lk.parts, lk.assets, hosted))
	})
}

// outletHosts are the views of the designs holding those of f, named
// name, that fill outlets, found in ds, each showing the design in its
// outlet, to lay out unseen, keeping how the outlet of each design, by
// its index, lays out in hosted. A host whose outlet does not lay out, as
// one in a box hidden, leaves its design the size of its device.
func outletHosts(cx *core.Scope, ds designs, name string, f flowfile.File, parts components, pics assets, hosted *core.Signal[map[int]hosting]) []core.Node {
	var pending map[int]hosting
	record := func(i int, h hosting) {
		if pending == nil {
			pending = map[int]hosting{}
			cx.Post(func() {
				out := maps.Clone(hosted.GetUntracked())
				if out == nil {
					out = map[int]hosting{}
				}
				for i, h := range pending {
					if _, ok := out[i]; !ok || h.size != (core.Size{}) {
						out[i] = h
					}
				}
				pending = nil
				hosted.Set(out)
			})
		}
		if _, ok := pending[i]; !ok || h.size != (core.Size{}) {
			pending[i] = h
		}
	}
	var nodes []core.Node
	for i := range f.Screens {
		if _, ok := f.HostOf(i); !ok {
			continue
		}
		hops, msg := ds.hosts(place{name, i})
		if msg != "" || len(hops) == 0 {
			continue
		}
		one, shown, key := composed(ds, hops, place{name, i})
		still := stillSelection(cx, one.Screens, parts, pics)
		still.fills = func(k string) ([]flowfile.Element, bool) { return fillOf(shown, ds, k) }
		still.unseen = true
		device := one.Screens[0].Device
		var look string
		if out, ok := flowfile.ElementAt(ds[hops[0].name].Screens[hops[0].index].UI, flowfile.FormatPath(hops[0].outlet)); ok {
			look = outletLook(out.Attr("class"))
		}
		still.report = func(k string, fr frame) {
			if k == key {
				record(i, hosting{fr.Size, device, look})
			}
		}
		nodes = append(nodes, core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0)},
			OnLayout: func(core.Bounds) { record(i, hosting{device: device, look: look}) },
			Children: []core.Node{screenView(cx, one, 0, still)}})
	}
	return nodes
}
