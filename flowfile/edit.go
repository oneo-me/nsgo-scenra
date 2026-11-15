package flowfile

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// The edits change d in place and say what is wrong instead when they
// cannot; d is unchanged then. Clone a document others hold first.

// AddNode adds n, or fills in the node with its ID.
func (d *Flow) AddNode(n Node) error {
	if err := checkNode(n); err != nil {
		return err
	}
	if n.Label == "" {
		n.Label = n.ID
	}
	if i := d.node(n.ID); i >= 0 {
		d.Nodes[i] = n
		return nil
	}
	d.Nodes = append(d.Nodes, n)
	return nil
}

// NodePatch is what UpdateNode changes: each set field.
type NodePatch struct {
	Label, Note, Kind, Status *string
	// Rename gives the node a new ID, which its edges follow.
	Rename string
}

// UpdateNode changes the node with id.
func (d *Flow) UpdateNode(id string, p NodePatch) error {
	i := d.node(id)
	if i < 0 {
		return fmt.Errorf("no node %q", id)
	}
	n := d.Nodes[i]
	set := func(dst *string, src *string) {
		if src != nil {
			*dst = *src
		}
	}
	set(&n.Label, p.Label)
	set(&n.Note, p.Note)
	set(&n.Kind, p.Kind)
	set(&n.Status, p.Status)
	if n.Label == "" {
		n.Label = n.ID
	}
	if p.Rename != "" && p.Rename != id {
		if d.node(p.Rename) >= 0 {
			return fmt.Errorf("a node %q exists", p.Rename)
		}
		n.ID = p.Rename
	}
	if err := checkNode(n); err != nil {
		return err
	}
	d.Nodes[i] = n
	if n.ID != id {
		for j := range d.Edges {
			if d.Edges[j].From == id {
				d.Edges[j].From = n.ID
			}
			if d.Edges[j].To == id {
				d.Edges[j].To = n.ID
			}
		}
	}
	return nil
}

// RemoveNode removes the node with id and its edges.
func (d *Flow) RemoveNode(id string) error {
	i := d.node(id)
	if i < 0 {
		return fmt.Errorf("no node %q", id)
	}
	d.Nodes = slices.Delete(d.Nodes, i, i+1)
	d.Edges = slices.DeleteFunc(d.Edges, func(e Edge) bool { return e.From == id || e.To == id })
	return nil
}

// AddEdge adds e between nodes that exist, or replaces the label and
// dashing of the edge between them.
func (d *Flow) AddEdge(e Edge) error {
	for _, id := range []string{e.From, e.To} {
		if d.node(id) < 0 {
			return fmt.Errorf("no node %q; add it first", id)
		}
	}
	if e.From == e.To {
		return fmt.Errorf("an edge cannot point from %q to itself", e.From)
	}
	if i := d.edge(e.From, e.To); i >= 0 {
		d.Edges[i] = e
		return nil
	}
	d.Edges = append(d.Edges, e)
	return nil
}

// RemoveEdge removes the edge from one node to another.
func (d *Flow) RemoveEdge(from, to string) error {
	i := d.edge(from, to)
	if i < 0 {
		return fmt.Errorf("no edge from %q to %q", from, to)
	}
	d.Edges = slices.Delete(d.Edges, i, i+1)
	return nil
}

func (d *Flow) edge(from, to string) int {
	return slices.IndexFunc(d.Edges, func(e Edge) bool { return e.From == from && e.To == to })
}

func checkNode(n Node) error {
	if !ValidID(n.ID) {
		return fmt.Errorf("%q is not a node ID; use letters, digits, _, - or .", n.ID)
	}
	if n.Kind != "" && !slices.Contains(Kinds, n.Kind) {
		return fmt.Errorf("unknown kind %q; use one of %s", n.Kind, strings.Join(Kinds, ", "))
	}
	if n.Status != "" && !slices.Contains(Statuses, n.Status) {
		return fmt.Errorf("unknown status %q; use one of %s", n.Status, strings.Join(Statuses, ", "))
	}
	return nil
}

// Element paths name an element of a design by its index among its
// siblings at each level, from 0 and joined by dots: "1.0" is the first
// element the second one holds.

// ParsePath reads an element path.
func ParsePath(path string) ([]int, error) {
	var out []int
	for _, part := range strings.Split(path, ".") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("%q is not an element path such as 1.0", path)
		}
		out = append(out, n)
	}
	return out, nil
}

// FormatPath writes an element path.
func FormatPath(path []int) string {
	parts := make([]string, len(path))
	for i, n := range path {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ".")
}

// ElementAt returns the element of els at path.
func ElementAt(els []Element, path string) (Element, bool) {
	at, err := ParsePath(path)
	if err != nil {
		return Element{}, false
	}
	for i, n := range at {
		if n >= len(els) {
			return Element{}, false
		}
		if i == len(at)-1 {
			return els[n], true
		}
		els = els[n].Children
	}
	return Element{}, false
}

// ReplaceElement returns els with the element at path replaced by with,
// which may be several elements or none; els is unchanged.
func ReplaceElement(els []Element, path string, with []Element) ([]Element, error) {
	at, err := ParsePath(path)
	if err != nil {
		return nil, err
	}
	var replace func(els []Element, at []int) ([]Element, bool)
	replace = func(els []Element, at []int) ([]Element, bool) {
		n := at[0]
		if n >= len(els) {
			return nil, false
		}
		out := slices.Clone(els)
		if len(at) == 1 {
			return slices.Concat(out[:n], with, out[n+1:]), true
		}
		if spec, _ := Spec(out[n].Kind); !spec.Holds {
			return nil, false
		}
		children, ok := replace(out[n].Children, at[1:])
		out[n].Children = children
		return out, ok
	}
	out, ok := replace(els, at)
	if !ok {
		return nil, fmt.Errorf("no element at %s", path)
	}
	return out, nil
}

// Comment is a user's comment or note on an element of a design: the
// index of the design among the file's, the element's path in it and the
// text.
type Comment struct {
	Screen     int
	Path, Text string
}

// Comments are the comments on f's designs, in the order their elements
// are written.
func (f File) Comments() []Comment { return f.remarks("comment") }

// Notes are the notes on f's designs, in the order their elements are
// written.
func (f File) Notes() []Comment { return f.remarks("note") }

// remarks are the attributes key of f's elements that are set.
func (f File) remarks(key string) []Comment {
	var out []Comment
	for s, screen := range f.Screens {
		var walk func(els []Element, parent []int)
		walk = func(els []Element, parent []int) {
			for i, el := range els {
				path := append(slices.Clone(parent), i)
				if text := el.Attr(key); text != "" {
					out = append(out, Comment{s, FormatPath(path), text})
				}
				walk(el.Children, path)
			}
		}
		walk(screen.UI, nil)
	}
	return out
}

// SetComment returns els with the comment on the element at path set to
// text, or removed when text is ""; els is unchanged.
func SetComment(els []Element, path, text string) ([]Element, error) {
	return SetAttr(els, path, "comment", text)
}

// SetNote returns els with the note on the element at path set to text,
// or removed when text is ""; els is unchanged.
func SetNote(els []Element, path, text string) ([]Element, error) {
	return SetAttr(els, path, "note", text)
}

// SetAttr returns els with the attribute key of the element at path set
// to value, or removed when value is ""; els is unchanged.
func SetAttr(els []Element, path, key, value string) ([]Element, error) {
	el, ok := ElementAt(els, path)
	if !ok {
		return nil, fmt.Errorf("no element at %s", path)
	}
	el.Attrs = maps.Clone(el.Attrs)
	if value == "" {
		delete(el.Attrs, key)
	} else {
		if el.Attrs == nil {
			el.Attrs = map[string]string{}
		}
		el.Attrs[key] = value
	}
	return ReplaceElement(els, path, []Element{el})
}
