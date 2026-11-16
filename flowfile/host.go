package flowfile

import (
	"fmt"
	"slices"
	"strings"
)

// A box of a design with outlet=<name> is a place other designs fill, as
// a window's content area beside its sidebar: what it holds shows until
// one does. A design with in="<file>#<outlet>" fills the outlet of that
// file's first design, and in="<file>#<design>/<outlet>" that of another
// design of it; the file is the design's path in the project without
// .scenra, as taps name it, or nothing for the same file. The design draws
// alone at the size the outlet lays out at, unless it sets one, and plays
// in its host, which may fill an outlet of its own, so the parts of a
// window around it stay and keep working. The first design's in= stands
// for the file's other designs that write none.

// Host is the outlet a design fills: the name of its file, "" for the
// same file, the ID of its design, "" for the file's first, and its name.
type Host struct{ File, Design, Outlet string }

// String writes h as an in= attribute.
func (h Host) String() string {
	s := h.File + "#"
	if h.Design != "" {
		s += h.Design + "/"
	}
	return s + h.Outlet
}

// ParseHost reads an in= attribute.
func ParseHost(s string) (Host, error) {
	file, rest, ok := strings.Cut(strings.TrimSpace(s), "#")
	design, outlet, two := strings.Cut(rest, "/")
	if !two {
		design, outlet = "", rest
	}
	if !ok || file != "" && !ValidDesignName(file) || two && !ValidID(design) || !ValidName(outlet) {
		return Host{}, fmt.Errorf("in=%q names no outlet; write the design's file without .scenra, # and the outlet's name, such as in=\"main#content\", with the design's ID and / before the outlet for another design of the file, such as in=\"settings#general/content\"", s)
	}
	return Host{file, design, outlet}, nil
}

// HostOf is the outlet the design at i fills, the first design's for one
// that names none, and false for one filling none.
func (f File) HostOf(i int) (Host, bool) {
	if f.Kind != Design || i >= len(f.Screens) {
		return Host{}, false
	}
	in := f.Screens[i].In
	if in == "" && i > 0 {
		in = f.Screens[0].In
	}
	if in == "" {
		return Host{}, false
	}
	h, err := ParseHost(in)
	return h, err == nil
}

// Outlet returns the path of the box of els that is the outlet name.
func Outlet(els []Element, name string) ([]int, bool) {
	for i, el := range els {
		if el.Attr("outlet") == name {
			return []int{i}, true
		}
		if at, ok := Outlet(el.Children, name); ok {
			return append([]int{i}, at...), true
		}
	}
	return nil, false
}

// OutletProblems are what is wrong with the outlets of els, a design's
// elements: a name two outlets have, which designs could not tell apart.
func OutletProblems(els []Element) []string {
	var seen, out []string
	var walk func(els []Element)
	walk = func(els []Element) {
		for _, el := range els {
			if name := el.Attr("outlet"); name != "" {
				if slices.Contains(seen, name) && !slices.Contains(out, name) {
					out = append(out, name)
				}
				seen = append(seen, name)
			}
			walk(el.Children)
		}
	}
	walk(els)
	for i, name := range out {
		out[i] = fmt.Sprintf("outlet %s is written twice; give each outlet its own name", name)
	}
	return out
}

// Resolve is the tap t, written in the file of designs name, naming its
// file: a design of the same file is one of name.
func (t Tap) Resolve(name string) Tap {
	if t.Action != TapOpen && t.Action != TapClose || !strings.HasPrefix(t.Target, "#") {
		return t
	}
	return Tap{t.Action, name + t.Target}
}

// InFile returns els, the elements of the file of designs name, with the
// designs their taps open named by their file, theirs and those their
// uses give, so they open the same designs drawn in another file's
// outlet. els is unchanged.
func InFile(els []Element, name string) []Element {
	out := cloneElements(els)
	var walk func(els []Element)
	walk = func(els []Element) {
		for i := range els {
			el := &els[i]
			for key, value := range el.Attrs {
				if key != "tap" && !strings.HasSuffix(key, ".tap") {
					continue
				}
				if t, err := ParseTap(value); err == nil {
					el.Attrs[key] = t.Resolve(name).String()
				}
			}
			walk(el.Children)
		}
	}
	walk(out)
	return out
}
