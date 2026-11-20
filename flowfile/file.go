package flowfile

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Devices a design may be drawn on; an empty device is a phone.
var Devices = []string{"phone", "tablet", "desktop"}

// MinSize and MaxSize bound a design's width and height in points.
const (
	MinSize = 100
	MaxSize = 4000
)

// DeviceSize is the size in points a design of device is drawn at
// unless its file sets width or height.
func DeviceSize(device string) (w, h int) {
	switch device {
	case "tablet":
		return 820, 1080
	case "desktop":
		return 1200, 760
	}
	return 390, 800
}

// Kind is what a file holds.
type Kind string

const (
	Design    Kind = "design"
	Component Kind = "component"
	Chart     Kind = "flow"
)

// DefaultGap, MaxGap bound the space in points the editor leaves between
// a file's designs or flows, which its first line may set with gap=.
const (
	DefaultGap = 80
	MaxGap     = 2000
)

// File is a Scenra file's content: designs of screens, a component
// designs use, or flow charts. A file holds
// one kind, as many designs or flows as it likes but one component, and
// the editor lays them out side by side, Gap apart unless it is 0.
type File struct {
	Kind    Kind
	Gap     int
	Screens []Screen
	Flows   []Flow
}

// Screen is one design of a file, or its component: the ID taps and the
// editor name it by, needed once a file holds several, the name the
// editor shows it by, its Device and UI. Width and Height, when set,
// replace the device's.
//
// A design's In, set by in=, names the outlet of another design it fills,
// whose size it takes unless it sets its own. Its Class, set by class=,
// is the classes of its frame, over the theme's background, outline and
// the device's corners, so a screen of another look needs no box drawing
// a second frame inside it.
//
// A component is as large as its elements unless it sets a size, so a
// button draws as a button; its width and height size only its own
// drawing, not its uses, which their classes size. Fill, set by
// width=fill, says it fills the width where it is used, which its file
// shows at Preview points, or else its device's width. Backdrop is the
// classes of what it is drawn on, such as the sidebar a row sits in, in
// layers from the bottom apart by >, and Variants the states drawn
// beside it.
type Screen struct {
	ID, Name      string
	Device        string
	Width, Height int
	In            string
	Class         string
	Fill          bool
	Preview       int
	Backdrop      string
	UI            []Element
	Variants      []Variant
}

// Layers are the classes of the component's backdrop, from the bottom:
// those apart by >, or else the theme's background under the one given,
// so a translucent one shows as it will in light and dark mode.
func (s Screen) Layers() []string {
	var out []string
	for _, layer := range strings.Split(s.Backdrop, ">") {
		if layer = strings.TrimSpace(layer); layer != "" {
			out = append(out, layer)
		}
	}
	if len(out) == 1 && !strings.Contains(s.Backdrop, ">") {
		out = append([]string{"bg-background"}, out...)
	}
	return out
}

// Variant is a state of a component its file draws beside it, such as a
// switch's on and off or a button's kinds, which uses pick by its ID:
// Attrs are a use's class, what it changes by names, the variants it
// starts from and the states it sets.
type Variant struct {
	ID    string
	Attrs map[string]string
}

// Use is the use of the component name drawing v.
func (v Variant) Use(name string) Element { return Element{Kind: "use", Text: name, Attrs: v.Attrs} }

// Title is the name the first design or flow gives itself, which the
// editor calls the file by instead of its file name, "" when none.
func (f File) Title() string {
	switch {
	case f.Kind == Chart && len(f.Flows) > 0:
		return f.Flows[0].Name
	case f.Kind != Chart && len(f.Screens) > 0:
		return f.Screens[0].Name
	}
	return ""
}

// Spacing is the space in points between the file's designs or flows.
func (f File) Spacing() int {
	if f.Gap == 0 {
		return DefaultGap
	}
	return f.Gap
}

// Size is the size in points the design at i is drawn at: its device's,
// or the width and height it sets. A component's width and height are 0
// unless set, as it is as large as it needs, and its width its device's
// when it fills.
func (f File) Size(i int) (w, h int) {
	s := f.Screens[i]
	w, h = DeviceSize(s.Device)
	if f.Kind == Component {
		h = 0
		switch {
		case !s.Fill:
			w = 0
		case s.Preview != 0:
			w = s.Preview
		}
	}
	if s.Width != 0 {
		w = s.Width
	}
	if s.Height != 0 {
		h = s.Height
	}
	return w, h
}

// Screen returns the index of the design with id, the first for "", and
// false when there is none.
func (f File) Screen(id string) (int, bool) {
	if id == "" {
		return 0, len(f.Screens) > 0
	}
	i := slices.IndexFunc(f.Screens, func(s Screen) bool { return s.ID == id })
	return i, i >= 0
}

// Several says the file holds more than one design or flow.
func (f File) Several() bool { return len(f.Screens) > 1 || len(f.Flows) > 1 }

// Element is a piece of a design: a box holding others, a
// text, an icon, an image or an SVG drawing, its look given by Tailwind
// CSS classes.
type Element struct {
	Kind string
	// Text is a text's content, an icon's name, an image's description
	// or an SVG's document.
	Text     string
	Attrs    map[string]string
	Children []Element
}

// Attr returns the element's attribute key, "" when unset.
func (el Element) Attr(key string) string { return el.Attrs[key] }

// ElementSpec says what an element kind is and takes.
type ElementSpec struct {
	Kind string
	// Holds says the element holds the lines up to its end.
	Holds bool
	// Attrs are the attributes it takes besides class.
	Attrs []string
	// Doc tells a reader, the model included, what the element is.
	Doc string
}

// Elements are the kinds of element an interface is made of: a few
// primitives that classes shape into anything, rather than ready-made
// controls whose look cannot change.
var Elements = []ElementSpec{
	{Kind: "box", Holds: true, Attrs: []string{"outlet"}, Doc: "a rectangle laying out what it holds, a column unless its class has flex-row: with classes, any bar, panel, card, button, field, pill, circle or line; outlet= names it a place designs of other files fill, what it holds showing until one does"},
	{Kind: "text", Doc: "its text, which wraps; size, weight and color it with classes"},
	{Kind: "icon", Doc: "the Lucide icon its text names, as high as its font size: size it with size-* and color it with text-*"},
	{Kind: "image", Attrs: []string{"src"}, Doc: "a picture its text describes: the one src names, a file of the assets folder such as assets/latte.jpg or a URL, or a placeholder; size it with classes"},
	{Kind: "svg", Doc: "any drawing, such as a chart, a logo or an illustration: its text is an SVG document, quoted with single quotes inside, sized by its class or else its viewBox"},
	{Kind: "use", Doc: "the component its text names, a file of the components folder, or else the design, at the top or in the views folder, its classes added to the component's element when it has one at its top, which win over its own, or else drawn as a box with its classes holding them; fill lines right after it, up to its end, fill the component's slots"},
	{Kind: "slot", Holds: true, Doc: "in a component, a box holding what a use's fill of the name its text gives holds, the default slot without one, or else what the slot holds"},
	{Kind: "fill", Holds: true, Doc: "inside a use, what the slot of its component the text names draws, the default slot without one"},
}

// fillAttrs are the only attributes a fill takes: it is drawn as what it
// holds, in its slot.
var fillAttrs = []string{"comment", "note"}

// commonAttrs are the attributes every element takes: the name a use of
// the component it is in changes it by, the classes that shape it, what pressing it does and the tab panel it is, a user's
// comment on it, which asks the assistant for a change until the
// assistant resolves it, and a note, which says what the design cannot
// show and stays.
var commonAttrs = []string{"name", "class", "tap", "panel", "comment", "note"}

// attrOrder is the order Format writes attributes in.
// What a use changes in its component goes before the comment.
var attrOrder = []string{"name", "variant", "state", "src", "class", "tap", "panel", "outlet", "comment", "note"}

// Spec returns the spec of the element kind.
func Spec(kind string) (ElementSpec, bool) {
	i := slices.IndexFunc(Elements, func(s ElementSpec) bool { return s.Kind == kind })
	if i < 0 {
		return ElementSpec{}, false
	}
	return Elements[i], true
}

func kinds() string {
	names := make([]string, len(Elements))
	for i, s := range Elements {
		names[i] = s.Kind
	}
	return strings.Join(names, ", ")
}

// uiReader reads element lines, holding the containers still open.
// design is set for a design's lines, which take no slots.
type uiReader struct {
	out    []Element
	open   []Element
	design bool
}

// level is the elements read so far where the next line goes.
func (r *uiReader) level() *[]Element {
	if n := len(r.open); n > 0 {
		return &r.open[n-1].Children
	}
	return &r.out
}

func (r *uiReader) add(el Element) {
	if n := len(r.open); n > 0 {
		r.open[n-1].Children = append(r.open[n-1].Children, el)
	} else {
		r.out = append(r.out, el)
	}
}

// line reads an element or an end, and returns what is wrong with it.
func (r *uiReader) line(toks []token) string {
	head := toks[0]
	if !head.quoted && head.text == "end" {
		if len(r.open) == 0 {
			return "end closes no box"
		}
		el := r.open[len(r.open)-1]
		r.open = r.open[:len(r.open)-1]
		r.add(el)
		return ""
	}
	spec, ok := Spec(head.text)
	if head.quoted || !ok {
		return fmt.Sprintf("unknown element %q; use one of %s", head.text, kinds())
	}
	inUse := len(r.open) > 0 && r.open[len(r.open)-1].Kind == "use"
	switch {
	case spec.Kind == "slot" && r.design:
		return "a design has no slots; put slots in a component, which uses fill"
	case spec.Kind == "fill" && !inUse:
		// A fill opens the use it follows.
		level := r.level()
		n := len(*level)
		if n == 0 || (*level)[n-1].Kind != "use" || len((*level)[n-1].Children) > 0 {
			return "fill fills a slot of the component a use draws; put it right after the use, which then takes an end"
		}
		use := (*level)[n-1]
		*level = (*level)[:n-1]
		r.open = append(r.open, use)
	case spec.Kind != "fill" && inUse:
		return fmt.Sprintf("a use holds only fill lines; end the use before %s", spec.Kind)
	}
	el := Element{Kind: spec.Kind}
	for _, t := range toks[1:] {
		key, value, isAttr := t.attr()
		switch {
		case !isAttr && el.Text == "":
			el.Text = t.text
		case !isAttr:
			return fmt.Sprintf("unexpected %q; quote text with spaces, and write attributes as key=value", t.text)
		case spec.Kind == "fill" && !slices.Contains(fillAttrs, key):
			return fmt.Sprintf("fill takes no attribute %q; use %s, and shape what it holds", key, strings.Join(fillAttrs, ", "))
		case spec.Kind == "use" && slices.Contains(useAttrs, key):
			if msg := checkUseAttr(key, value); msg != "" {
				return msg
			}
			el.setAttr(key, "", value)
		case spec.Kind == "use" && !slices.Contains(commonAttrs, key):
			if _, msg := parseOverride(key, value); msg != "" {
				return msg
			}
			el.setAttr(key, "", value)
		case !slices.Contains(commonAttrs, key) && !slices.Contains(spec.Attrs, key):
			return fmt.Sprintf("%s takes no attribute %q; use %s", spec.Kind, key, strings.Join(slices.Concat(spec.Attrs, commonAttrs), ", "))
		default:
			if el.Attrs == nil {
				el.Attrs = map[string]string{}
			}
			el.Attrs[key] = value
		}
	}
	if msg := checkElement(el); msg != "" {
		return msg
	}
	if el.Kind == "icon" {
		el.Text, _ = IconName(el.Text)
	}
	if el.Kind == "fill" && slices.ContainsFunc(r.open[len(r.open)-1].Children, func(f Element) bool { return f.Text == el.Text }) {
		return fmt.Sprintf("the use fills %s twice; put what it holds in one fill", slotName(el.Text))
	}
	if spec.Holds {
		r.open = append(r.open, el)
	} else {
		r.add(el)
	}
	return ""
}

// checkElement says what is wrong with an element's text, its name, its
// tap or its panel.
func checkElement(el Element) string {
	if name, ok := el.Attrs["name"]; ok && !ValidName(name) {
		return fmt.Sprintf("name %q is not a name of letters, digits, _ or -, nor one of the words elements take: %s", name, strings.Join(ReservedNames(), ", "))
	}
	if outlet, ok := el.Attrs["outlet"]; ok && !ValidName(outlet) {
		return fmt.Sprintf("outlet %q is not a name of letters, digits, _ or -", outlet)
	}
	if tap, ok := el.Attrs["tap"]; ok {
		if _, err := ParseTap(tap); err != nil {
			return err.Error()
		}
	}
	if panel, ok := el.Attrs["panel"]; ok && !ValidID(panel) {
		return fmt.Sprintf("panel %q is not a name of letters, digits, _, - or .", panel)
	}
	switch el.Kind {
	case "icon":
		if _, ok := IconName(el.Text); !ok {
			msg := fmt.Sprintf("unknown icon %q; use a Lucide icon's name", el.Text)
			if similar := similarIcons(el.Text, 12); len(similar) > 0 {
				msg += " such as " + strings.Join(similar, ", ")
			}
			return msg + ", or draw it with svg"
		}
	case "svg":
		if err := checkSVG(el.Text); err != nil {
			return "svg " + err.Error()
		}
	case "use":
		if !ValidComponentName(el.Text) {
			return fmt.Sprintf("%q names no component; use the name of a file of %s/ without .scenra, such as tab-bar", el.Text, ComponentsDir)
		}
	case "slot", "fill":
		if el.Text != "" && (!ValidID(el.Text) || strings.Contains(el.Text, ".")) {
			return fmt.Sprintf("%s %q is not a name of letters, digits, _ or -", el.Kind, el.Text)
		}
	}
	return ""
}

// checkSVG says why doc is not an SVG document.
func checkSVG(doc string) error {
	dec := xml.NewDecoder(strings.NewReader(doc))
	root := ""
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if start, ok := tok.(xml.StartElement); ok && root == "" {
			root = start.Name.Local
		}
	}
	if root != "svg" {
		return errors.New("needs a document such as <svg viewBox='0 0 24 24'>…</svg>")
	}
	return nil
}

// done returns the elements read, or what is still open.
func (r *uiReader) done() ([]Element, string) {
	if n := len(r.open); n > 0 {
		return nil, fmt.Sprintf("%s has no end", r.open[n-1].Kind)
	}
	return r.out, ""
}

// ParseUI reads the lines of a design: elements, without its first line.
func ParseUI(src string) ([]Element, error) {
	var r uiReader
	if err := eachLine(src, r.line); err != nil {
		return nil, err
	}
	els, msg := r.done()
	if msg != "" {
		return nil, &Error{strings.Count(src, "\n") + 1, msg}
	}
	return els, nil
}

// Parse reads a Scenra file, whose first line says whether it holds
// designs, a component or flows: each design or flow starts with its
// line, as the first does.
func Parse(src string) (File, error) {
	var f File
	var ui uiReader
	lines := 0
	// closeScreen takes the elements of the design read so far.
	closeScreen := func() string {
		if f.Kind == Chart || len(f.Screens) == 0 {
			return ""
		}
		els, msg := ui.done()
		f.Screens[len(f.Screens)-1].UI = els
		ui = uiReader{design: f.Kind == Design}
		return msg
	}
	err := eachLine(src, func(toks []token) string {
		lines++
		head := toks[0]
		word := ""
		if !head.quoted {
			word = head.text
		}
		switch {
		case lines == 1:
			msg := f.header(toks, true)
			ui.design = f.Kind == Design
			return msg
		case word == string(f.Kind) && f.Kind == Component:
			return "a component file holds one component; put another in a file of its own"
		case word == string(f.Kind):
			if msg := closeScreen(); msg != "" {
				return msg
			}
			return f.header(toks, false)
		case word == "variant" && f.Kind == Component:
			if len(ui.open) > 0 {
				return "a variant is a line of the component's file, outside its boxes"
			}
			return f.Screens[0].variant(toks)
		case word == "variant":
			return "only a component has variants; draw a design's states as designs of its file"
		case word == string(Design) || word == string(Component) || word == string(Chart):
			return fmt.Sprintf("a file holds designs, a component or flows, never two kinds; put the %s in a file of its own", word)
		case f.Kind == Chart:
			return f.Flows[len(f.Flows)-1].line(toks)
		case word == "node" || len(toks) > 2 && !toks[1].quoted && (toks[1].text == "->" || toks[1].text == "..>"):
			return fmt.Sprintf("a %s holds no flow; draw a flow in a file of its own", f.Kind)
		default:
			return ui.line(toks)
		}
	})
	if err != nil {
		return File{}, err
	}
	if lines == 0 {
		return File{}, &Error{1, headerHelp}
	}
	if msg := closeScreen(); msg != "" {
		return File{}, &Error{lines, msg}
	}
	for i := range f.Flows {
		f.Flows[i].fillLabels()
	}
	if err := f.checkIDs(); err != nil {
		return File{}, &Error{1, err.Error()}
	}
	return f, nil
}

// variant reads a variant of the component s, and returns what is wrong
// with it.
func (s *Screen) variant(toks []token) string {
	if len(toks) < 2 || toks[1].quoted || toks[1].eq >= 0 || !ValidID(toks[1].text) {
		return "variant needs an ID of letters, digits, _, - or ., such as variant secondary"
	}
	v := Variant{ID: toks[1].text, Attrs: map[string]string{}}
	if slices.ContainsFunc(s.Variants, func(o Variant) bool { return o.ID == v.ID }) {
		return fmt.Sprintf("two variants have the ID %q; give each its own", v.ID)
	}
	for _, t := range toks[2:] {
		key, value, ok := t.attr()
		switch {
		case !ok:
			return fmt.Sprintf("unexpected %q; a variant changes the component as a use does, such as label.class=\"…\", variant= or state=", t.text)
		case key == "class":
		case slices.Contains(useAttrs, key):
			if msg := checkUseAttr(key, value); msg != "" {
				return msg
			}
		case slices.Contains(commonAttrs, key):
			return fmt.Sprintf("a variant takes no %s; give it class, variant=, state= and what a use changes by names", key)
		default:
			if _, msg := parseOverride(key, value); msg != "" {
				return msg
			}
		}
		v.Attrs[key] = value
	}
	s.Variants = append(s.Variants, v)
	return ""
}

// checkIDs says what is wrong with the IDs of the file's designs or
// flows: each needs one once there are several, unique in the file.
func (f File) checkIDs() error {
	var ids []string
	for _, s := range f.Screens {
		ids = append(ids, s.ID)
	}
	for _, d := range f.Flows {
		ids = append(ids, d.ID)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		switch {
		case id == "" && len(ids) > 1:
			return fmt.Errorf("a file of several %ss gives each an ID, such as %s sign-up", f.Kind, f.Kind)
		case id != "" && seen[id]:
			return fmt.Errorf("two %ss have the ID %q; give each its own", f.Kind, id)
		}
		seen[id] = true
	}
	return nil
}

const headerHelp = "a file starts with design, component or flow, the one it holds"

// header reads the line starting a design, a component or a flow: its
// ID, name, and device and size, or direction. The file's first line may
// set the file's gap too.
func (f *File) header(toks []token, first bool) string {
	head := toks[0]
	if head.quoted || !slices.Contains([]Kind{Design, Component, Chart}, Kind(head.text)) {
		return headerHelp
	}
	f.Kind = Kind(head.text)
	var s Screen
	var d Flow
	id, name := &s.ID, &s.Name
	if f.Kind == Chart {
		id, name = &d.ID, &d.Name
	}
	for i, t := range toks[1:] {
		key, value, isAttr := t.attr()
		switch {
		case !isAttr && i == 0 && !t.quoted && f.Kind != Component:
			if !ValidID(t.text) {
				return fmt.Sprintf("%q is not an ID of letters, digits, _, - or .", t.text)
			}
			*id = t.text
		case !isAttr:
			return fmt.Sprintf("unexpected %q; write attributes as key=value", t.text)
		case key == "name":
			*name = strings.TrimSpace(value)
		case key == "gap" && !first:
			return "gap is the file's; set it on the first line"
		case key == "gap":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > MaxGap {
				return fmt.Sprintf("gap=%q is not a gap; use whole points from 1 to %d", value, MaxGap)
			}
			f.Gap = n
		case f.Kind != Chart && key == "device":
			if !slices.Contains(Devices, value) {
				return fmt.Sprintf("unknown device %q; use one of %s", value, strings.Join(Devices, ", "))
			}
			s.Device = value
		case f.Kind == Component && key == "width" && value == "fill":
			s.Fill = true
		case f.Kind == Component && key == "preview":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > MaxSize {
				return fmt.Sprintf("preview=%q is not a width; use whole points from 1 to %d", value, MaxSize)
			}
			s.Preview = n
		case f.Kind == Design && key == "in":
			if _, err := ParseHost(value); err != nil {
				return err.Error()
			}
			s.In = value
		case f.Kind == Design && key == "class":
			s.Class = strings.TrimSpace(value)
		case f.Kind == Component && key == "backdrop":
			s.Backdrop = strings.TrimSpace(value)
		case f.Kind != Chart && (key == "width" || key == "height"):
			n, err := strconv.Atoi(value)
			if err != nil || n < MinSize || n > MaxSize {
				return fmt.Sprintf("%s=%q is not a size; use whole points from %d to %d", key, value, MinSize, MaxSize)
			}
			if key == "width" {
				s.Width = n
			} else {
				s.Height = n
			}
		case f.Kind == Chart && key == "direction":
			switch dir := Direction(value); dir {
			case Down, Right:
				d.Direction = dir
			default:
				return fmt.Sprintf("unknown direction %q; use down or right", value)
			}
		case f.Kind == Component:
			return fmt.Sprintf("unknown attribute %q; a component takes name, device, width, preview, height and backdrop", key)
		case f.Kind != Chart:
			return fmt.Sprintf("unknown attribute %q; a %s takes name, device, width, height, in and class", key, f.Kind)
		default:
			return fmt.Sprintf("unknown attribute %q; a flow takes name and direction", key)
		}
	}
	if s.Preview != 0 && !s.Fill {
		return "preview= sets the width a width=fill component shows at; add width=fill, or set width="
	}
	if f.Kind == Chart {
		f.Flows = append(f.Flows, d)
	} else {
		f.Screens = append(f.Screens, s)
	}
	return ""
}

// Clone returns a copy of f that shares nothing with it.
func (f File) Clone() File {
	f.Screens = slices.Clone(f.Screens)
	for i := range f.Screens {
		f.Screens[i].UI = cloneElements(f.Screens[i].UI)
		f.Screens[i].Variants = slices.Clone(f.Screens[i].Variants)
		for j, v := range f.Screens[i].Variants {
			f.Screens[i].Variants[j].Attrs = maps.Clone(v.Attrs)
		}
	}
	f.Flows = slices.Clone(f.Flows)
	for i := range f.Flows {
		f.Flows[i] = f.Flows[i].Clone()
	}
	return f
}

func cloneElements(els []Element) []Element {
	els = slices.Clone(els)
	for i := range els {
		els[i].Attrs = maps.Clone(els[i].Attrs)
		els[i].Children = cloneElements(els[i].Children)
	}
	return els
}

// Format writes f as a Scenra file, its designs or flows a blank line
// apart. A file without any writes the line of an empty one.
func (f File) Format() string {
	var b strings.Builder
	kind := f.Kind
	if kind == "" {
		kind = Design
	}
	gap := func(i int) {
		if i == 0 && f.Gap > 0 {
			fmt.Fprintf(&b, " gap=%d", f.Gap)
		}
	}
	if kind == Chart {
		flows := f.Flows
		if len(flows) == 0 {
			flows = []Flow{{}}
		}
		for i, d := range flows {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(string(Chart))
			writeID(&b, d.ID, d.Name)
			gap(i)
			if d.Direction != "" && d.Direction != Down {
				fmt.Fprintf(&b, " direction=%s", d.Direction)
			}
			b.WriteByte('\n')
			d.body(&b)
		}
		return b.String()
	}
	screens := f.Screens
	if len(screens) == 0 {
		screens = []Screen{{}}
	}
	for i, s := range screens {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(string(kind))
		writeID(&b, s.ID, s.Name)
		gap(i)
		if s.Device != "" && s.Device != "phone" {
			fmt.Fprintf(&b, " device=%s", s.Device)
		}
		if s.Fill {
			b.WriteString(" width=fill")
			if s.Preview != 0 {
				fmt.Fprintf(&b, " preview=%d", s.Preview)
			}
		} else if s.Width != 0 {
			fmt.Fprintf(&b, " width=%d", s.Width)
		}
		if s.Height != 0 {
			fmt.Fprintf(&b, " height=%d", s.Height)
		}
		if s.In != "" {
			fmt.Fprintf(&b, " in=%s", quote(s.In))
		}
		if s.Class != "" {
			fmt.Fprintf(&b, " class=%s", quote(s.Class))
		}
		if s.Backdrop != "" {
			fmt.Fprintf(&b, " backdrop=%s", quote(s.Backdrop))
		}
		b.WriteByte('\n')
		formatUI(&b, s.UI, "")
		for _, v := range s.Variants {
			b.WriteString("variant " + v.ID)
			for _, key := range []string{"variant", "state", "class"} {
				if value, ok := v.Attrs[key]; ok {
					fmt.Fprintf(&b, " %s=%s", key, quote(value))
				}
			}
			for _, key := range overrideKeys(v.Attrs) {
				fmt.Fprintf(&b, " %s=%s", key, quote(v.Attrs[key]))
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// writeID writes the ID and name of a design's or flow's line.
func writeID(b *strings.Builder, id, name string) {
	if id != "" {
		b.WriteString(" " + id)
	}
	if name != "" {
		fmt.Fprintf(b, " name=%s", quote(name))
	}
}

// FormatUI writes elements as the lines of a design.
func FormatUI(els []Element) string {
	var b strings.Builder
	formatUI(&b, els, "")
	return b.String()
}

func formatUI(b *strings.Builder, els []Element, indent string) {
	for _, el := range els {
		b.WriteString(indent + el.Kind)
		if el.Text != "" {
			b.WriteString(" " + quote(el.Text))
		}
		for _, key := range attrOrder {
			if key == "comment" {
				for _, o := range overrideKeys(el.Attrs) {
					fmt.Fprintf(b, " %s=%s", o, quote(el.Attrs[o]))
				}
			}
			if value, ok := el.Attrs[key]; ok {
				fmt.Fprintf(b, " %s=%s", key, quote(value))
			}
		}
		b.WriteByte('\n')
		if spec, _ := Spec(el.Kind); spec.Holds || len(el.Children) > 0 {
			formatUI(b, el.Children, indent+"  ")
			b.WriteString(indent + "end\n")
		}
	}
}
