package scenraview

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// selection is which element of a file's designs is selected, by its
// key, how pressing one selects it or plays its tap, and where each
// element was laid out.
type selection struct {
	selected func() string
	choose   func(key string)
	// tap plays a tap of the design at index screen.
	tap func(t flowfile.Tap, screen int)
	// frames are the elements' frames in their parents', by key, which
	// laidOut counts the changes of, and origins where each design is in
	// the plane. A key is the index of the element's design, a slash and
	// its path, and in a component the key of the use drawing it, a colon
	// and its path in the component.
	frames  map[string]frame
	origins []core.Point
	laidOut *core.Signal[int]
	play    *player
	// screen is the index of the design drawn.
	screen int
	// parts are the components a use draws.
	parts components
	// assets are the pictures of the folder an image's src may name.
	assets assets
	scope
	// holder is the key of the element whose view the elements drawn are
	// in, "" at the top of a design.
	holder string
	// tapping is the tap of the element an element is in, which pressing
	// it plays.
	tapping *flowfile.Tap
}

// frame is where an element was laid out in the view of its holder, by
// its key, which is not its parent's when a slot draws it.
type frame struct {
	core.Bounds
	holder string
}

// scope is where elements are drawn: prefix starts their keys. Inside a
// component, owner is the key of the design's use, which its elements
// select, using the components drawn around them, which they cannot use
// again, and slots the fills of the use drawing it.
type scope struct {
	prefix string
	owner  string
	using  []string
	slots  *slots
}

// slots are the fills of the use at path in outer, which the slots of
// its component draw in outer, where the fills are written.
type slots struct {
	path  []int
	fills []flowfile.Element
	outer scope
}

// inside returns what el, at path in sc, draws inside it: the elements,
// the scope they are in and the path they are under there; ok is false
// for a use of a missing component or of one it is in. A use draws its
// component, and a slot what the use's fill of it holds, or else its
// own.
func (sc scope) inside(el flowfile.Element, path []int, parts components) (els []flowfile.Element, in scope, under []int, ok bool) {
	switch el.Kind {
	case "use":
		part, ok := parts[el.Text]
		if !ok || slices.Contains(sc.using, el.Text) {
			return nil, sc, nil, false
		}
		key := sc.prefix + flowfile.FormatPath(path)
		owner := sc.owner
		if owner == "" {
			owner = key
		}
		return flowfile.Instance(el, part), scope{prefix: key + ":", owner: owner, using: append(slices.Clone(sc.using), el.Text),
			slots: &slots{path: path, fills: el.Children, outer: sc}}, nil, true
	case "slot":
		if sc.slots != nil {
			if i := slices.IndexFunc(sc.slots.fills, func(f flowfile.Element) bool { return f.Text == el.Text }); i >= 0 {
				return sc.slots.fills[i].Children, sc.slots.outer, append(slices.Clone(sc.slots.path), i), true
			}
		}
	}
	return el.Children, sc, path, true
}

// elementKey is the key of the element at path of the design at index
// screen, and splitKey its parts.
func elementKey(screen int, path string) string { return screenPrefix(screen) + path }

func splitKey(key string) (screen int, path string, ok bool) {
	head, path, ok := strings.Cut(key, "/")
	n, err := strconv.Atoi(head)
	if !ok || err != nil || n < 0 || path == "" {
		return 0, "", false
	}
	return n, path, true
}

// screenPrefix starts the keys of the design at index screen.
func screenPrefix(screen int) string { return strconv.Itoa(screen) + "/" }

// bounds is the frame of the element at key in the plane of the designs.
func (sel selection) bounds(key string) (core.Bounds, bool) {
	screen, _, ok := splitKey(key)
	if !ok || screen >= len(sel.origins) {
		return core.Bounds{}, false
	}
	f, ok := sel.frames[key]
	if !ok {
		return core.Bounds{}, false
	}
	b := f.Bounds
	b.Origin.X += sel.origins[screen].X
	b.Origin.Y += sel.origins[screen].Y
	for f.holder != "" {
		if f, ok = sel.frames[f.holder]; !ok {
			return core.Bounds{}, false
		}
		b.Origin.X += f.Origin.X
		b.Origin.Y += f.Origin.Y
	}
	return b, true
}

// hit is an element under the pointer, by its key.
type hit struct {
	key string
	el  flowfile.Element
}

// hits are the elements of els, under parent in sc, at p, from the
// outermost to the innermost, in the design's coordinates with their
// holder at origin: the one drawn last where several overlap, and into
// what uses and slots draw. Children are looked in where they reach
// outside their parent, as positioned ones can, and hidden panels are
// passed over.
func (sel selection) hits(els []flowfile.Element, sc scope, parent []int, origin, p core.Point) []hit {
	for i := len(els) - 1; i >= 0; i-- {
		path := append(slices.Clone(parent), i)
		key := sc.prefix + flowfile.FormatPath(path)
		f, ok := sel.frames[key]
		if !ok || !sel.play.showing(groupKey(sc.prefix, parent), els[i]) {
			continue
		}
		at := core.Point{X: origin.X + f.Origin.X, Y: origin.Y + f.Origin.Y}
		var inner []hit
		if kids, in, under, ok := sc.inside(els[i], path, sel.parts); ok {
			inner = sel.hits(kids, in, under, at, p)
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

// taps are the keys of the elements of els, under parent in sc, with a
// tap that show, in what uses and slots draw too, as hits finds them.
func (sel selection) taps(els []flowfile.Element, sc scope, parent []int) []string {
	var out []string
	for i, el := range els {
		path := append(slices.Clone(parent), i)
		if !sel.play.showing(groupKey(sc.prefix, parent), el) {
			continue
		}
		if _, ok := el.Tap(); ok {
			out = append(out, sc.prefix+flowfile.FormatPath(path))
		}
		if kids, in, under, ok := sc.inside(el, path, sel.parts); ok {
			out = append(out, sel.taps(kids, in, under)...)
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
// top, the design's prefix at its top.
func groupKey(prefix string, parent []int) string {
	if len(parent) == 0 {
		return strings.TrimSuffix(prefix, ":")
	}
	return prefix + flowfile.FormatPath(parent)
}

// player plays the designs of a file: which of their panels show, and
// which taps lead to what shows.
type player struct {
	// tabs is the panel shown among the panels of each element, by its
	// key, while the user has shown another than the first; panels the
	// panels' names there, in order, as the designs were last drawn.
	tabs   *core.Signal[map[string]string]
	panels map[string][]string
	// file is the name of the file shown, which taps opening its designs
	// name, and ids its designs' IDs, as last drawn.
	file func() string
	ids  []string
}

func newPlayer(cx *core.Scope, file func() string) *player {
	return &player{tabs: core.NewSignalFunc(cx, map[string]string(nil), maps.Equal[map[string]string]), file: file}
}

// scan finds the panels of the designs, and of the components they use.
func (pl *player) scan(screens []flowfile.Screen, parts components) {
	pl.panels = map[string][]string{}
	pl.ids = nil
	var walk func(els []flowfile.Element, sc scope, parent []int)
	walk = func(els []flowfile.Element, sc scope, parent []int) {
		for i, el := range els {
			path := append(slices.Clone(parent), i)
			if name := el.Attr("panel"); name != "" {
				g := groupKey(sc.prefix, parent)
				if !slices.Contains(pl.panels[g], name) {
					pl.panels[g] = append(pl.panels[g], name)
				}
			}
			if kids, in, under, ok := sc.inside(el, path, parts); ok {
				walk(kids, in, under)
			}
		}
	}
	for i, s := range screens {
		pl.ids = append(pl.ids, s.ID)
		walk(s.UI, scope{prefix: screenPrefix(i)}, nil)
	}
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

// leadsTo says tap, in the design at index screen, leads to what shows:
// it opens that design, or shows a tab that shows there.
func (pl *player) leadsTo(t flowfile.Tap, screen int) bool {
	switch t.Action {
	case flowfile.TapOpen:
		file, id := t.Opens()
		if file != "" && file != pl.file() || screen >= len(pl.ids) {
			return false
		}
		return id == pl.ids[screen] || id == "" && screen == 0
	case flowfile.TapTab:
		for g, names := range pl.panels {
			if strings.HasPrefix(g, screenPrefix(screen)) && slices.Contains(names, t.Target) &&
				pl.showing(g, flowfile.Element{Attrs: map[string]string{"panel": t.Target}}) {
				return true
			}
		}
	}
	return false
}

// showTab shows the panels named name in the design at index screen in
// place of their siblings.
func (pl *player) showTab(name string, screen int) {
	tabs := maps.Clone(pl.tabs.GetUntracked())
	if tabs == nil {
		tabs = map[string]string{}
	}
	for g, names := range pl.panels {
		if strings.HasPrefix(g, screenPrefix(screen)) && slices.Contains(names, name) {
			tabs[g] = name
		}
	}
	pl.tabs.Set(tabs)
}

// screenView draws the design of f at index i at its size: its elements
// stack in its frame, which clips what overflows. A component without a
// height is as high as it needs.
func screenView(cx *core.Scope, f flowfile.File, i int, sel selection) core.Node {
	colors := ui.ColorsOf(cx)
	w, h := f.Size(i)
	var height core.Length
	if h != 0 {
		height = core.Pt(core.Points(h))
	}
	radius := core.Points(44)
	switch {
	case f.Kind == flowfile.Component:
		radius = 12
	case f.Screens[i].Device == "tablet":
		radius = 28
	case f.Screens[i].Device == "desktop":
		radius = 12
	}
	sel.screen, sel.prefix = i, screenPrefix(i)
	return core.View{Style: core.Style{Width: core.Pt(core.Points(w)), Height: height, CornerRadius: radius, FlexShrink: 0,
		BorderCurve: core.BorderCurveContinuous, Overflow: core.Hidden, BorderWidth: 1, FontSize: 15},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Background(), Color: colors.Text(), BorderColor: colors.Border()}
		},
		Children: elements(cx, f.Screens[i].UI, nil, sel)}
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
// select the use; a slot draws what the use's fill of it holds, which
// selects itself, or else its own, and in the component's own file an
// empty slot shows where it is.
func element(cx *core.Scope, el flowfile.Element, path []int, sel selection) core.Node {
	class := el.Attr("class")
	key := sel.prefix + flowfile.FormatPath(path)
	layout := func(b core.Bounds) {
		sel.frames[key] = frame{b, sel.holder}
		sel.laidOut.Update(func(n int) int { return n + 1 })
	}
	id, testID := key, "element-"+key
	if sel.owner != "" {
		id = sel.owner
	}
	var selected func() bool
	if t, ok := el.Tap(); ok {
		sel.tapping = &t
		selected = func() bool { return sel.play.leadsTo(t, sel.screen) }
	}
	press := func(ev core.PressEvent) {
		if sel.tapping != nil && !ev.Modifiers.Alt {
			sel.tap(*sel.tapping, sel.screen)
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
	// children draws what the element draws inside it, unless it cannot.
	children := func() ([]core.Node, bool) {
		els, in, under, ok := sel.inside(el, path, sel.parts)
		inner := sel
		inner.scope, inner.holder = in, key
		return elements(cx, els, under, inner), ok
	}
	switch el.Kind {
	case "box":
		kids, _ := children()
		return core.View{Class: class, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide, Selected: selected,
			Children: kids}
	case "slot":
		kids, _ := children()
		if len(kids) == 0 && sel.slots == nil {
			// The component's own file shows where its empty slots are.
			kids = []core.Node{core.Icon{SVG: lucide.SquareDashed(), Class: "size-5 text-muted-foreground"}}
			class = "min-h-10 items-center justify-center rounded-lg border border-dashed border-border " + class
		}
		return core.View{Class: class, Label: el.Text, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide, Selected: selected,
			Children: kids}
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
		kids, ok := children()
		if !ok {
			return placeholder(lucide.Component)
		}
		return core.View{Class: class, Label: el.Text, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide, Selected: selected,
			Children: kids}
	}
	return core.Group{}
}

// componentProblems are what is wrong with the components els use: those
// missing from the folder, those used inside themselves, directly or
// through others, the fills of slots they do not have and what uses
// change in them that is not there. self is the name of the component
// els are, "" for a design, whose slots are checked too.
func componentProblems(els []flowfile.Element, self string, parts components) []string {
	var out []string
	// loop is the chain of uses from name back to a component of using,
	// nil when there is none.
	var loop func(name string, using []string) []string
	loop = func(name string, using []string) []string {
		using = append(slices.Clone(using), name)
		if slices.Index(using, name) < len(using)-1 {
			return using[slices.Index(using, name):]
		}
		for _, n := range flowfile.Uses(parts[name]) {
			if chain := loop(n, using); chain != nil {
				return chain
			}
		}
		return nil
	}
	var using []string
	if self != "" {
		using = []string{self}
		out = append(out, flowfile.SlotProblems(els)...)
	}
	for _, name := range flowfile.Uses(els) {
		if _, ok := parts[name]; !ok {
			out = append(out, fmt.Sprintf("use %q: there is no %s", name, flowfile.ComponentPath(name)))
		} else if chain := loop(name, using); chain != nil {
			out = append(out, fmt.Sprintf("use %q: components use each other in a loop, %s; take a use out of it", name, strings.Join(chain, " uses ")))
		}
	}
	find := func(name string) ([]flowfile.Element, bool) {
		part, ok := parts[name]
		return part, ok
	}
	walk(els, func(el flowfile.Element) {
		if el.Kind == "use" {
			out = append(out, flowfile.FillProblems(el, find)...)
			out = append(out, flowfile.OverrideProblems(el, find)...)
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
