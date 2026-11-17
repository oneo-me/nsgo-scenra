package scenraview

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/nsgo-dev/nsgo/core"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// Agents cannot see the canvas, so the editor lays the folder's designs
// and components out unseen, as the canvas would, and tells them where
// each element went and what that layout gets wrong: text cut off,
// elements outside their design or of no size, w-full that fills
// nothing, and uses far narrower than their components show.

// layoutTarget is a design or component to lay out: file draws it as the
// design at index 0, its key and name naming it, under the prefix of the
// keys of its elements, those of the outlet showing it and a bar for a
// design filling one.
type layoutTarget struct {
	path, name string
	file       flowfile.File
	fills      fills
	under      string
	// component is the component drawn, "" for a design, and variant the
	// variant it draws as.
	component, variant string
}

// placed is an element as it laid out: its key, its path under the
// target's prefix, what it is, where it went in the target's
// coordinates, the element holding it, -1 for none, the component it
// draws for a use and how deep it is.
type placed struct {
	key, path string
	el        flowfile.Element
	bounds    core.Bounds
	parent    int
	depth     int
}

// laidOut is a target laid out: its elements in the order they draw and
// its size.
type laidOut struct {
	layoutTarget
	els  []placed
	size core.Size
}

// probe asks the editor to lay targets out, and done hears how they did,
// once, as nil when nothing could lay them out.
type probe struct {
	targets []layoutTarget
	done    func([]laidOut)
}

// layOut lays targets out unseen, and has done hear how they did, nil
// when no Workspace shows the editor to lay them out.
func (e *Editor) layOut(targets []layoutTarget, done func([]laidOut)) {
	if e.probing == 0 || len(targets) == 0 {
		done(nil)
		return
	}
	p := &probe{targets, done}
	if e.probe.GetUntracked() != nil {
		e.probes = append(e.probes, p)
		return
	}
	e.probe.Set(p)
}

// finish ends the probe laying out, with what it found, and starts the
// next.
func (e *Editor) finish(p *probe, out []laidOut) {
	var next *probe
	if len(e.probes) > 0 {
		next, e.probes = e.probes[0], e.probes[1:]
	}
	e.probe.Set(next)
	p.done(out)
}

// probeTries is how many turns of the UI's loop a probe waits for its
// targets to lay out.
const probeTries = 200

// probeLayer lays out unseen what the editor's probe asks for.
func probeLayer(cx *core.Scope, e *Editor) core.Node {
	e.probing++
	cx.Cleanup(func() {
		e.probing--
		if p := e.probe.GetUntracked(); p != nil && e.probing == 0 {
			e.finish(p, nil)
		}
	})
	return core.Dynamic(func(cx *core.Scope) core.Node {
		p := e.probe.Get()
		if p == nil {
			return core.Group{}
		}
		parts, pics := e.parts.GetUntracked(), e.assets.GetUntracked()
		ds := e.allDesigns()
		sels := make([]selection, len(p.targets))
		var nodes []core.Node
		for i, t := range p.targets {
			still := stillSelection(cx, t.file.Screens, parts, pics)
			still.origins, still.unseen = []core.Point{{}}, true
			shown := t.fills
			still.fills = func(k string) ([]flowfile.Element, bool) { return fillOf(shown, ds, k) }
			if t.component != "" {
				still.using = []string{t.component}
			}
			sels[i] = still
			nodes = append(nodes, core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), AlignItems: core.AlignStart},
				Children: []core.Node{screenView(cx, t.file, 0, still)}})
		}
		tries := 0
		var check func()
		check = func() {
			if e.probe.GetUntracked() != p {
				return
			}
			ready := true
			for i, t := range p.targets {
				if len(t.file.Screens[0].UI) > 0 {
					_, ok := sels[i].frames[t.under+"0"]
					ready = ready && ok
				}
			}
			if !ready && tries < probeTries {
				tries++
				cx.Post(check)
				return
			}
			out := make([]laidOut, len(p.targets))
			for i, t := range p.targets {
				out[i] = laidOutOf(t, sels[i])
			}
			e.finish(p, out)
		}
		cx.Post(check)
		return unseen("layout-probe", nodes)
	})
}

// allDesigns are the folder's designs, the one shown as the editor has
// it.
func (e *Editor) allDesigns() designs {
	d := e.shown()
	return e.designs.GetUntracked().with(e.designName(d.path), d.doc.GetUntracked())
}

// laidOutOf is where t's elements went, as sel recorded them.
func laidOutOf(t layoutTarget, sel selection) laidOut {
	out := laidOut{layoutTarget: t}
	var origin core.Point
	if t.under != screenPrefix(0) {
		b, _ := sel.bounds(strings.TrimSuffix(t.under, outletSep))
		origin, out.size = b.Origin, b.Size
	}
	var walk func(els []flowfile.Element, sc scope, parent []int, holder, depth int)
	walk = func(els []flowfile.Element, sc scope, parent []int, holder, depth int) {
		for i, el := range els {
			path := append(slices.Clone(parent), i)
			key := sc.prefix + flowfile.FormatPath(path)
			if !sel.play.showing(groupKey(sc.prefix, parent), el) {
				continue
			}
			at := holder
			if strings.HasPrefix(key, t.under) {
				if b, ok := sel.bounds(key); ok {
					b.Origin.X -= origin.X
					b.Origin.Y -= origin.Y
					out.els = append(out.els, placed{key, strings.TrimPrefix(key, t.under), el, b, holder, depth})
					at = len(out.els) - 1
				}
			}
			if kids, in, under, ok := sc.inside(el, path, sel.parts); ok {
				walk(kids, in, under, at, depth+1)
			}
		}
	}
	walk(t.file.Screens[0].UI, sel.top(0), nil, -1, 0)
	// The elements of a design an outlet shows start at its top.
	if len(out.els) > 0 {
		base := out.els[0].depth
		for i := range out.els {
			out.els[i].depth -= base
		}
	}
	if t.under == screenPrefix(0) {
		for _, p := range out.els {
			if p.parent < 0 {
				out.size.Width = max(out.size.Width, p.bounds.Origin.X+p.bounds.Size.Width)
				out.size.Height = max(out.size.Height, p.bounds.Origin.Y+p.bounds.Size.Height)
			}
		}
		if w, h := t.file.Size(0); w != 0 && h != 0 {
			out.size = core.Size{Width: core.Points(w), Height: core.Points(h)}
		}
	}
	return out
}

// targetsOf are the designs of f, the file at path, named name, or its
// component, and the variants it draws as, to lay out; a design filling
// an outlet lays out in its host, as it plays.
func targetsOf(path string, f flowfile.File, ds designs) []layoutTarget {
	var out []layoutTarget
	name := nameOf(path)
	switch f.Kind {
	case flowfile.Component:
		self, ok := flowfile.ComponentName(path)
		if !ok || len(f.Screens) == 0 {
			return nil
		}
		s := f.Screens[0]
		one := flowfile.File{Kind: flowfile.Component, Screens: []flowfile.Screen{s}}
		out = append(out, layoutTarget{path: path, name: self, file: one, under: screenPrefix(0), component: self})
		for _, v := range s.Variants {
			vf := one.Clone()
			vf.Screens[0].UI = flowfile.Instance(v.Use(self), flowfile.Part{UI: s.UI, Variants: s.Variants})
			out = append(out, layoutTarget{path: path, name: self + " variant " + v.ID, file: vf, under: screenPrefix(0), component: self, variant: v.ID})
		}
	case flowfile.Design:
		for i, s := range f.Screens {
			title := path
			if f.Several() {
				title += "#" + s.ID
			}
			if _, in := f.HostOf(i); in {
				hops, msg := ds.hosts(place{name, i})
				if msg == "" && len(hops) > 0 {
					one, shown, key := composed(ds, hops, place{name, i})
					out = append(out, layoutTarget{path: path, name: title, file: one, fills: shown, under: key + outletSep})
					continue
				}
			}
			one := flowfile.File{Kind: flowfile.Design, Screens: []flowfile.Screen{s}}
			out = append(out, layoutTarget{path: path, name: title, file: one, under: screenPrefix(0)})
		}
	}
	return out
}

// describe names p in a message: its path, its kind and the start of its
// text, or the component a use draws.
func describe(p placed) string {
	s := p.path + " " + p.el.Kind
	switch p.el.Kind {
	case "use":
		s += " " + p.el.Text
		if v := p.el.Attr("variant"); v != "" {
			s += " variant=" + quote(v)
		}
	case "text":
		s += " " + quote(clip(p.el.Text, 40))
	case "icon", "slot", "fill":
		if p.el.Text != "" {
			s += " " + p.el.Text
		}
	}
	return s
}

func quote(s string) string { return fmt.Sprintf("%q", s) }

// clip shortens s to n runes.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// pt writes a length in points, to a tenth.
func pt(v core.Points) string {
	return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", float64(v)), "0"), ".")
}

// layoutText is l, an element a line, indented by depth: its path, kind,
// x, y, width and height. When element is set only the element at that
// path and what it holds tell, false when l has none, and when depth is
// not negative only those up to depth levels under it, each holding
// more saying how many.
func layoutText(l laidOut, element string, depth int) (string, bool) {
	root := -1
	if element != "" {
		root = slices.IndexFunc(l.els, func(p placed) bool { return p.path == element })
		if root < 0 {
			return "", false
		}
	}
	// level is how far under the root the element at i is, -1 outside it.
	level := func(i int) int {
		if root < 0 {
			return l.els[i].depth
		}
		n := 0
		for ; i != root; i = l.els[i].parent {
			if i < 0 {
				return -1
			}
			n++
		}
		return n
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s laid out %s×%s points; each element's x and y are from its top left corner.\n", l.name, pt(l.size.Width), pt(l.size.Height))
	if root >= 0 {
		fmt.Fprintf(&b, "Only %s and what it holds:\n", element)
	}
	// more counts the elements left out under each listed at the depth.
	more := map[int]int{}
	for i := range l.els {
		n := level(i)
		if n < 0 || depth < 0 || n <= depth {
			continue
		}
		at := i
		for level(at) > depth {
			at = l.els[at].parent
		}
		more[at]++
	}
	base := 0
	if root >= 0 {
		base = l.els[root].depth
	}
	for i, p := range l.els {
		n := level(i)
		if n < 0 || depth >= 0 && n > depth {
			continue
		}
		indent := strings.Repeat("  ", p.depth-base)
		fmt.Fprintf(&b, "%s%s at %s,%s %s×%s\n", indent, describe(p),
			pt(p.bounds.Origin.X), pt(p.bounds.Origin.Y), pt(p.bounds.Size.Width), pt(p.bounds.Size.Height))
		if m := more[i]; m > 0 {
			fmt.Fprintf(&b, "%s  … %d more elements inside; ask for %s with a greater depth to see them\n", indent, m, p.path)
		}
	}
	return b.String(), true
}

// slack is how far in points an element may reach past what holds it
// before it counts as cut off, as text rounds.
const slack = 0.5

// hasClass says class holds one of the words, or one starting with a
// prefix ending in -.
func hasClass(class string, words ...string) bool {
	for _, c := range strings.Fields(class) {
		for _, w := range words {
			if c == w || strings.HasSuffix(w, "-") && strings.HasPrefix(c, w) {
				return true
			}
		}
	}
	return false
}

// layoutProblems are what is wrong with how l laid out, and widths of
// components, by name, the uses of which are far narrower.
func layoutProblems(l laidOut, widths map[string]core.Points) []string {
	var out []string
	inside := func(b, c core.Bounds) bool {
		return b.Origin.X >= c.Origin.X-slack && b.Origin.Y >= c.Origin.Y-slack &&
			b.Origin.X+b.Size.Width <= c.Origin.X+c.Size.Width+slack && b.Origin.Y+b.Size.Height <= c.Origin.Y+c.Size.Height+slack
	}
	whole := core.Bounds{Size: l.size}
	clipsDesign := l.component == "" || l.file.Screens[0].Height != 0
	// cut are the elements cut off, whose own are not told again.
	cut := map[int]bool{}
	for i, p := range l.els {
		class := p.el.Attr("class")
		if p.parent >= 0 && cut[p.parent] || hasClass(class, "hidden") {
			cut[i] = true
			continue
		}
		// A slot is where uses put what they hold, empty where none does.
		if p.el.Kind == "slot" || p.el.Kind == "fill" {
			continue
		}
		// What holds it clips it where it hides its overflow, as the
		// design does; scrolling hides it on purpose.
		var clipper string
		var clipBounds core.Bounds
		scrolls := false
		for a := p.parent; a >= 0; a = l.els[a].parent {
			c := l.els[a].el.Attr("class")
			if hasClass(c, "overflow-auto", "overflow-scroll", "overflow-x-auto", "overflow-y-auto", "overflow-x-scroll", "overflow-y-scroll") {
				scrolls = true
				break
			}
			if hasClass(c, "overflow-hidden", "overflow-clip", "overflow-x-hidden", "overflow-y-hidden") {
				clipper, clipBounds = describe(l.els[a]), l.els[a].bounds
				break
			}
		}
		if scrolls {
			continue
		}
		switch {
		case clipper != "" && !inside(p.bounds, clipBounds):
			cut[i] = true
			if p.el.Kind == "text" && !hasClass(class, "truncate", "line-clamp-", "text-ellipsis") {
				out = append(out, fmt.Sprintf("%s: text %s is cut off by %s, which hides what overflows it", l.name, describe(p), clipper))
			} else if p.el.Kind != "text" {
				out = append(out, fmt.Sprintf("%s: %s reaches outside %s, which hides what overflows it", l.name, describe(p), clipper))
			}
		case clipsDesign && !inside(p.bounds, whole) && !hasClass(class, "absolute"):
			cut[i] = true
			what := "the design"
			if l.under != screenPrefix(0) {
				what = "the outlet it fills"
			}
			out = append(out, fmt.Sprintf("%s: %s, at %s,%s %s×%s, reaches outside %s, which cuts it off", l.name, describe(p),
				pt(p.bounds.Origin.X), pt(p.bounds.Origin.Y), pt(p.bounds.Size.Width), pt(p.bounds.Size.Height), what))
		case p.bounds.Size.Width < slack || p.bounds.Size.Height < slack:
			if p.el.Kind == "box" && len(p.el.Children) == 0 && !hasClass(class, "bg-", "border", "shadow") {
				continue
			}
			cut[i] = true
			out = append(out, fmt.Sprintf("%s: %s laid out %s×%s, so it does not show", l.name, describe(p), pt(p.bounds.Size.Width), pt(p.bounds.Size.Height)))
		}
		if msg := unfilled(l, i); msg != "" {
			out = append(out, l.name+": "+msg)
		}
		if p.el.Kind == "use" {
			// Only a component of a width set shows how wide its uses are.
			if w, ok := widths[p.el.Text]; ok && w > 0 && p.bounds.Size.Width < w/2 {
				out = append(out, fmt.Sprintf("%s: %s is %s wide, under half the %s %s shows at; check it fills what it should, or give the component preview= its width there (a hint)",
					l.name, describe(p), pt(p.bounds.Size.Width), pt(w), flowfile.ComponentPath(p.el.Text)))
			}
		}
	}
	return out
}

// sizing are the classes that give an element its own width.
var sizing = []string{"w-", "size-", "flex-1", "flex-auto", "grow", "basis-", "min-w-", "self-stretch", "absolute", "fixed"}

// lastClass is the last class of class, outside variants, that is one of
// the words or starts with a prefix ending in -, as the later of
// conflicting classes wins; "" when none is.
func lastClass(class string, words ...string) string {
	fields := strings.Fields(class)
	for i := len(fields) - 1; i >= 0; i-- {
		if c := fields[i]; !strings.Contains(c, ":") && hasClass(c, words...) {
			return c
		}
	}
	return ""
}

// inRow says class lays out what its box holds in a row, as boxes are
// columns unless they say otherwise.
func inRow(class string) bool {
	return strings.HasPrefix(lastClass(class, "flex-row", "flex-row-reverse", "flex-col", "flex-col-reverse"), "flex-row")
}

// stretched says a column of class stretches the element of own across
// it, as it does unless either aligns elsewhere.
func stretched(own, class string) bool {
	if self := lastClass(own, "self-"); self != "" && self != "self-auto" {
		return self == "self-stretch"
	}
	items := lastClass(class, "items-")
	return items == "" || items == "items-stretch"
}

// sideInsets are the padding and border widths class gives the left and
// right of its box, the later classes winning per side.
func sideInsets(class string) core.Points {
	var pad, border [2]core.Points
	set := func(to *[2]core.Points, side string, v core.Points) {
		switch side {
		case "", "x":
			to[0], to[1] = v, v
		case "l", "s":
			to[0] = v
		case "r", "e":
			to[1] = v
		}
	}
	for _, c := range strings.Fields(class) {
		if strings.Contains(c, ":") {
			continue
		}
		name, value, _ := strings.Cut(c, "-")
		switch {
		case name == "p" || len(name) == 2 && name[0] == 'p':
			if v, ok := spacing(value); ok {
				set(&pad, name[1:], v)
			}
		case name == "border":
			side, width, _ := strings.Cut(value, "-")
			if !slices.Contains([]string{"", "x", "y", "t", "b", "l", "r", "s", "e"}, side) {
				side, width = "", value
			}
			v, ok := core.Points(1), true
			if width != "" {
				v, ok = length(width)
			}
			if ok {
				set(&border, side, v)
			}
		}
	}
	return pad[0] + pad[1] + border[0] + border[1]
}

// spacing is a value of Tailwind's spacing scale, a quarter rem a step,
// px or a length in brackets.
func spacing(v string) (core.Points, bool) {
	if v == "px" {
		return 1, true
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil && n >= 0 {
		return core.Points(n * 4), true
	}
	return length(v)
}

// length is a whole number of pixels, or a length in brackets, such as
// [13px].
func length(v string) (core.Points, bool) {
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return core.Points(n), true
	}
	inner, ok := strings.CutPrefix(v, "[")
	if inner, ok = strings.CutSuffix(inner, "px]"); ok {
		if n, err := strconv.ParseFloat(inner, 64); err == nil && n >= 0 {
			return core.Points(n), true
		}
	}
	return 0, false
}

// unfilled says how the w-full or row flex-1 of the element at i fills
// nothing: what holds it is only as wide as what it holds, as it sets no
// width of its own and the column it is in does not stretch it, and is
// narrower than the content of what holds that, inside its padding and
// borders, or than what a row it is in leaves its siblings and gaps.
func unfilled(l laidOut, i int) string {
	p := l.els[i]
	class := p.el.Attr("class")
	if p.parent < 0 || l.els[p.parent].parent < 0 {
		return ""
	}
	parent, outer := l.els[p.parent], l.els[l.els[p.parent].parent]
	pc, oc := parent.el.Attr("class"), outer.el.Attr("class")
	if !hasClass(class, "w-full") && !(inRow(pc) && hasClass(class, "flex-1")) || hasClass(pc, sizing...) || outer.el.Kind == "use" {
		return ""
	}
	row := inRow(oc)
	if !row && stretched(pc, oc) {
		return ""
	}
	content := outer.bounds.Size.Width - sideInsets(oc)
	used := parent.bounds.Size.Width
	if row {
		// The row's other elements and its gaps take their share.
		left, right := parent.bounds.Origin.X, parent.bounds.Origin.X+parent.bounds.Size.Width
		for _, q := range l.els {
			if q.parent == parent.parent && !hasClass(q.el.Attr("class"), "absolute", "fixed", "hidden") {
				left, right = min(left, q.bounds.Origin.X), max(right, q.bounds.Origin.X+q.bounds.Size.Width)
			}
		}
		used = right - left
	}
	if used >= content-slack {
		return ""
	}
	what := "w-full"
	if !hasClass(class, "w-full") {
		what = "flex-1"
	}
	return fmt.Sprintf("%s of %s fills only %s, which is as wide as what it holds, %s of the %s content width of %s; give %s w-full, flex-1 or a width",
		what, describe(p), describe(parent), pt(parent.bounds.Size.Width), pt(content), outer.path, parent.path)
}

// layoutSection tells what is wrong with how out laid out, a line each,
// or nothing.
func layoutSection(out []laidOut) string {
	// The width a component shows at, when it sets one or fills.
	widths := map[string]core.Points{}
	for _, l := range out {
		if w, _ := l.file.Size(0); l.component != "" && l.variant == "" && w != 0 {
			widths[l.component] = core.Points(w)
		}
	}
	var lines []string
	for _, l := range out {
		for _, p := range layoutProblems(l, widths) {
			if !slices.Contains(lines, p) {
				lines = append(lines, p)
			}
		}
	}
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("How the designs and components lay out, as get_layout tells element by element, where it goes wrong:\n")
	for _, l := range lines {
		b.WriteString("- " + l + "\n")
	}
	return b.String()
}
