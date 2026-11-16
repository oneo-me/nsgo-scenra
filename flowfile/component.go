package flowfile

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// ComponentsDir is the folder of a project's components, at its top and
// without folders of its own: the component tab-bar is
// components/tab-bar.scenra.
const ComponentsDir = "components"

// ValidComponentName reports whether name can name a component: an ID,
// the name of a file of the components folder without .scenra.
func ValidComponentName(name string) bool {
	return ValidID(name) && name != "." && name != ".."
}

// ComponentPath is the path of the component named name, relative to the
// project's folder, with slashes.
func ComponentPath(name string) string { return ComponentsDir + "/" + name + ".scenra" }

// ComponentName is the name of the component at the path, relative to
// the project's folder with slashes, when the path is a file of the
// components folder.
func ComponentName(p string) (string, bool) {
	rest, ok := strings.CutPrefix(path.Clean(p), ComponentsDir+"/")
	name, scenra := strings.CutSuffix(rest, ".scenra")
	if !ok || !scenra || !ValidComponentName(name) {
		return "", false
	}
	return name, true
}

// Uses are the names of the components els use, each once, in the order
// they are first written.
func Uses(els []Element) []string {
	var out []string
	var walk func(els []Element)
	walk = func(els []Element) {
		for _, el := range els {
			if el.Kind == "use" && !slices.Contains(out, el.Text) {
				out = append(out, el.Text)
			}
			walk(el.Children)
		}
	}
	walk(els)
	return out
}

// A component's slot is where a use puts elements of its own, as Vue's
// and web components' slots do: the use's fill of the slot's name holds
// them, written after the use up to its end, and the slot draws them as
// a box holding them, or what it holds itself when no fill does. A
// slot without a name is the default one, which a fill without one
// fills. What a fill holds is the design's, or the component's using
// it: it selects itself there and takes comments, and a slot of that
// component inside it passes on what that component's own use fills.
// A slot's name names it as name= does: variants and uses add to its
// classes by it, and change nothing else of it.

// slotName names the slot called name in a message.
func slotName(name string) string {
	if name == "" {
		return "the default slot"
	}
	return "slot " + name
}

// Slots are the names of the slots of els, a component's elements, each
// once, in the order they are first written, including those in the
// fills of their uses, which pass on the component's own fills.
func Slots(els []Element) []string {
	var out []string
	var walk func(els []Element)
	walk = func(els []Element) {
		for _, el := range els {
			if el.Kind == "slot" && !slices.Contains(out, el.Text) {
				out = append(out, el.Text)
			}
			walk(el.Children)
		}
	}
	walk(els)
	return out
}

// SlotProblems are what is wrong with the slots of els, a component's
// elements: a name two slots have, which would draw a fill twice.
func SlotProblems(els []Element) []string {
	var out, seen []string
	var walk func(els []Element)
	walk = func(els []Element) {
		for _, el := range els {
			if el.Kind == "slot" {
				if slices.Contains(seen, el.Text) && !slices.Contains(out, el.Text) {
					out = append(out, el.Text)
				}
				seen = append(seen, el.Text)
			}
			walk(el.Children)
		}
	}
	walk(els)
	for i, name := range out {
		out[i] = fmt.Sprintf("%s is written twice; give each slot its own name", slotName(name))
	}
	return out
}

// FillProblems are what is wrong with the fills of use, whose component
// parts finds by name: fills of slots it does not have. A use of a
// missing component has none.
func FillProblems(use Element, parts Parts) []string {
	part, ok := parts(use.Text)
	if !ok {
		return nil
	}
	slots := Slots(part.UI)
	var out []string
	for _, fill := range use.Children {
		if slices.Contains(slots, fill.Text) {
			continue
		}
		msg := fmt.Sprintf("use %q: the fill of %s fills nothing, as %s has no such slot", use.Text, slotName(fill.Text), ComponentPath(use.Text))
		if len(slots) > 0 {
			names := make([]string, len(slots))
			for i, s := range slots {
				names[i] = slotName(s)
			}
			msg += "; it has " + strings.Join(names, ", ")
		}
		out = append(out, msg)
	}
	return out
}
