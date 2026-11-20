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
	// tap plays a tap of the design at index screen, pressed in what the
	// outlet at the key outlet shows, "" for the design's own elements.
	tap func(t flowfile.Tap, screen int, outlet string)
	// report, when set, hears where each element was laid out, and
	// unseen says nothing drawn is seen, so tests find none of it.
	report func(key string, f frame)
	unseen bool
	// bare, when set, says whether a component draws without the dashed
	// line marking its bounds.
	bare func() bool
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
	// parts are the components a use draws, and fills what the outlets
	// show, by their keys, nil where they show their own elements.
	parts components
	fills func(key string) ([]flowfile.Element, bool)
	// hosted are the outlets designs filling them lay out at, by the
	// designs' index.
	hosted map[int]hosting
	// assets are the pictures of the folder an image's src may name.
	assets assets
	scope
	// holder is the key of the element whose view the elements drawn are
	// in, "" at the top of a design.
	holder string
	// tapping is the tap of the element an element is in, which pressing
	// it plays, and tappingIn the outlet showing that element.
	tapping   *flowfile.Tap
	tappingIn string
	// stand is the use the element drawn is drawn in place of, nil for
	// any other.
	stand *standIn
}

// standIn is a use whose component's element at its top draws in its
// place: the element lays out where the use is, recording the use's
// frame, and hides, is selected and is found by tests as the use.
type standIn struct {
	at       func(core.Bounds)
	hide     func() core.Style
	selected func() bool
	testID   string
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
// again, and slots the fills of the use drawing it. Inside a design an
// outlet shows, owner is the outlet's key, and outlet too. fill is what
// the outlets show by their keys.
type scope struct {
	prefix string
	owner  string
	using  []string
	slots  *slots
	outlet string
	fill   func(key string) ([]flowfile.Element, bool)
}

// top is the scope of the elements at the top of the design at index i.
func (sel selection) top(i int) scope { return scope{prefix: screenPrefix(i), fill: sel.fills} }

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
// component, a slot what the use's fill of it holds, or else its own,
// and an outlet the design it shows, or else its own.
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
	case "box":
		if el.Attr("outlet") != "" && sc.fill != nil {
			key := sc.prefix + flowfile.FormatPath(path)
			if els, ok := sc.fill(key); ok {
				owner := sc.owner
				if owner == "" {
					owner = key
				}
				return els, scope{prefix: key + outletSep, owner: owner, outlet: key, fill: sc.fill}, nil, true
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

// hit is an element under the pointer, by its key, and the outlet
// showing it.
type hit struct {
	key, outlet string
	el          flowfile.Element
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
			return append([]hit{{key, sc.outlet, els[i]}}, inner...)
		}
		if p.X >= at.X && p.Y >= at.Y && p.X < at.X+f.Size.Width && p.Y < at.Y+f.Size.Height {
			return []hit{{key, sc.outlet, els[i]}}
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
// element of the design, and the innermost tap among them, if any, with
// the outlet showing it.
func target(hits []hit) (path string, tap *flowfile.Tap, outlet string) {
	for _, h := range hits {
		if !strings.ContainsAny(h.key, ":"+outletSep) {
			path = h.key
		}
		if t, ok := h.el.Tap(); ok {
			tap, outlet = &t, h.outlet
		}
	}
	return path, tap, outlet
}

// groupKey is the key of the element the elements at parent are in, by
// which its panels take turns: the use drawing them at a component's
// top, the design's prefix at its top, and the outlet showing them at
// the top of a design an outlet shows.
func groupKey(prefix string, parent []int) string {
	if len(parent) == 0 {
		return strings.TrimSuffix(strings.TrimSuffix(prefix, ":"), outletSep)
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
	// shown are the designs the outlets show, and designs the folder's,
	// the file shown as the editor has it.
	shown   *core.Signal[fills]
	designs func() designs
}

func newPlayer(cx *core.Scope, file func() string) *player {
	return &player{tabs: core.NewSignalFunc(cx, map[string]string(nil), maps.Equal[map[string]string]), file: file,
		shown: core.NewSignalFunc(cx, fills(nil), sameFills), designs: func() designs { return nil }}
}

// fill is what the outlet at key shows, which a caller reading it
// follows.
func (pl *player) fill(key string) ([]flowfile.Element, bool) {
	return fillOf(pl.shown.Get(), pl.designs(), key)
}

// open shows the design t opens in an outlet of the design at index
// screen, or of a design it shows, when it fills one, which says it did;
// opening the design itself shows its outlets' own elements again.
func (pl *player) open(t flowfile.Tap, screen int) bool {
	ds := pl.designs()
	name, id := t.Opens()
	if name == "" {
		name = pl.file()
	}
	f, ok := ds[name]
	i, found := f.Screen(id)
	if !ok || !found || screen >= len(pl.ids) {
		return false
	}
	here := place{pl.file(), screen}
	if (place{name, i}) == here {
		if len(pl.shownIn(screen)) == 0 {
			return false
		}
		shown := maps.Clone(pl.shown.GetUntracked())
		shown.drop(screenPrefix(screen))
		pl.shown.Set(shown)
		return true
	}
	hops, _ := ds.hosts(place{name, i})
	m := slices.IndexFunc(hops, func(h hop) bool { return h.place == here })
	if m < 0 {
		return false
	}
	pl.shown.Set(pl.shown.GetUntracked().show(route(hops, m, screenPrefix(screen), place{name, i})))
	return true
}

// close shows what the outlet at key showed before, or its own elements,
// and says it did; it does nothing for "".
func (pl *player) close(key string) bool {
	shown := maps.Clone(pl.shown.GetUntracked())
	if key == "" || len(shown[key]) == 0 {
		return false
	}
	shown.drop(key + outletSep)
	if shown[key] = shown[key][:len(shown[key])-1]; len(shown[key]) == 0 {
		delete(shown, key)
	}
	pl.shown.Set(shown)
	return true
}

// closing is the outlet whose design closing t, a close naming a design,
// takes away, pressed in what the outlet at the key outlet shows of the
// design at index screen, or nothing: "" when the design at index screen
// closes, as do the designs it fills, which are not drawn, and false
// when t names none of those the design tapped shows in.
func (pl *player) closing(t flowfile.Tap, screen int, outlet string) (string, bool) {
	if t.Target == flowfile.Window {
		return "", true
	}
	name, id := t.Opens()
	if name == "" {
		name = pl.file()
	}
	ds := pl.designs()
	f, ok := ds[name]
	i, found := f.Screen(id)
	if !ok || !found {
		return "", false
	}
	want := place{name, i}
	// The outlets showing what was tapped, from the innermost out.
	shown := pl.shown.GetUntracked()
	for key := outlet; key != ""; {
		if p, ok := shown.top(key); ok && p == want {
			return key, true
		}
		at := strings.LastIndex(key, outletSep)
		if at < 0 {
			break
		}
		key = key[:at]
	}
	here := place{pl.file(), screen}
	if want == here {
		return "", true
	}
	hops, _ := ds.hosts(here)
	return "", slices.ContainsFunc(hops, func(h hop) bool { return h.place == want })
}

// forget drops what the outlets of the design at index screen show, as
// it closes.
func (pl *player) forget(screen int) {
	shown := maps.Clone(pl.shown.GetUntracked())
	shown.drop(screenPrefix(screen))
	pl.shown.Set(shown)
}

// shownIn are the designs the outlets of the design at index screen
// show.
func (pl *player) shownIn(screen int) []place {
	var out []place
	for key, stack := range pl.shown.Get() {
		if strings.HasPrefix(key, screenPrefix(screen)) && len(stack) > 0 {
			out = append(out, stack[len(stack)-1])
		}
	}
	return out
}

// scan finds the panels of the designs, of the components they use and
// of the designs their outlets show, from top.
func (pl *player) scan(screens []flowfile.Screen, parts components, top func(i int) scope) {
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
		walk(s.UI, top(i), nil)
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
// it opens a design an outlet of it shows, or else that design, or
// shows a tab that shows there.
func (pl *player) leadsTo(t flowfile.Tap, screen int) bool {
	switch t.Action {
	case flowfile.TapOpen:
		file, id := t.Opens()
		if file == "" {
			file = pl.file()
		}
		if shown := pl.shownIn(screen); len(shown) > 0 {
			return slices.ContainsFunc(shown, func(p place) bool {
				f := pl.designs()[p.name]
				return p.name == file && p.index < len(f.Screens) && (id == f.Screens[p.index].ID || id == "" && p.index == 0)
			})
		}
		if file != pl.file() || screen >= len(pl.ids) {
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
// stack in its frame, which clips what overflows. A component is drawn
// bare rather than as a card, on its backdrop when it sets one, a dashed
// line marking its bounds unless sel is bare, and is as large as it needs where it sets no
// size.
func screenView(cx *core.Scope, f flowfile.File, i int, sel selection) core.Node {
	colors := ui.ColorsOf(cx)
	w, h := sizeOf(f, i, sel.hosted)
	sel.screen, sel.scope = i, sel.top(i)
	kids := elements(cx, f.Screens[i].UI, nil, sel)
	if f.Kind == flowfile.Component {
		st := core.Style{FlexShrink: 0, AlignSelf: core.AlignStart, FontSize: 15}
		if w != 0 {
			st.Width = core.Pt(core.Points(w))
		}
		if h != 0 {
			st.Height, st.Overflow = core.Pt(core.Points(h)), core.Hidden
		}
		outline := core.View{TestID: "component-outline", PointerEvents: core.PointerEventsNone,
			Style: core.Style{Position: core.Absolute, Left: core.Pt(-1), Top: core.Pt(-1), Right: core.Pt(-1), Bottom: core.Pt(-1),
				BorderStyle: core.BorderDashed},
			StyleFunc: func() core.Style {
				if sel.bare != nil && sel.bare() {
					return core.Style{}
				}
				return core.Style{BorderWidth: 1, BorderColor: colors.Muted()}
			}}
		// The backdrop's layers draw from the bottom, each over the one
		// holding it.
		layers := f.Screens[i].Layers()
		bottom, layerID, componentID := "", "backdrop-layer", "component-"+strconv.Itoa(i)
		if sel.unseen {
			layerID, componentID, outline.TestID = "", "", ""
		}
		if len(layers) > 0 {
			bottom = layers[0]
			for j := len(layers) - 1; j > 0; j-- {
				kids = []core.Node{core.View{Class: layers[j], Style: core.Style{FlexGrow: 1}, TestID: layerID, Children: kids}}
			}
		}
		return core.View{Class: bottom, Style: st, TestID: componentID,
			StyleFunc: func() core.Style { return core.Style{Color: colors.Text()} },
			Children:  append(kids, outline)}
	}
	var height core.Length
	if h != 0 {
		height = core.Pt(core.Points(h))
	}
	radius := core.Points(44)
	device := f.Screens[i].Device
	if host, ok := sel.hosted[i]; ok {
		device = host.device
	}
	switch device {
	case "tablet":
		radius = 28
	case "desktop":
		radius = 12
	}
	return core.View{Style: core.Style{Width: core.Pt(core.Points(w)), Height: height, CornerRadius: radius, FlexShrink: 0,
		BorderCurve: core.BorderCurveContinuous, Overflow: core.Hidden, BorderWidth: 1, FontSize: 15},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Background(), Color: colors.Text(), BorderColor: colors.Border()}
		},
		Children: kids}
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
// component with what it changes, in the use's place when it has one
// element at its top, and the elements of a component select the use; a slot draws what the use's fill of it holds, which
// selects itself, or else its own, and in the component's own file an
// empty slot shows where it is.
func element(cx *core.Scope, el flowfile.Element, path []int, sel selection) core.Node {
	class := el.Attr("class")
	key := sel.prefix + flowfile.FormatPath(path)
	stand := sel.stand
	sel.stand = nil
	layout := func(b core.Bounds) {
		if stand != nil {
			stand.at(b)
			b.Origin = core.Point{}
		}
		sel.frames[key] = frame{b, sel.holder}
		if sel.report != nil {
			sel.report(key, frame{b, sel.holder})
		}
		sel.laidOut.Update(func(n int) int { return n + 1 })
	}
	id, testID := key, "element-"+key
	if sel.unseen {
		testID = ""
	}
	if sel.owner != "" {
		id = sel.owner
	}
	var selected func() bool
	if t, ok := el.Tap(); ok {
		sel.tapping, sel.tappingIn = &t, sel.outlet
		selected = func() bool { return sel.play.leadsTo(t, sel.screen) }
	}
	if stand != nil {
		testID = stand.testID
		if selected == nil {
			selected = stand.selected
		}
	}
	press := func(ev core.PressEvent) {
		if sel.tapping != nil && !ev.Modifiers.Alt {
			sel.tap(*sel.tapping, sel.screen, sel.tappingIn)
			return
		}
		sel.choose(id)
	}
	var hide func() core.Style
	if stand != nil {
		hide = stand.hide
	}
	if el.Attr("panel") != "" {
		group := groupKey(sel.prefix, path[:len(path)-1])
		outer := hide
		hide = func() core.Style {
			if !sel.play.showing(group, el) {
				return core.Style{Display: core.DisplayNone}
			}
			if outer != nil {
				return outer()
			}
			return core.Style{}
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
		els, in, under, ok := sel.inside(el, path, sel.parts)
		if !ok {
			return placeholder(lucide.Component)
		}
		inner := sel
		inner.scope, inner.holder = in, key
		if !flowfile.Rooted(els) {
			return core.View{Class: flowfile.UseClass(el, sel.parts[el.Text]), Label: el.Text, OnPress: press, OnLayout: layout, TestID: testID, StyleFunc: hide, Selected: selected,
				Children: elements(cx, els, under, inner)}
		}
		// The component's element at its top, which took the use's
		// classes, draws in the use's place.
		inner.stand = &standIn{testID: testID, hide: hide, selected: selected, at: func(b core.Bounds) {
			if stand != nil {
				stand.at(b)
				b.Origin = core.Point{}
			}
			sel.frames[key] = frame{b, sel.holder}
			if sel.report != nil {
				sel.report(key, frame{b, sel.holder})
			}
		}}
		return element(cx, els[0], append(slices.Clone(under), 0), inner)
	}
	return core.Group{}
}

// componentProblems are what is wrong with the components the design s
// uses: those missing from the folder, those used inside themselves,
// directly or through others, the fills of slots they do not have and
// what uses change in them that is not there. self is the name of the
// component s is, "" for a design, whose slots and variants are checked
// too.
func componentProblems(s flowfile.Screen, self string, parts components) []string {
	els := s.UI
	var out []string
	// loop is the chain of uses from name back to a component of using,
	// nil when there is none.
	var loop func(name string, using []string) []string
	loop = func(name string, using []string) []string {
		using = append(slices.Clone(using), name)
		if slices.Index(using, name) < len(using)-1 {
			return using[slices.Index(using, name):]
		}
		for _, n := range flowfile.Uses(parts[name].UI) {
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
			out = append(out, fmt.Sprintf("use %q: there is no %s, nor a design of that name", name, flowfile.ComponentPath(name)))
		} else if chain := loop(name, using); chain != nil {
			out = append(out, fmt.Sprintf("use %q: components use each other in a loop, %s; take a use out of it", name, strings.Join(chain, " uses ")))
		}
	}
	find := parts.find
	if self != "" {
		find = func(name string) (flowfile.Part, bool) {
			if name == self {
				return flowfile.Part{UI: els, Variants: s.Variants}, true
			}
			return parts.find(name)
		}
	}
	walk(els, func(el flowfile.Element) {
		if el.Kind == "use" {
			out = append(out, flowfile.FillProblems(el, find)...)
			out = append(out, flowfile.OverrideProblems(el, find)...)
		}
	})
	if self != "" {
		out = append(out, flowfile.VariantProblems(s, self, find)...)
	}
	return out
}

// unknownClasses are the classes of a design's elements, of a
// component's backdrop and of what its variants add that draw nothing.
func unknownClasses(s flowfile.Screen) []string {
	var out []string
	add := func(class string) {
		for _, c := range core.UnknownClasses(class) {
			if !slices.Contains(out, c) {
				out = append(out, c)
			}
		}
	}
	for _, layer := range s.Layers() {
		add(layer)
	}
	var walk func(els []flowfile.Element)
	walk = func(els []flowfile.Element) {
		for _, el := range els {
			add(el.Attr("class"))
			walk(el.Children)
		}
	}
	walk(s.UI)
	for _, v := range s.Variants {
		for key, value := range v.Attrs {
			if key == "class" || strings.HasSuffix(key, ".class") {
				add(value)
			}
		}
	}
	return out
}
