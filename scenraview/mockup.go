package scenraview

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// selection is which element of a design is selected, by its path, how
// pressing one selects it or plays its tap, and where each element was
// laid out.
type selection struct {
	selected func() string
	choose   func(path string)
	tap      func(flowfile.Tap)
	// frames are the elements' frames in their parents', by key, which
	// laidOut counts the changes of. A key is an element's path, and in a
	// component the key of the use drawing it, a colon and its path in
	// the component.
	frames  map[string]core.Bounds
	laidOut *core.Signal[int]
	play    *player
	// parts are the components a use draws. Inside one, owner is the
	// path of the use, which its elements select, prefix the key of the
	// use and a colon, and using the components it is in, which it cannot
	// use again.
	parts  components
	owner  string
	// assets are the pictures of the folder an image's src may name.
	assets assets
	prefix string
	using  []string
	// tapping is the tap of the element an element is in, which pressing
	// it plays.
	tapping *flowfile.Tap
}

// bounds is the frame of the element at key in the design's.
func (sel selection) bounds(key string) (core.Bounds, bool) {
	if key == "" {
		return core.Bounds{}, false
	}
	var b core.Bounds
	prefix := ""
	for _, part := range strings.Split(key, ":") {
		at, err := flowfile.ParsePath(part)
		if err != nil {
			return core.Bounds{}, false
		}
		for i := range at {
			f, ok := sel.frames[prefix+flowfile.FormatPath(at[:i+1])]
			if !ok {
				return core.Bounds{}, false
			}
			b.Origin.X += f.Origin.X
			b.Origin.Y += f.Origin.Y
			b.Size = f.Size
		}
		prefix += part + ":"
	}
	return b, true
}

// hit is an element under the pointer, by its key.
type hit struct {
	key string
	el  flowfile.Element
}

// hits are the elements of els at p, from the outermost to the
// innermost, in the design's coordinates with els's parent at origin:
// the one drawn last where several overlap, and into the components
// uses draw. Children are looked in where they reach outside their
// parent, as positioned ones can, and hidden panels are passed over.
func (sel selection) hits(els []flowfile.Element, prefix string, parent []int, origin, p core.Point, using []string) []hit {
	for i := len(els) - 1; i >= 0; i-- {
		path := append(slices.Clone(parent), i)
		key := prefix + flowfile.FormatPath(path)
		f, ok := sel.frames[key]
		if !ok || !sel.play.showing(groupKey(prefix, parent), els[i]) {
			continue
		}
		at := core.Point{X: origin.X + f.Origin.X, Y: origin.Y + f.Origin.Y}
		var inner []hit
		if el := els[i]; el.Kind == "use" {
			if part, ok := sel.parts[el.Text]; ok && !slices.Contains(using, el.Text) {
				inner = sel.hits(flowfile.Instance(el, part), key+":", nil, at, p, append(slices.Clone(using), el.Text))
			}
		} else {
			inner = sel.hits(el.Children, prefix, path, at, p, using)
		}
		if inner != nil {
			return append([]hit{{key, els[i]}}, inner...)
		}
		if p.X >= at.X && p.Y >= at.Y && p.X < at.X+f.Size.Width && p.Y < at.Y+f.Size.Height {
			return []hit{{key, els[i]}}
		}
	}
	return nil
}

// taps are the keys of the elements of els with a tap that show, in the
// components uses draw too, as hits finds them.
func (sel selection) taps(els []flowfile.Element, prefix string, parent []int, using []string) []string {
	var out []string
	for i, el := range els {
		path := append(slices.Clone(parent), i)
		key := prefix + flowfile.FormatPath(path)
		if !sel.play.showing(groupKey(prefix, parent), el) {
			continue
		}
		if _, ok := el.Tap(); ok {
			out = append(out, key)
		}
		if el.Kind == "use" {
			if part, ok := sel.parts[el.Text]; ok && !slices.Contains(using, el.Text) {
				out = append(out, sel.taps(flowfile.Instance(el, part), key+":", nil, append(slices.Clone(using), el.Text))...)
			}
		} else {
			out = append(out, sel.taps(el.Children, prefix, path, using)...)
		}
	}
	return out
}

// target is what pressing the elements hit selects, the innermost
// element of the design, and the innermost tap among them, if any.
func target(hits []hit) (path string, tap *flowfile.Tap) {
	for _, h := range hits {
		if !strings.Contains(h.key, ":") {
			path = h.key
		}
		if t, ok := h.el.Tap(); ok {
			tap = &t
		}
	}
	return path, tap
}

// groupKey is the key of the element the elements at parent are in, by
// which its panels take turns: the use drawing them at a component's
// top, "" at the design's.
func groupKey(prefix string, parent []int) string {
	if len(parent) == 0 {
		return strings.TrimSuffix(prefix, ":")
	}
	return prefix + flowfile.FormatPath(parent)
}

// player plays a design: which of its panels show, and which taps lead
// to what shows.
type player struct {
	// tabs is the panel shown among the panels of each element, by its
	// key, while the user has shown another than the first; panels the
	// panels' names there, in order, as the design was last drawn.
	tabs   *core.Signal[map[string]string]
	panels map[string][]string
	// shown is the name of the design shown, which taps opening it
	// lead to.
	shown func() string
}

func newPlayer(cx *core.Scope, shown func() string) *player {
	return &player{tabs: core.NewSignalFunc(cx, map[string]string(nil), maps.Equal[map[string]string]), shown: shown}
}

// scan finds the panels of els, at the design's top when prefix is "",
// and in the components they use.
func (pl *player) scan(els []flowfile.Element, parts components) {
	pl.panels = map[string][]string{}
	var walk func(els []flowfile.Element, prefix string, parent []int, using []string)
	walk = func(els []flowfile.Element, prefix string, parent []int, using []string) {
		for i, el := range els {
			path := append(slices.Clone(parent), i)
			if name := el.Attr("panel"); name != "" {
				g := groupKey(prefix, parent)
				if !slices.Contains(pl.panels[g], name) {
					pl.panels[g] = append(pl.panels[g], name)
				}
			}
			if part, ok := parts[el.Text]; el.Kind == "use" && ok && !slices.Contains(using, el.Text) {
				walk(flowfile.Instance(el, part), prefix+flowfile.FormatPath(path)+":", nil, append(slices.Clone(using), el.Text))
			}
			walk(el.Children, prefix, path, using)
		}
	}
	walk(els, "", nil, nil)
}

// showing says el, in the element at the key group, shows: it is no
// panel, or the panel shown there, the first unless a tap showed another.
func (pl *player) showing(group string, el flowfile.Element) bool {
	name := el.Attr("panel")
	if pl == nil || name == "" {
		return true
	}
	names := pl.panels[group]
	shown := pl.tabs.Get()[group]
	if !slices.Contains(names, shown) && len(names) > 0 {
		shown = names[0]
	}
	return shown == name
}

// leadsTo says tap leads to what shows: it opens the design shown, or
// shows a tab that shows.
func (pl *player) leadsTo(t flowfile.Tap) bool {
	switch t.Action {
	case flowfile.TapOpen:
		return t.Target == pl.shown()
	case flowfile.TapTab:
		for g, names := range pl.panels {
			if slices.Contains(names, t.Target) && pl.showing(g, flowfile.Element{Attrs: map[string]string{"panel": t.Target}}) {
				return true
			}
		}
	}
	return false
}

// showTab shows the panels named name in place of their siblings.
func (pl *player) showTab(name string) {
	tabs := maps.Clone(pl.tabs.GetUntracked())
	if tabs == nil {
		tabs = map[string]string{}
	}
	for g, names := range pl.panels {
		if slices.Contains(names, name) {
			tabs[g] = name
		}
	}
	pl.tabs.Set(tabs)
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
// pointer or plays its tap; its own press is for the keyboard and
// assistive technology. A panel shows while its tab does, and a box
// whose tap leads to what shows is selected. An image shows the picture
// of the assets folder or the URL its src names; one without a src, one
// of a picture the folder lacks, an SVG NSGo cannot read, or a use of a component that is
// missing or in which it is, draws a placeholder. A use draws its
// component with what it changes, and the elements of a component
// select the use.
func element(cx *core.Scope, el flowfile.Element, path []int, sel selection) core.Node {
	class := el.Attr("class")
	key := sel.prefix + flowfile.FormatPath(path)
	layout := func(b core.Bounds) {
		sel.frames[key] = b
		sel.laidOut.Update(func(n int) int { return n + 1 })
	}
	id, testID := key, "element-"+key
	if sel.owner != "" {
		id = sel.owner
	}
	var selected func() bool
	if t, ok := el.Tap(); ok {
		sel.tapping = &t
		selected = func() bool { return sel.play.leadsTo(t) }
	}
	press := func(ev core.PressEvent) {
		if sel.tapping != nil && !ev.Modifiers.Alt {
			sel.tap(*sel.tapping)
			return
		}
		sel.choose(id)
	}
	var hide func() core.Style
	if el.Attr("panel") != "" {
		group := groupKey(sel.prefix, path[:len(path)-1])
		hide = func() core.Style {
			if sel.play.showing(group, el) {
				return core.Style{}
			}
			return core.Style{Display: core.DisplayNone}
		}
	}
	placeholder := func(icon func() *core.SVG) core.Node {
		return core.View{Class: "h-40 items-center justify-center overflow-hidden bg-muted " + class, Label: el.Text,
			OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide,
			Children: []core.Node{core.Icon{SVG: icon(), Class: "size-8 text-muted-foreground"}}}
	}
	switch el.Kind {
	case "box":
		return core.View{Class: class, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide, Selected: selected,
			Children: elements(cx, el.Children, path, sel)}
	case "text":
		return core.Text{Content: el.Text, Class: class, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide}
	case "icon":
		return core.Icon{SVG: iconNamed(el.Text)(), Class: class, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide}
	case "image":
		src := el.Attr("src")
		if src == "" {
			return placeholder(lucide.Image)
		}
		source := core.Src(src)
		if p, ok := flowfile.AssetPath(src); ok {
			pic, ok := sel.assets[p]
			if !ok {
				return placeholder(lucide.ImageOff)
			}
			source = core.ImageSource{Data: pic.data}
		}
		return core.Image{Source: source, ResizeMode: core.Cover, Class: "h-40 overflow-hidden bg-muted " + class, Label: el.Text,
			OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide}
	case "svg":
		doc, err := core.ParseSVG([]byte(el.Text))
		if err != nil {
			return placeholder(lucide.ImageOff)
		}
		return core.SVGView{SVG: doc, Class: class, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide}
	case "use":
		els, ok := sel.parts[el.Text]
		if !ok || slices.Contains(sel.using, el.Text) {
			return placeholder(lucide.Component)
		}
		inner := sel
		inner.owner = id
		inner.prefix = key + ":"
		inner.using = append(slices.Clone(sel.using), el.Text)
		return core.View{Class: class, Label: el.Text, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide, Selected: selected,
			Children: elements(cx, flowfile.Instance(el, els), nil, inner)}
	}
	return core.Group{}
}

// componentProblems are what is wrong with the components els use: those
// missing from the folder, those used inside themselves, and what uses
// change in them that is not there. self is the name of the component
// els are, "" for a design.
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
	walk(els, func(el flowfile.Element) {
		if el.Kind == "use" {
			out = append(out, flowfile.OverrideProblems(el, func(name string) ([]flowfile.Element, bool) {
				part, ok := parts[name]
				return part, ok
			})...)
		}
	})
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
