package flowfile

import (
	"fmt"
	"slices"
	"strings"
)

// A component's elements may take a name, by which a use of it changes
// them where it is, as Figma's instances and Sketch's symbols override
// their layers: use card title="Latte" draws the card with the text of
// its elements named title replaced, title.class="text-rose-600" adds
// to their classes, and title.src= and title.tap= replace their image
// and tap. The text of an icon is its name, and that of a use the
// component it draws, which a use swaps. row/title reaches the
// elements named title of the component the use named row draws, and
// so on down. Every element of a name changes, so one name may stand
// for several elements that show one value.

// OverrideAttrs are the attributes of a named element a use may change
// besides its text: classes, which add to its own, and those replacing
// its own.
var OverrideAttrs = []string{"class", "src", "tap"}

// ValidName reports whether name can name an element of a component:
// letters, digits, _ and -, and no attribute an element takes.
func ValidName(name string) bool {
	if !ValidID(name) || strings.Contains(name, ".") || slices.Contains(commonAttrs, name) {
		return false
	}
	return !slices.ContainsFunc(Elements, func(s ElementSpec) bool { return slices.Contains(s.Attrs, name) })
}

// Override is what a use changes in its component: the elements a path
// of names leads to, each the name of an element inside the component
// the one before it draws, and their attribute Attr, or their text when
// Attr is "".
type Override struct {
	Path        []string
	Attr, Value string
}

// Key is the attribute of the use setting o.
func (o Override) Key() string {
	key := strings.Join(o.Path, "/")
	if o.Attr != "" {
		key += "." + o.Attr
	}
	return key
}

// parseOverride reads the key of an attribute of a use that changes its
// component, and says what is wrong with it.
func parseOverride(key, value string) (Override, string) {
	names, attr, _ := strings.Cut(key, ".")
	o := Override{Path: strings.Split(names, "/"), Attr: attr, Value: value}
	for _, name := range o.Path {
		if !ValidName(name) {
			return Override{}, fmt.Sprintf("use takes no attribute %q; use %s, or name= of an element of the component, with .%s to change those",
				key, strings.Join(commonAttrs, ", "), strings.Join(OverrideAttrs, ", ."))
		}
	}
	if attr != "" && !slices.Contains(OverrideAttrs, attr) {
		return Override{}, fmt.Sprintf("use cannot change %q of %s; change its text with %s=, or its %s", attr, names, names, strings.Join(OverrideAttrs, ", "))
	}
	if attr == "tap" {
		if _, err := ParseTap(value); err != nil {
			return Override{}, err.Error()
		}
	}
	return o, ""
}

// Overrides are what a use changes in its component, by their keys.
func (el Element) Overrides() []Override {
	if el.Kind != "use" {
		return nil
	}
	var out []Override
	for _, key := range overrideKeys(el.Attrs) {
		o, _ := parseOverride(key, el.Attrs[key])
		out = append(out, o)
	}
	return out
}

// overrideKeys are the keys of attrs no element takes, sorted.
func overrideKeys(attrs map[string]string) []string {
	var out []string
	for key := range attrs {
		if !slices.Contains(attrOrder, key) {
			out = append(out, key)
		}
	}
	slices.Sort(out)
	return out
}

// Instance returns the elements of part, the component use draws, as it
// draws them: with what it changes changed, and what it changes inside
// the components those elements use given to them, overriding what they
// change themselves. The use's classes go to the element at the top of
// a component of one, after its own so they win, as React's className
// goes to a component's root: the use draws as it rather than as a box
// holding it, so w-full and flex-1 work where the use is. part is
// unchanged.
func Instance(use Element, part []Element) []Element {
	overrides := use.Overrides()
	class := use.Attr("class")
	if len(overrides) == 0 && (class == "" || !Rooted(part)) {
		return part
	}
	out := cloneElements(part)
	if class != "" && Rooted(out) {
		out[0].setAttr("class", "class", class)
	}
	var walk func(els []Element)
	walk = func(els []Element) {
		for i := range els {
			el := &els[i]
			if name := el.Attr("name"); name != "" {
				for _, o := range overrides {
					if o.Path[0] == name {
						el.override(o)
					}
				}
			}
			walk(el.Children)
		}
	}
	walk(out)
	return out
}

// Rooted says a component of elements part has one at its top, which a
// use draws in its place.
func Rooted(part []Element) bool { return len(part) == 1 }

// override changes el as o, which names el first.
func (el *Element) override(o Override) {
	if len(o.Path) > 1 {
		if el.Kind == "use" {
			inner := o
			inner.Path = o.Path[1:]
			el.setAttr(inner.Key(), inner.Attr, o.Value)
		}
		return
	}
	switch {
	case o.Attr != "":
		el.setAttr(o.Attr, o.Attr, o.Value)
	case el.Kind == "icon":
		el.Text, _ = IconName(o.Value)
	case el.Kind != "box":
		el.Text = o.Value
	}
}

// setAttr sets the attribute key to value, or adds value to the classes
// there when attr is class.
func (el *Element) setAttr(key, attr, value string) {
	if el.Attrs == nil {
		el.Attrs = map[string]string{}
	}
	if old := el.Attrs[key]; attr == "class" && old != "" {
		value = old + " " + value
	}
	el.Attrs[key] = value
}

// named are the elements of els, and of the boxes they are in, with the
// name, outside the components they use.
func named(els []Element, name string) []Element {
	var out []Element
	for _, el := range els {
		if el.Attr("name") == name {
			out = append(out, el)
		}
		out = append(out, named(el.Children, name)...)
	}
	return out
}

// OverrideProblems are what is wrong with what use changes in the
// component it draws, which parts finds by name: names no element of it
// has, text of boxes, src of what is no image, icons, SVG and
// components that do not exist, and names inside elements that are no
// uses. A use of a missing component has none.
func OverrideProblems(use Element, parts func(name string) ([]Element, bool)) []string {
	part, ok := parts(use.Text)
	if !ok {
		return nil
	}
	var out []string
	for _, o := range use.Overrides() {
		if msg := overrideProblem(o, o.Path, use.Text, Instance(use, part), parts); msg != "" {
			out = append(out, fmt.Sprintf("use %q: %s=%s %s", use.Text, o.Key(), quote(o.Value), msg))
		}
	}
	return out
}

// overrideProblem says what is wrong with o where path is left of it in
// els, the elements of the component name as drawn.
func overrideProblem(o Override, path []string, name string, els []Element, parts func(string) ([]Element, bool)) string {
	found := named(els, path[0])
	if len(found) == 0 {
		return fmt.Sprintf("names nothing: no element of %s has name=%s", ComponentPath(name), path[0])
	}
	for _, el := range found {
		if len(path) > 1 {
			if el.Kind != "use" {
				return fmt.Sprintf("goes inside %s, a %s of %s, not a use", path[0], el.Kind, ComponentPath(name))
			}
			if part, ok := parts(el.Text); ok {
				if msg := overrideProblem(o, path[1:], el.Text, Instance(el, part), parts); msg != "" {
					return msg
				}
			}
			continue
		}
		switch {
		case o.Attr == "src" && el.Kind != "image":
			return fmt.Sprintf("sets the src of %s, a %s, which only images take", path[0], el.Kind)
		case o.Attr != "":
		case el.Kind == "box":
			return fmt.Sprintf("sets the text of %s, a box, which has none; change its class instead", path[0])
		case el.Kind == "use":
			if _, ok := parts(o.Value); !ok {
				return fmt.Sprintf("swaps %s for %s, which does not exist", path[0], ComponentPath(o.Value))
			}
		default:
			el.Text = o.Value
			if msg := checkElement(el); msg != "" {
				return msg
			}
		}
	}
	return ""
}

// VariantProblems are what is wrong with what the variants of s, the
// component name, change in it, as OverrideProblems tells of a use's.
func VariantProblems(s Screen, name string, parts func(name string) ([]Element, bool)) []string {
	var out []string
	for _, v := range s.Variants {
		use := v.Use(name)
		for _, o := range use.Overrides() {
			if msg := overrideProblem(o, o.Path, name, Instance(use, s.UI), parts); msg != "" {
				out = append(out, fmt.Sprintf("variant %s: %s=%s %s", v.ID, o.Key(), quote(o.Value), msg))
			}
		}
	}
	return out
}
