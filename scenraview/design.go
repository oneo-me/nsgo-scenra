package scenraview

import (
	"math"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// The design canvas's zoom limits and steps, as the flow chart's.
const (
	designMinZoom   = .1
	designMaxZoom   = 4
	designZoomStep  = 1.25
	designFitMargin = 32
	// designWheelZoom and designPinchZoom are the scroll distances in
	// points that zoom by ×1.5: a wheel's notches, and a touchpad's scroll
	// with Control or Command held.
	designWheelZoom = 120
	designPinchZoom = 48
)

// designCanvas is the camera over a design: the design is laid
// out at its device's size on a plane that the camera moves and scales,
// fitted and centered until the user pans or zooms it.
type designCanvas struct {
	cx         *core.Scope
	x, y, zoom *core.AnimatedValue
	cam        struct{ x, y, zoom float64 }
	// zoomed is where the camera's zoom is going, for what keeps its
	// width on screen at any zoom.
	zoomed *core.Signal[float64]
	// view is where the camera is going, x, y and zoom, for what is drawn
	// over the design at a point of it.
	view               *core.Signal[[3]float64]
	width, height      float64
	contentW, contentH float64
	moved              bool
	// pan is where the camera was when a drag began, and pinchFrom the
	// pinch's scale at its last change.
	pan       [2]float64
	pinchFrom float64
	run       *core.AnimationRun
}

func newDesignCanvas(cx *core.Scope) *designCanvas {
	c := &designCanvas{cx: cx, x: core.NewAnimatedValue(cx, 0), y: core.NewAnimatedValue(cx, 0), zoom: core.NewAnimatedValue(cx, 1),
		zoomed: core.NewSignal(cx, 1.0), view: core.NewSignal(cx, [3]float64{0, 0, 1})}
	c.cam.zoom = 1
	return c
}

// move sends the camera to x, y and zoom, at once or easing there.
func (c *designCanvas) move(x, y, zoom float64, animated bool) {
	c.cam.x, c.cam.y, c.cam.zoom = x, y, zoom
	c.zoomed.Set(zoom)
	c.view.Set([3]float64{x, y, zoom})
	if c.run != nil {
		c.run.Stop()
	}
	if !animated || core.Untracked(c.cx, c.cx.Env().ReduceMotion) {
		c.x.Set(x)
		c.y.Set(y)
		c.zoom.Set(zoom)
		return
	}
	spring := core.SpringConfig{Stiffness: 666, Damping: 71, Mass: 1}
	c.run = c.cx.StartAnimation(core.Parallel(core.Spring(c.x, x, spring), core.Spring(c.y, y, spring), core.Spring(c.zoom, zoom, spring)), nil)
}

// fit shows the whole design centered, no larger than its size.
func (c *designCanvas) fit(animated bool) {
	if c.width == 0 || c.contentW == 0 {
		return
	}
	zoom := math.Max(designMinZoom, math.Min(1, math.Min((c.width-2*designFitMargin)/c.contentW, (c.height-2*designFitMargin)/c.contentH)))
	c.moved = false
	c.move((c.width-c.contentW*zoom)/2, (c.height-c.contentH*zoom)/2, zoom, animated)
}

// zoomAt scales the design around a point of the view.
func (c *designCanvas) zoomAt(x, y, factor float64, animated bool) {
	zoom := math.Max(designMinZoom, math.Min(designMaxZoom, c.cam.zoom*factor))
	gx, gy := (x-c.cam.x)/c.cam.zoom, (y-c.cam.y)/c.cam.zoom
	c.moved = true
	c.move(x-gx*zoom, y-gy*zoom, zoom, animated)
}

func (c *designCanvas) zoomBy(factor float64) { c.zoomAt(c.width/2, c.height/2, factor, true) }

// toView is the point of the view over a point of the design, where the
// camera is going.
func (c *designCanvas) toView(p core.Point) core.Point {
	v := c.view.Get()
	return core.Point{X: core.Points(v[0] + float64(p.X)*v[2]), Y: core.Points(v[1] + float64(p.Y)*v[2])}
}

// toDesign is the point of the design under a point of the view.
func (c *designCanvas) toDesign(p core.Point) core.Point {
	return core.Point{X: core.Points((float64(p.X) - c.cam.x) / c.cam.zoom), Y: core.Points((float64(p.Y) - c.cam.y) / c.cam.zoom)}
}

// resize keeps a design the user has not moved fitted, and otherwise
// keeps the view's center where it is.
func (c *designCanvas) resize(b core.Bounds) {
	w, h := float64(b.Size.Width), float64(b.Size.Height)
	if w == c.width && h == c.height {
		return
	}
	dx, dy := (w-c.width)/2, (h-c.height)/2
	first := c.width == 0
	c.width, c.height = w, h
	if first || !c.moved {
		c.fit(false)
		return
	}
	c.move(c.cam.x+dx, c.cam.y+dy, c.cam.zoom, false)
}

// measured takes the design's size, fitting it while the user has not
// moved it.
func (c *designCanvas) measured(b core.Bounds) {
	w, h := float64(b.Size.Width), float64(b.Size.Height)
	if w == c.contentW && h == c.contentH {
		return
	}
	c.contentW, c.contentH = w, h
	if !c.moved {
		c.fit(false)
	}
}

// wheel pans and zooms as the flow chart does: a touchpad's two fingers
// pan unless Control or Command is held, which zooms, and a wheel zooms.
func (c *designCanvas) wheel(e core.WheelEvent) {
	modified := e.Modifiers.Control || e.Modifiers.Platform
	switch {
	case e.Phase == core.GestureEnded || e.Phase == core.GestureCancelled:
	case e.Touchpad && !modified:
		c.moved = true
		c.move(c.cam.x+float64(e.Delta.X), c.cam.y+float64(e.Delta.Y), c.cam.zoom, false)
	case !e.Touchpad && !modified && e.Delta.Y == 0:
		c.moved = true
		c.move(c.cam.x+float64(e.Delta.X), c.cam.y, c.cam.zoom, false)
	default:
		distance := float64(designWheelZoom)
		if e.Touchpad {
			distance = designPinchZoom
		}
		c.zoomAt(float64(e.Position.X), float64(e.Position.Y), math.Exp(float64(e.Delta.Y)*math.Log(1.5)/distance), false)
	}
}

// lookOf is what a design is drawn from: its text, and the components it
// may use.
type lookOf struct {
	text  string
	parts components
}

// peekDelay is how long the pointer rests on an element before its card
// shows, or off it before the card goes.
const peekDelay = 600 * time.Millisecond

// designPane is the design shown on the canvas, with its zoom
// tools, played as the app would be. A layer over the design takes the
// pointer and finds the element under it itself, as the camera's
// transform moves the design where the platforms' hit testing does not
// follow: an outline marks the element the pointer is over, pressing an
// element with a tap plays it, as do those it holds, and pressing another
// selects it, as Alt does any, and pressing around it clears the
// selection. Pins mark the elements the user commented on or noted, and
// a card edits the comment and note of the element the pointer rests on
// or selected, beside where it pressed or else beside the element; ask,
// when set, sends the assistant a prompt about them.
func designPane(cx *core.Scope, e *Editor, ask func(prompt string)) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	fin := ui.FinOf(cx)
	c := newDesignCanvas(cx)
	// A design is drawn again as it or a component changes.
	look := core.NewMemoFunc(cx, func() lookOf { return lookOf{e.cur().doc.Get().Format(), e.parts.Get()} },
		func(a, b lookOf) bool { return a.text == b.text && sameComponents(a.parts, b.parts) })
	play := newPlayer(cx, func() string { return e.designName(e.cur().path) })
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
	// Another design shows fitted, its tabs at their first, and no card.
	shown := core.NewMemo(cx, e.cur)
	cx.Effect(func() {
		shown.Get()
		core.Untracked(cx, func() bool {
			play.tabs.Set(nil)
			peek.Set("")
			clicked.Set(click{})
			return true
		})
		cx.Post(func() { c.fit(false) })
	})
	sel := selection{
		selected: func() string { return e.cur().element.Get() },
		choose: func(path string) {
			cx.Batch(func() {
				e.shown().element.Set(path)
				peek.Set(path)
			})
		},
		tap: func(t flowfile.Tap) {
			peek.Set("")
			switch t.Action {
			case flowfile.TapOpen:
				e.tapOpen(t.Target)
			case flowfile.TapClose:
				e.tapClose()
			case flowfile.TapTab:
				play.showTab(t.Target)
			}
		},
		frames:  map[string]core.Bounds{},
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
				w := core.Points(width / c.zoomed.Get())
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
	// targetAt is the element of the design under a point of the view and
	// the tap pressing there plays.
	targetAt := func(p core.Point) (string, *flowfile.Tap) {
		s := e.cur().doc.Get()
		at := c.toDesign(p)
		if at.X < 0 || at.Y < 0 || float64(at.X) >= c.contentW || float64(at.Y) >= c.contentH {
			return "", nil
		}
		return target(sel.hits(s.UI, "", nil, core.Point{}, at, nil))
	}
	// A changed design is laid out afresh, the pointer over nothing until
	// it moves.
	cx.Effect(func() {
		look.Get()
		hovered.Set("")
		tappable.Set(false)
	})
	picture := core.Dynamic(func(cx *core.Scope) core.Node {
		sel.parts = look.Get().parts
		s := core.Untracked(cx, func() flowfile.File { return e.cur().doc.Get() })
		clear(sel.frames)
		play.scan(s.UI, sel.parts)
		if len(s.UI) == 0 {
			return core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0),
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 10}, Children: []core.Node{
				ui.Icon(lucide.PanelsTopLeft, 28, colors.Muted),
				core.Text{Content: tr.DesignEmpty(), Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(320)},
					StyleFunc: ui.Colored(colors.Muted)}}}
		}
		return core.View{TestID: "design", Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0),
			TransformOrigin: core.ObjectPosition{X: core.PositionAxis{Offset: core.Pt(0)}, Y: core.PositionAxis{Offset: core.Pt(0)}},
			Animated:        &core.AnimatedStyle{TranslateX: c.x, TranslateY: c.y, Scale: c.zoom}},
			OnLayout: c.measured,
			Children: append(append([]core.Node{screenView(cx, s, sel)}, remarkPins(cx, s, sel, c.zoomed.Get)...), hover, selected)}
	})
	// comments counts the design's comments.
	comments := func() int { return len(e.cur().doc.Get().Comments()) }
	// anchor is the point of the view the card opens beside.
	anchor := func() (core.Point, bool) {
		path := peek.Get()
		if k := clicked.Get(); k.path == path {
			return c.toView(k.at), true
		}
		sel.laidOut.Get()
		b, ok := sel.bounds(path)
		return c.toView(core.Point{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y}), ok
	}
	// place keeps a card of size beside the anchor and inside the view.
	place := func(size core.Size) (core.Point, bool) {
		p, ok := anchor()
		if !ok {
			return core.Point{}, false
		}
		const gap, margin = 12, 8
		w, h := core.Points(c.width), core.Points(c.height)
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
			tool(lucide.ZoomOut, tr.ZoomOut(), func() { c.zoomBy(1 / designZoomStep) }, "design-zoom-out"),
			tool(lucide.ZoomIn, tr.ZoomIn(), func() { c.zoomBy(designZoomStep) }, "design-zoom-in"),
			tool(lucide.Scan, tr.Fit(), func() { c.fit(true) }, "design-fit"),
			core.Show{When: func() bool { return ask != nil && comments() > 0 && !e.Working() }, Then: func(*core.Scope) core.Node {
				return tool(lucide.MessagesSquare, tr.ResolveComments(), func() { ask(tr.ResolveCommentsPrompt(core.Untracked(cx, comments))) }, "resolve-comments")
			}},
		}}
	pointer := core.View{TestID: "design-pointer", NoFocus: true, ImportantForAccessibility: core.ImportantNo,
		Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)},
		StyleFunc: func() core.Style {
			if tappable.Get() {
				return core.Style{Cursor: core.CursorPointer}
			}
			return core.Style{}
		},
		OnPress: func(ev core.PressEvent) {
			path, tap := targetAt(ev.Position)
			if tap != nil && !ev.Modifiers.Alt {
				sel.tap(*tap)
				return
			}
			clicked.Set(click{path, c.toDesign(ev.Position)})
			sel.choose(path)
		},
		OnMouseMove: func(ev core.MouseEvent) {
			path, tap := "", (*flowfile.Tap)(nil)
			if ev.PressedButton == core.MouseButtonNone {
				path, tap = targetAt(ev.Position)
			}
			cx.Batch(func() {
				hovered.Set(path)
				tappable.Set(tap != nil)
			})
		},
		OnMouseLeave: func(core.MouseEvent) {
			cx.Batch(func() {
				hovered.Set("")
				tappable.Set(false)
			})
		},
		OnWheel:      c.wheel,
		OnPan: func(ev core.PanEvent) {
			switch ev.Phase {
			case core.GestureBegan:
				c.moved = true
				c.pan = [2]float64{c.cam.x, c.cam.y}
			case core.GestureChanged:
				c.move(c.pan[0]+float64(ev.Translation.X), c.pan[1]+float64(ev.Translation.Y), c.cam.zoom, false)
			}
		},
		OnPinch: func(ev core.GestureEvent) {
			switch ev.Phase {
			case core.GestureBegan:
				c.pinchFrom = 1
			case core.GestureChanged:
				if c.pinchFrom > 0 && ev.Value > 0 {
					c.zoomAt(float64(ev.Position.X), float64(ev.Position.Y), float64(ev.Value)/c.pinchFrom, false)
					c.pinchFrom = float64(ev.Value)
				}
			}
		},
	}
	return core.View{Style: ui.FillRest.Merge(core.Style{Overflow: core.Hidden}), StyleFunc: ui.Filled(colors.Inset), NoFocus: true,
		OnLayout: c.resize,
		Children: []core.Node{picture, pointer, toolbar,
			// The user cannot comment while the agent works.
			core.Show{When: func() bool { return peek.Get() != "" && !e.Working() }, Then: func(cx *core.Scope) core.Node {
				return remarkCard(cx, e, remarkOptions{path: peek.Get, ask: ask, place: place, over: overCard.Set, held: held.Set})
			}},
		}}
}
