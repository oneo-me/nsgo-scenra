package flowfile

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// A component's elements may take a name, by which a use of it changes
// them where it is, as Figma's instances and Sketch's symbols override
// their layers: use card title="Latte" draws the card with the text of
// its elements named title replaced, title.class="text-rose-600" adds
// to their classes, and title.src=, title.tap= and title.panel= replace
// their image, tap and panel. The text of an icon is its name, and that
// of a use the component it draws, which a use swaps. row/title reaches
// the elements named title of the component the use named row draws, and
// so on down. Every element of a name changes, so one name may stand
// for several elements that show one value. A slot answers to its own
// name too, besides any name= it has, but a use changes only its
// classes: what a fill holds is the using file's.
//
// A use picks the component's variants by their IDs with variant=, and
// row.variant= those of the component its element row draws: the
// component draws first, then each variant in the order written, then
// what the use changes itself, the later winning where they meet, as
// classes written later and texts set again do. A variant may pick
// others of its component the same way, as the state it starts from.
//
// A use's state= puts the component's element at its top in states,
// such as selected or hover, as pointing at it or tapping would: its
// selected: or hover: classes apply, and the group-selected: or
// group-hover: classes of what any group inside it holds; selected@option
// puts the elements named option in the state instead.

// OverrideAttrs are the attributes of a named element a use may change
// besides its text: classes, which add to its own, glass, those replacing its
// own, a text's message key, the note the using file keeps on it, and the variants and states
// of what it draws.
var OverrideAttrs = []string{"class", "glass", "key", "src", "tap", "panel", "note", "variant", "state"}

// useAttrs are the attributes a use takes besides the common ones and
// what it changes by names: the variants of its component it draws,
// the states it puts it in and its Liquid Glass effect.
var useAttrs = []string{"variant", "state", "glass"}

// States are the states state= puts an element in, and the class variant
// each applies.
var States = []string{"selected", "hover", "pressed", "disabled", "focus"}

// stateVariant is the class variant of a state: Tailwind calls pressed
// active.
func stateVariant(state string) string {
	if state == "pressed" {
		return "active"
	}
	return state
}

// ReservedNames are the words no element may be named, as they are the
// attributes elements and uses take.
func ReservedNames() []string {
	out := slices.Concat(commonAttrs, useAttrs)
	for _, s := range Elements {
		for _, a := range s.Attrs {
			if !slices.Contains(out, a) {
				out = append(out, a)
			}
		}
	}
	return out
}

// ValidName reports whether name can name an element of a component:
// letters, digits, _ and -, and no attribute an element takes.
func ValidName(name string) bool {
	return ValidID(name) && !strings.Contains(name, ".") && !slices.Contains(ReservedNames(), name)
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
				key, strings.Join(slices.Concat(commonAttrs, useAttrs), ", "), strings.Join(OverrideAttrs, ", ."))
		}
	}
	if attr != "" && !slices.Contains(OverrideAttrs, attr) {
		return Override{}, fmt.Sprintf("use cannot change %q of %s; change its text with %s=, or its %s", attr, names, names, strings.Join(OverrideAttrs, ", "))
	}
	if msg := checkUseAttr(attr, value); msg != "" {
		return Override{}, msg
	}
	return o, ""
}

// checkUseAttr says what is wrong with the value of an attribute a use
// sets, on itself or by a name.
func checkUseAttr(attr, value string) string {
	switch attr {
	case "glass":
		if _, err := ParseGlass(value); err != nil {
			return err.Error()
		}
	case "tap":
		if _, err := ParseTap(value); err != nil {
			return err.Error()
		}
	case "key":
		if !ValidKey(value) {
			return keyHelp(value)
		}
	case "panel":
		if !ValidID(value) {
			return fmt.Sprintf("panel %q is not a name of letters, digits, _, - or .", value)
		}
	case "variant":
		ids := strings.Fields(value)
		if len(ids) == 0 {
			return "variant= takes the IDs of the component's variants, apart by spaces, such as variant=\"selected waiting\""
		}
		for _, id := range ids {
			if !ValidID(id) {
				return fmt.Sprintf("variant %q is not an ID of letters, digits, _, - or .", id)
			}
		}
	case "state":
		if _, msg := ParseStates(value); msg != "" {
			return msg
		}
	}
	return ""
}

// StateOf is a state state= puts elements in: the elements named Target,
// or the component's element at its top for "".
type StateOf struct{ State, Target string }

// ParseStates reads a state= attribute: states apart by spaces, each
// with @ and the name of the elements it puts in it, if not the top.
func ParseStates(s string) ([]StateOf, string) {
	var out []StateOf
	for _, word := range strings.Fields(s) {
		state, target, at := strings.Cut(word, "@")
		if !slices.Contains(States, state) {
			return nil, fmt.Sprintf("unknown state %q; use %s, and @<name> for the elements of that name", state, strings.Join(States, ", "))
		}
		if at && !ValidName(target) {
			return nil, fmt.Sprintf("state %q names no element; write @ and the name= of one, such as selected@option", word)
		}
		out = append(out, StateOf{state, target})
	}
	if len(out) == 0 {
		return nil, "state= takes states such as selected or hover"
	}
	return out, ""
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

// Part is a component as its uses draw it: its elements and variants,
// and the path of its file, which messages name, when it is not the
// component's path, as for a design drawn as a component.
type Part struct {
	Path     string
	UI       []Element
	Variants []Variant
}

// PartPath is the path of the file of the component parts finds by name:
// its own, or else its path in the components folder.
func PartPath(name string, parts Parts) string {
	if p, ok := parts(name); ok && p.Path != "" {
		return p.Path
	}
	return ComponentPath(name)
}

// Variant returns the variant of p with id.
func (p Part) Variant(id string) (Variant, bool) {
	i := slices.IndexFunc(p.Variants, func(v Variant) bool { return v.ID == id })
	if i < 0 {
		return Variant{}, false
	}
	return p.Variants[i], true
}

// Parts finds a component by its name.
type Parts func(name string) (Part, bool)

// setting is an attribute a use sets, in the order it applies.
type setting struct{ key, value string }

// settings are what attrs, a use's or a variant's of part, set in the
// order they apply: the variants they pick, each once and none of
// those picking it, then their class, what they change by names and
// their states.
func settings(attrs map[string]string, part Part, picking []string) []setting {
	var out []setting
	for _, id := range strings.Fields(attrs["variant"]) {
		v, ok := part.Variant(id)
		if !ok || slices.Contains(picking, id) {
			continue
		}
		out = append(out, settings(v.Attrs, part, append(slices.Clone(picking), id))...)
	}
	if class, ok := attrs["class"]; ok {
		out = append(out, setting{"class", class})
	}
	if glass, ok := attrs["glass"]; ok {
		out = append(out, setting{"glass", glass})
	}
	for _, key := range overrideKeys(attrs) {
		out = append(out, setting{key, attrs[key]})
	}
	if state, ok := attrs["state"]; ok {
		out = append(out, setting{"state", state})
	}
	return out
}

// Settled is what use sets in part, the variants it picks resolved, as
// the attributes of a use picking no variant: classes and states joined
// in the order they apply, and the last value of anything else.
func Settled(use Element, part Part) map[string]string {
	out := map[string]string{}
	for _, s := range settings(use.Attrs, part, nil) {
		if old := out[s.key]; old != "" && joins(s.key) {
			s.value = old + " " + s.value
		}
		out[s.key] = s.value
	}
	return out
}

// joins says the values an attribute is set to add up: classes, which
// win in the order written, variants and states.
func joins(key string) bool {
	_, attr, _ := strings.Cut(key, ".")
	if !strings.Contains(key, ".") {
		attr = key
	}
	return attr == "class" || attr == "variant" || attr == "state"
}

// UseClass is the classes use gives the component it draws, those of the
// variants it picks first: a component of several elements at its top
// draws as a box of them.
func UseClass(use Element, part Part) string { return Settled(use, part)["class"] }

// UseGlass is the Liquid Glass effect use gives the component it draws,
// from its variants and itself, the last winning.
func UseGlass(use Element, part Part) string { return Settled(use, part)["glass"] }

// Instance returns the elements of part, the component use draws, as it
// draws them: changed by the variants it picks, in order, then by what it
// changes itself, and what it changes inside the components those
// elements use given to them, overriding what they change themselves.
// The classes go to the element at the top of a component of one, after
// its own so they win, as React's className goes to a component's root:
// the use draws as it rather than as a box holding it, so w-full and
// flex-1 work where the use is. Last, the states it sets apply. part is
// unchanged.
func Instance(use Element, part Part) []Element {
	sets := settings(use.Attrs, part, nil)
	if len(sets) == 0 {
		return part.UI
	}
	out := cloneElements(part.UI)
	for _, s := range sets {
		switch s.key {
		case "class":
			if Rooted(out) {
				out[0].setAttr("class", "class", s.value)
			}
		case "glass":
			if Rooted(out) && (out[0].Kind == "box" || out[0].Kind == "slot" || out[0].Kind == "use") {
				out[0].setAttr("glass", "glass", s.value)
			}
		case "state":
			for _, word := range strings.Fields(s.value) {
				if strings.HasPrefix(word, groupOnly) || strings.HasPrefix(word, heldOnly) {
					// A state of what holds the use reaches its top as is.
					for i := range out {
						out[i].setAttr("state", "state", word)
					}
					continue
				}
				states, _ := ParseStates(word)
				if len(states) == 0 {
					continue
				}
				st := states[0]
				if st.Target == "" {
					for i := range out {
						out[i].setAttr("state", "state", st.State)
					}
					continue
				}
				eachNamed(out, st.Target, func(el *Element) { el.setAttr("state", "state", st.State) })
			}
		default:
			o, msg := parseOverride(s.key, s.value)
			if msg != "" {
				continue
			}
			eachNamed(out, o.Path[0], func(el *Element) { el.override(o) })
		}
	}
	settleStates(out)
	return out
}

// eachNamed runs fn on the elements of els, and of the boxes they are in,
// with the name, outside the components they use.
func eachNamed(els []Element, name string, fn func(el *Element)) {
	for i := range els {
		if els[i].called(name) {
			fn(&els[i])
		}
		eachNamed(els[i].Children, name, fn)
	}
}

// Rooted says a component of elements part has one at its top, which a
// use draws in its place.
func Rooted(part []Element) bool { return len(part) == 1 }

// called says the element answers to name: its name=, or a slot's own
// name.
func (el Element) called(name string) bool {
	return el.Attr("name") == name || el.Kind == "slot" && el.Text != "" && el.Text == name
}

// override changes el as o, which names el first.
func (el *Element) override(o Override) {
	if el.Kind == "slot" && (len(o.Path) > 1 || o.Attr != "class" && o.Attr != "glass") {
		// A use only shapes a slot; what it shows is the fill's.
		return
	}
	if len(o.Path) > 1 {
		if el.Kind == "use" {
			inner := o
			inner.Path = o.Path[1:]
			el.setAttr(inner.Key(), inner.Attr, o.Value)
		}
		return
	}
	switch {
	case o.Attr == "variant" && el.Kind != "use":
	case o.Attr != "":
		el.setAttr(o.Attr, o.Attr, o.Value)
	case el.Kind == "icon":
		el.Text, _ = IconName(o.Value)
	case el.Kind != "box":
		// A text set where the component is used shows instead of its
		// message.
		el.Text = o.Value
		delete(el.Attrs, "key")
	}
}

// setAttr sets the attribute key to value, or adds value to the classes,
// variants or states there when attr is one of those.
func (el *Element) setAttr(key, attr, value string) {
	if el.Attrs == nil {
		el.Attrs = map[string]string{}
	}
	if old := el.Attrs[key]; joins(attr) && old != "" {
		value = old + " " + value
	}
	el.Attrs[key] = value
}

// A state reaches a use inside the element it is set on as a state of
// what that use's component holds alone: groupOnly marks one whose
// group-*: classes apply to all it holds, as a group holds it, and
// heldOnly one whose own element is held, so only its groups pass it on.
const (
	groupOnly = "~"
	heldOnly  = "^"
)

// settleStates applies the states set on els, and on the elements they
// hold, outside the uses, which take theirs when they draw.
func settleStates(els []Element) {
	for i := range els {
		el := &els[i]
		if el.Kind != "use" && el.Attr("state") != "" {
			for _, word := range strings.Fields(el.Attr("state")) {
				state, own, grouped := word, true, false
				if s, ok := strings.CutPrefix(word, groupOnly); ok {
					state, own, grouped = s, false, true
				} else if s, ok := strings.CutPrefix(word, heldOnly); ok {
					state, own = s, false
				}
				state, _, _ = strings.Cut(state, "@")
				el.applyState(stateVariant(state), own, grouped)
			}
			delete(el.Attrs, "state")
		}
		settleStates(el.Children)
	}
}

// applyState applies the class variant of a state to el: its own
// variant: classes when own, and the group-variant: classes of what it
// holds inside a group, which grouped says el is in.
func (el *Element) applyState(variant string, own, grouped bool) {
	if own {
		el.promote(variant)
	}
	if grouped && !own {
		el.promote("group-" + variant)
	}
	inside := grouped || isGroup(el.Attr("class"))
	for i := range el.Children {
		kid := &el.Children[i]
		if kid.Kind == "use" {
			mark := heldOnly
			if inside {
				mark = groupOnly
			}
			kid.setAttr("state", "state", mark+stateOf(variant))
			continue
		}
		kid.applyState(variant, false, inside)
	}
}

// stateOf is the state whose class variant is variant.
func stateOf(variant string) string {
	if variant == "active" {
		return "pressed"
	}
	return variant
}

// isGroup says classes mark their element a group, whose group-*:
// classes inside follow.
func isGroup(class string) bool {
	for _, c := range strings.Fields(class) {
		if c == "group" || strings.HasPrefix(c, "group/") {
			return true
		}
	}
	return false
}

// promote adds to el's classes those that apply in the class variant,
// without it, so they win over the classes before them.
func (el *Element) promote(variant string) {
	var add []string
	for _, c := range strings.Fields(el.Attr("class")) {
		if p, ok := withoutVariant(c, variant); ok {
			add = append(add, p)
		}
	}
	if len(add) > 0 {
		el.setAttr("class", "class", strings.Join(add, " "))
	}
}

// withoutVariant is class without the variant among those it starts
// with, when it has it: group-hover/name: counts as group-hover:.
func withoutVariant(class, variant string) (string, bool) {
	parts := splitVariants(class)
	for i, p := range parts[:len(parts)-1] {
		name, _, _ := strings.Cut(p, "/")
		if name == variant {
			return strings.Join(slices.Delete(slices.Clone(parts), i, i+1), ":"), true
		}
	}
	return "", false
}

// splitVariants splits a class at the colons ending its variants, those
// outside brackets.
func splitVariants(class string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range class {
		switch r {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case ':':
			if depth == 0 {
				out = append(out, class[start:i])
				start = i + 1
			}
		}
	}
	return append(out, class[start:])
}

// hasStateClasses says els, or the components they draw, have classes
// a state applies: its variant: or group-variant: ones.
func hasStateClasses(els []Element, state string, parts Parts) bool {
	variant := stateVariant(state)
	found := false
	seen := map[string]bool{}
	var visit func(els []Element)
	visit = func(els []Element) {
		for _, el := range els {
			for _, c := range strings.Fields(el.Attr("class")) {
				_, own := withoutVariant(c, variant)
				_, group := withoutVariant(c, "group-"+variant)
				found = found || own || group
			}
			if el.Kind == "use" && !seen[el.Text] {
				seen[el.Text] = true
				if p, ok := parts(el.Text); ok {
					visit(p.UI)
				}
			}
			visit(el.Children)
		}
	}
	visit(els)
	return found
}

// named are the elements of els, and of the boxes they are in, with the
// name, outside the components they use.
func named(els []Element, name string) []Element {
	var out []Element
	for _, el := range els {
		if el.called(name) {
			out = append(out, el)
		}
		out = append(out, named(el.Children, name)...)
	}
	return out
}

// OverrideProblems are what is wrong with what use changes in the
// component it draws, which parts finds by name: variants it does not
// have, states it draws nothing for, names no element of it has, text of
// boxes, src of what is no image, icons, SVG and components that do not
// exist, and names inside elements that are no uses. A use of a missing
// component has none.
func OverrideProblems(use Element, parts Parts) []string {
	part, ok := parts(use.Text)
	if !ok {
		return nil
	}
	var out []string
	if msg := variantProblem(use.Attr("variant"), part); msg != "" {
		out = append(out, fmt.Sprintf("use %s: %s", use.Text, msg))
	}
	drawn := Instance(Element{Kind: "use", Text: use.Text, Attrs: withoutStates(use.Attrs)}, part)
	for _, msg := range stateProblems(use.Attr("state"), use.Text, drawn, parts) {
		out = append(out, fmt.Sprintf("use %s: %s", use.Text, msg))
	}
	for _, o := range use.Overrides() {
		if msg := overrideProblem(o, o.Path, use.Text, drawn, parts); msg != "" {
			out = append(out, fmt.Sprintf("use %q: %s=%s %s", use.Text, o.Key(), quote(o.Value), msg))
		}
	}
	return out
}

// withoutStates are attrs without state=, which OverrideProblems checks
// against the component as it draws without the states.
func withoutStates(attrs map[string]string) map[string]string {
	out := maps.Clone(attrs)
	delete(out, "state")
	return out
}

// variantProblem says which IDs of ids, a variant= attribute, part has no
// variant of.
func variantProblem(ids string, part Part) string {
	for _, id := range strings.Fields(ids) {
		if _, ok := part.Variant(id); !ok {
			msg := fmt.Sprintf("no variant %q", id)
			if len(part.Variants) > 0 {
				var have []string
				for _, v := range part.Variants {
					have = append(have, v.ID)
				}
				msg += "; it has " + strings.Join(have, ", ")
			}
			return msg
		}
	}
	return ""
}

// stateProblems are what is wrong with states, a state= attribute, in
// els, the elements of the component name as drawn: states nothing
// draws, and elements named that are not there.
func stateProblems(states, name string, els []Element, parts Parts) []string {
	list, _ := ParseStates(states)
	var out []string
	for _, st := range list {
		scope := els
		if st.Target != "" {
			scope = named(els, st.Target)
			if len(scope) == 0 {
				out = append(out, fmt.Sprintf("state=%s names nothing: no element of %s has name=%s", st.State+"@"+st.Target, PartPath(name, parts), st.Target))
				continue
			}
		}
		if !hasStateClasses(scope, st.State, parts) {
			v := stateVariant(st.State)
			out = append(out, fmt.Sprintf("state=%s draws nothing: %s has no %s: or group-%s: classes there", st.State, PartPath(name, parts), v, v))
		}
	}
	return out
}

// overrideProblem says what is wrong with o where path is left of it in
// els, the elements of the component name as drawn.
func overrideProblem(o Override, path []string, name string, els []Element, parts Parts) string {
	found := named(els, path[0])
	if len(found) == 0 {
		return fmt.Sprintf("names nothing: no element of %s has name=%s, nor is it a slot's name", PartPath(name, parts), path[0])
	}
	for _, el := range found {
		if el.Kind == "slot" {
			switch {
			case len(path) > 1:
				return fmt.Sprintf("goes inside %s, a slot of %s, not a use; change what a fill holds where it is written", path[0], PartPath(name, parts))
			case o.Attr == "":
				return fmt.Sprintf("sets the text of %s, a slot, which has none; fill it, or change its class", path[0])
			case o.Attr != "class" && o.Attr != "glass":
				return fmt.Sprintf("sets the %s of %s, a slot, which takes only class and glass; set it on what a fill holds", o.Attr, path[0])
			}
			continue
		}
		if len(path) > 1 {
			if el.Kind != "use" {
				return fmt.Sprintf("goes inside %s, a %s of %s, not a use", path[0], el.Kind, PartPath(name, parts))
			}
			if part, ok := parts(el.Text); ok {
				if msg := overrideProblem(o, path[1:], el.Text, Instance(el, part), parts); msg != "" {
					return msg
				}
			}
			continue
		}
		switch {
		case o.Attr == "glass" && el.Kind != "box" && el.Kind != "slot" && el.Kind != "use":
			return fmt.Sprintf("sets the glass of %s, a %s, which only boxes take", path[0], el.Kind)
		case o.Attr == "key" && el.Kind != "text":
			return fmt.Sprintf("sets the key of %s, a %s, which only texts take", path[0], el.Kind)
		case o.Attr == "src" && el.Kind != "image":
			return fmt.Sprintf("sets the src of %s, a %s, which only images take", path[0], el.Kind)
		case o.Attr == "variant" && el.Kind != "use":
			return fmt.Sprintf("picks variants of %s, a %s of %s, not a use", path[0], el.Kind, PartPath(name, parts))
		case o.Attr == "variant":
			if part, ok := parts(el.Text); ok {
				if msg := variantProblem(o.Value, part); msg != "" {
					return fmt.Sprintf("picks %s, a use of %s, which has %s", path[0], PartPath(el.Text, parts), msg)
				}
			}
		case o.Attr == "state":
			if msg := stateProblems(o.Value, name, []Element{el}, parts); len(msg) > 0 {
				return msg[0]
			}
		case o.Attr != "":
		case el.Kind == "box":
			return fmt.Sprintf("sets the text of %s, a box, which has none; change its class instead", path[0])
		case el.Kind == "use":
			if _, ok := parts(o.Value); !ok {
				return fmt.Sprintf("swaps %s for %s, which is neither %s nor a design", path[0], o.Value, ComponentPath(o.Value))
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

// VariantProblems are what is wrong with the variants of s, the
// component name: variants they pick that it does not have, variants
// picking each other in a loop, and what they change in it, as
// OverrideProblems tells of a use's.
func VariantProblems(s Screen, name string, parts Parts) []string {
	part := Part{UI: s.UI, Variants: s.Variants}
	var out []string
	for _, v := range s.Variants {
		if msg := variantProblem(v.Attrs["variant"], part); msg != "" {
			out = append(out, fmt.Sprintf("variant %s: %s", v.ID, msg))
		}
		if chain := variantLoop(part, []string{v.ID}); chain != nil && chain[0] == v.ID {
			out = append(out, fmt.Sprintf("variant %s: variants pick each other in a loop, %s; take a variant= out of it", v.ID, strings.Join(chain, " picks ")))
		}
		use := v.Use(name)
		drawn := Instance(Element{Kind: "use", Text: name, Attrs: withoutStates(use.Attrs)}, part)
		for _, msg := range stateProblems(use.Attr("state"), name, drawn, parts) {
			out = append(out, fmt.Sprintf("variant %s: %s", v.ID, msg))
		}
		for _, o := range use.Overrides() {
			if msg := overrideProblem(o, o.Path, name, drawn, parts); msg != "" {
				out = append(out, fmt.Sprintf("variant %s: %s=%s %s", v.ID, o.Key(), quote(o.Value), msg))
			}
		}
	}
	return out
}

// variantLoop is the chain of variants picking from the last of chain
// back to one of it, nil when there is none.
func variantLoop(part Part, chain []string) []string {
	v, _ := part.Variant(chain[len(chain)-1])
	for _, id := range strings.Fields(v.Attrs["variant"]) {
		next := append(slices.Clone(chain), id)
		if i := slices.Index(chain, id); i >= 0 {
			return next[i:]
		}
		if loop := variantLoop(part, next); loop != nil {
			return loop
		}
	}
	return nil
}

// VariantHints are the variants of part whose changes use writes out
// itself, which it could pick with variant= instead.
func VariantHints(use Element, part Part) []string {
	var out []string
	own := use.Attrs
	for _, v := range part.Variants {
		want := Settled(v.Use(use.Text), part)
		delete(want, "variant")
		if len(want) == 0 || slices.Contains(strings.Fields(use.Attr("variant")), v.ID) {
			continue
		}
		match := true
		for key, value := range want {
			if own[key] != value {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		var keys []string
		for _, key := range slices.Sorted(maps.Keys(want)) {
			keys = append(keys, key+"="+quote(want[key]))
		}
		out = append(out, fmt.Sprintf("use %s: %s is variant %s; write variant=%s instead", use.Text, strings.Join(keys, " "), v.ID, v.ID))
	}
	return out
}
