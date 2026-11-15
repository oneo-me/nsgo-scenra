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

// File is a Scenra file's content: a design of a screen, its Device and
// UI, a component designs use, its UI drawn at its Device's width, or a
// flow chart, its Flow. Width and Height, when set, replace the device's.
// Name, when set, is what the editor calls the file instead of its file
// name.
type File struct {
	Kind          Kind
	Name          string
	Device        string
	Width, Height int
	UI            []Element
	Flow          Flow
}

// Size is the size in points a design is drawn at: its device's, or the
// width and height it sets. A component's height is 0 unless set, as it
// is as high as it needs.
func (f File) Size() (w, h int) {
	w, h = DeviceSize(f.Device)
	if f.Kind == Component {
		h = 0
	}
	if f.Width != 0 {
		w = f.Width
	}
	if f.Height != 0 {
		h = f.Height
	}
	return w, h
}

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
	{Kind: "box", Holds: true, Doc: "a rectangle laying out what it holds, a column unless its class has flex-row: with classes, any bar, panel, card, button, field, pill, circle or line"},
	{Kind: "text", Doc: "its text, which wraps; size, weight and color it with classes"},
	{Kind: "icon", Doc: "the Lucide icon its text names, as high as its font size: size it with size-* and color it with text-*"},
	{Kind: "image", Attrs: []string{"src"}, Doc: "a picture its text describes: the image at the URL src, or a placeholder; size it with classes"},
	{Kind: "svg", Doc: "any drawing, such as a chart, a logo or an illustration: its text is an SVG document, quoted with single quotes inside, sized by its class or else its viewBox"},
	{Kind: "use", Doc: "the component its text names, a file of the components folder, drawn as a box with its classes holding the component's elements"},
}

// commonAttrs are the attributes every element takes: the name a use of
// the component it is in changes it by, the classes that shape it, what pressing it does and the tab panel it is, a user's
// comment on it, which asks the assistant for a change until the
// assistant resolves it, and a note, which says what the design cannot
// show and stays.
var commonAttrs = []string{"name", "class", "tap", "panel", "comment", "note"}

// attrOrder is the order Format writes attributes in.
// What a use changes in its component goes before the comment.
var attrOrder = []string{"name", "src", "class", "tap", "panel", "comment", "note"}

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
type uiReader struct {
	out  []Element
	open []Element
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
	el := Element{Kind: spec.Kind}
	for _, t := range toks[1:] {
		key, value, isAttr := t.attr()
		switch {
		case !isAttr && el.Text == "":
			el.Text = t.text
		case !isAttr:
			return fmt.Sprintf("unexpected %q; quote text with spaces, and write attributes as key=value", t.text)
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
		return fmt.Sprintf("name %q is not a name of letters, digits, _ or -, nor an attribute such as class", name)
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
			return fmt.Sprintf("%q names no component; use the path of a file in %s/ without .scenra, such as tab-bar", el.Text, ComponentsDir)
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

// Parse reads a Scenra file, whose first line says whether it holds a
// design or a flow.
func Parse(src string) (File, error) {
	var f File
	var ui uiReader
	lines := 0
	err := eachLine(src, func(toks []token) string {
		lines++
		head := toks[0]
		word := ""
		if !head.quoted {
			word = head.text
		}
		switch {
		case lines == 1:
			return f.header(toks)
		case word == string(Design) || word == string(Component) || word == string(Chart):
			return "a file holds one design, component or flow; put another in a file of its own"
		case f.Kind == Chart:
			return f.Flow.line(toks)
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
	els, msg := ui.done()
	if msg != "" {
		return File{}, &Error{lines, msg}
	}
	f.UI = els
	f.Flow.fillLabels()
	return f, nil
}

const headerHelp = "a file starts with design, component or flow, the one it holds"

// header reads the first line: its name, and design or component with
// its device and size, or flow with its direction.
func (f *File) header(toks []token) string {
	head := toks[0]
	if head.quoted || !slices.Contains([]Kind{Design, Component, Chart}, Kind(head.text)) {
		return headerHelp
	}
	f.Kind = Kind(head.text)
	for _, t := range toks[1:] {
		key, value, isAttr := t.attr()
		switch {
		case !isAttr:
			return fmt.Sprintf("unexpected %q; write attributes as key=value", t.text)
		case key == "name":
			f.Name = strings.TrimSpace(value)
		case f.Kind != Chart && key == "device":
			if !slices.Contains(Devices, value) {
				return fmt.Sprintf("unknown device %q; use one of %s", value, strings.Join(Devices, ", "))
			}
			f.Device = value
		case f.Kind != Chart && (key == "width" || key == "height"):
			n, err := strconv.Atoi(value)
			if err != nil || n < MinSize || n > MaxSize {
				return fmt.Sprintf("%s=%q is not a size; use whole points from %d to %d", key, value, MinSize, MaxSize)
			}
			if key == "width" {
				f.Width = n
			} else {
				f.Height = n
			}
		case f.Kind == Chart && key == "direction":
			switch dir := Direction(value); dir {
			case Down, Right:
				f.Flow.Direction = dir
			default:
				return fmt.Sprintf("unknown direction %q; use down or right", value)
			}
		case f.Kind != Chart:
			return fmt.Sprintf("unknown attribute %q; a %s takes name, device, width and height", key, f.Kind)
		default:
			return fmt.Sprintf("unknown attribute %q; a flow takes name and direction", key)
		}
	}
	return ""
}

// Clone returns a copy of f that shares nothing with it.
func (f File) Clone() File {
	f.UI = cloneElements(f.UI)
	f.Flow = f.Flow.Clone()
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

// Format writes f as a Scenra file.
func (f File) Format() string {
	var b strings.Builder
	if f.Kind == Chart {
		b.WriteString(string(Chart))
		f.formatName(&b)
		if f.Flow.Direction != "" && f.Flow.Direction != Down {
			fmt.Fprintf(&b, " direction=%s", f.Flow.Direction)
		}
		b.WriteByte('\n')
		f.Flow.body(&b)
		return b.String()
	}
	kind := f.Kind
	if kind == "" {
		kind = Design
	}
	b.WriteString(string(kind))
	f.formatName(&b)
	if f.Device != "" && f.Device != "phone" {
		fmt.Fprintf(&b, " device=%s", f.Device)
	}
	if f.Width != 0 {
		fmt.Fprintf(&b, " width=%d", f.Width)
	}
	if f.Height != 0 {
		fmt.Fprintf(&b, " height=%d", f.Height)
	}
	b.WriteByte('\n')
	formatUI(&b, f.UI, "")
	return b.String()
}

func (f File) formatName(b *strings.Builder) {
	if f.Name != "" {
		fmt.Fprintf(b, " name=%s", quote(f.Name))
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
		if spec, _ := Spec(el.Kind); spec.Holds {
			formatUI(b, el.Children, indent+"  ")
			b.WriteString(indent + "end\n")
		}
	}
}
