package scenraview

import (
	"fmt"
	"slices"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// selection is which element of a design is selected, by its path, how
// pressing one selects it, and where each element was laid out.
type selection struct {
	selected func() string
	choose   func(path string)
	// frames are the elements' frames in their parents', by path, which
	// laidOut counts the changes of.
	frames  map[string]core.Bounds
	laidOut *core.Signal[int]
	// parts are the components a use draws. Inside one, owner is the
	// path of the use, which its elements select, and using the
	// components it is in, which it cannot use again.
	parts components
	owner string
	using []string
}

// bounds is the frame of the element at path in the design's.
func (sel selection) bounds(path string) (core.Bounds, bool) {
	at, err := flowfile.ParsePath(path)
	if path == "" || err != nil {
		return core.Bounds{}, false
	}
	var b core.Bounds
	for i := range at {
		f, ok := sel.frames[flowfile.FormatPath(at[:i+1])]
		if !ok {
			return core.Bounds{}, false
		}
		b.Origin.X += f.Origin.X
		b.Origin.Y += f.Origin.Y
		b.Size = f.Size
	}
	return b, true
}

// elementAt is the path of the innermost element of els at p, in the
// design's coordinates with els's parent at origin, the one drawn last
// where several overlap, or "" for none. Children are looked in where
// they reach outside their parent, as positioned ones can.
func (sel selection) elementAt(els []flowfile.Element, parent []int, origin, p core.Point) string {
	for i := len(els) - 1; i >= 0; i-- {
		path := append(slices.Clone(parent), i)
		f, ok := sel.frames[flowfile.FormatPath(path)]
		if !ok {
			continue
		}
		at := core.Point{X: origin.X + f.Origin.X, Y: origin.Y + f.Origin.Y}
		if hit := sel.elementAt(els[i].Children, path, at, p); hit != "" {
			return hit
		}
		if p.X >= at.X && p.Y >= at.Y && p.X < at.X+f.Size.Width && p.Y < at.Y+f.Size.Height {
			return flowfile.FormatPath(path)
		}
	}
	return ""
}

// screenView draws a design at its size: its elements stack in its
// frame, which clips what overflows. A component without a height is as
// high as it needs.
func screenView(cx *core.Scope, s flowfile.File, sel selection) core.Node {
	colors := ui.ColorsOf(cx)
	w, h := s.Size()
	var height core.Length
	if h != 0 {
		height = core.Pt(core.Points(h))
	}
	radius := core.Points(44)
	switch {
	case s.Kind == flowfile.Component:
		radius = 12
	case s.Device == "tablet":
		radius = 28
	case s.Device == "desktop":
		radius = 12
	}
	return core.View{Style: core.Style{Width: core.Pt(core.Points(w)), Height: height, CornerRadius: radius, FlexShrink: 0,
		BorderCurve: core.BorderCurveContinuous, Overflow: core.Hidden, BorderWidth: 1, FontSize: 15},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Background(), Color: colors.Text(), BorderColor: colors.Border()}
		},
		Children: elements(cx, s.UI, nil, sel)}
}

func elements(cx *core.Scope, els []flowfile.Element, parent []int, sel selection) []core.Node {
	out := make([]core.Node, len(els))
	for i, el := range els {
		out[i] = element(cx, el, append(slices.Clone(parent), i), sel)
	}
	return out
}

// element draws the element at path, shaped by its classes, recording
// where it is laid out. The design's pointer layer selects it under the
// pointer; its own press is for the keyboard and assistive technology.
// An image without a picture, an SVG NSGo cannot read, or a use of a
// component that is missing or in which it is, draws a placeholder. The
// elements of a component are the use's, so they record nothing and
// select it.
func element(cx *core.Scope, el flowfile.Element, path []int, sel selection) core.Node {
	class := el.Attr("class")
	id := flowfile.FormatPath(path)
	var layout func(core.Bounds)
	testID := ""
	if sel.owner == "" {
		layout = func(b core.Bounds) {
			sel.frames[id] = b
			sel.laidOut.Update(func(n int) int { return n + 1 })
		}
		testID = "element-" + id
	} else {
		id = sel.owner
	}
	press := func(core.PressEvent) { sel.choose(id) }
	placeholder := func(icon func() *core.SVG) core.Node {
		return core.View{Class: "h-40 items-center justify-center overflow-hidden bg-muted " + class, Label: el.Text,
			OnPress: press, OnLayout: layout, TestID: testID,
			Children: []core.Node{core.Icon{SVG: icon(), Class: "size-8 text-muted-foreground"}}}
	}
	switch el.Kind {
	case "box":
		return core.View{Class: class, OnPress: press, OnLayout: layout, TestID: testID, Children: elements(cx, el.Children, path, sel)}
	case "text":
		return core.Text{Content: el.Text, Class: class, OnPress: press, OnLayout: layout, TestID: testID}
	case "icon":
		return core.Icon{SVG: iconNamed(el.Text)(), Class: class, OnPress: press, OnLayout: layout, TestID: testID}
	case "image":
		if src := el.Attr("src"); src != "" {
			return core.Image{Source: core.Src(src), ResizeMode: core.Cover, Class: "h-40 overflow-hidden bg-muted " + class, Label: el.Text,
				OnPress: press, OnLayout: layout, TestID: testID}
		}
		return placeholder(lucide.Image)
	case "svg":
		doc, err := core.ParseSVG([]byte(el.Text))
		if err != nil {
			return placeholder(lucide.ImageOff)
		}
		return core.SVGView{SVG: doc, Class: class, OnPress: press, OnLayout: layout, TestID: testID}
	case "use":
		els, ok := sel.parts[el.Text]
		if !ok || slices.Contains(sel.using, el.Text) {
			return placeholder(lucide.Component)
		}
		inner := sel
		inner.owner = id
		inner.using = append(slices.Clone(sel.using), el.Text)
		return core.View{Class: class, Label: el.Text, OnPress: press, OnLayout: layout, TestID: testID, Children: elements(cx, els, nil, inner)}
	}
	return core.Group{}
}

// componentProblems are what is wrong with the components els use: those
// missing from the folder, and those used inside themselves. self is the
// name of the component els are, "" for a design.
func componentProblems(els []flowfile.Element, self string, parts components) []string {
	var out []string
	var visit func(name string, using []string) bool
	visit = func(name string, using []string) bool {
		if slices.Contains(using, name) {
			return false
		}
		for _, n := range flowfile.Uses(parts[name]) {
			if !visit(n, slices.Concat(using, []string{name})) {
				return false
			}
		}
		return true
	}
	var using []string
	if self != "" {
		using = []string{self}
	}
	for _, name := range flowfile.Uses(els) {
		if _, ok := parts[name]; !ok {
			out = append(out, fmt.Sprintf("use %q: there is no %s", name, flowfile.ComponentPath(name)))
		} else if !visit(name, using) {
			out = append(out, fmt.Sprintf("use %q: components use each other in a loop", name))
		}
	}
	return out
}

// unknownClasses are the classes of a design's elements that draw
// nothing.
func unknownClasses(els []flowfile.Element) []string {
	var out []string
	var walk func(els []flowfile.Element)
	walk = func(els []flowfile.Element) {
		for _, el := range els {
			for _, c := range core.UnknownClasses(el.Attr("class")) {
				if !slices.Contains(out, c) {
					out = append(out, c)
				}
			}
			walk(el.Children)
		}
	}
	walk(els)
	return out
}
