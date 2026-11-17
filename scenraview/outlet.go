package scenraview

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo/core"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// designs are the files of designs of the open folder, by their names,
// their file names without .scenra, as taps and in= name them.
type designs map[string]flowfile.File

func sameDesigns(a, b designs) bool {
	return maps.EqualFunc(a, b, func(x, y flowfile.File) bool { return x.Format() == y.Format() })
}

// with is ds with the file named name f, as the editor has it.
func (ds designs) with(name string, f flowfile.File) designs {
	out := maps.Clone(ds)
	if out == nil {
		out = designs{}
	}
	if name != "" && f.Kind == flowfile.Design {
		out[name] = f
	}
	return out
}

// place is a design of the folder: its file's name and its index there.
type place struct {
	name  string
	index int
}

// hop is a design holding another in an outlet: where it is and the path
// of the outlet among its elements.
type hop struct {
	place
	outlet []int
}

// hosts are the designs holding the design at p, from the one whose
// outlet it fills to one filling none, or what is wrong with them: a
// file, design or outlet in= names that is not there, or designs filling
// each other in a loop.
func (ds designs) hosts(p place) ([]hop, string) {
	var out []hop
	seen := []place{p}
	for {
		f, ok := ds[p.name]
		if !ok || p.index >= len(f.Screens) {
			return out, ""
		}
		h, ok := f.HostOf(p.index)
		if !ok {
			return out, ""
		}
		name := h.File
		if name == "" {
			name = p.name
		}
		hf, ok := ds[name]
		in := fmt.Sprintf("in=%q", h.String())
		if !ok {
			return nil, fmt.Sprintf("%s: there is no design %s, at the top or in %s/", in, name, viewsDir)
		}
		i, ok := hf.Screen(h.Design)
		if !ok {
			return nil, fmt.Sprintf("%s: %s has no design %s", in, name, h.Design)
		}
		at, ok := flowfile.Outlet(hf.Screens[i].UI, h.Outlet)
		if !ok {
			return nil, fmt.Sprintf("%s: no box of %s has outlet=%s", in, ds.title(place{name, i}), h.Outlet)
		}
		next := place{name, i}
		if slices.Contains(seen, next) {
			var chain []string
			for _, s := range append(seen[slices.Index(seen, next):], next) {
				chain = append(chain, ds.title(s))
			}
			return nil, fmt.Sprintf("%s: designs fill each other in a loop, %s; take an in= out of it", in, strings.Join(chain, " in "))
		}
		seen = append(seen, next)
		out = append(out, hop{next, at})
		p = next
	}
}

// title names the design at p in a message as taps do: its file's name,
// with # and its ID in a file of several.
func (ds designs) title(p place) string {
	f := ds[p.name]
	if f.Several() && p.index < len(f.Screens) {
		return p.name + "#" + f.Screens[p.index].ID
	}
	return p.name
}

// route is what the outlets of hops show so the design d shows in the
// one of hops[0], keyed from prefix, the key the element at the top of
// hops[top] is drawn under: each outlet from hops[top] down by its key,
// in order, and the design it shows.
func route(hops []hop, top int, prefix string, d place) []filled {
	var out []filled
	key := prefix
	for m := top; m >= 0; m-- {
		key += flowfile.FormatPath(hops[m].outlet)
		shows := d
		if m > 0 {
			shows = hops[m-1].place
		}
		out = append(out, filled{key, shows})
		key += outletSep
	}
	return out
}

// filled is the design an outlet shows, by the outlet's key.
type filled struct {
	key string
	place
}

// outletSep joins the key of an outlet and the paths of the elements of
// the design it shows.
const outletSep = "|"

// fills are the designs the outlets of the designs played show, by the
// outlets' keys, the latest last, which closing returns from.
type fills map[string][]place

func sameFills(a, b fills) bool { return maps.EqualFunc(a, b, slices.Equal[[]place]) }

// show has the outlets show what steps lead to, keeping what each showed
// before to return to, and dropping what the outlets inside the ones
// changing showed.
func (fs fills) show(steps []filled) fills {
	out := maps.Clone(fs)
	if out == nil {
		out = fills{}
	}
	for _, s := range steps {
		stack := out[s.key]
		if n := len(stack); n > 0 && stack[n-1] == s.place {
			continue
		}
		out.drop(s.key + outletSep)
		out[s.key] = append(slices.Clone(stack), s.place)
	}
	return out
}

// drop forgets what the outlets whose keys start with prefix show.
func (fs fills) drop(prefix string) {
	for key := range fs {
		if strings.HasPrefix(key, prefix) {
			delete(fs, key)
		}
	}
}

// top is the design the outlet at key shows, none for its own elements.
func (fs fills) top(key string) (place, bool) {
	stack := fs[key]
	if len(stack) == 0 {
		return place{}, false
	}
	return stack[len(stack)-1], true
}

// fillOf is what the outlet at key shows, by fs, drawn from ds: the
// elements of the design, with its taps naming its file.
func fillOf(fs fills, ds designs, key string) ([]flowfile.Element, bool) {
	p, ok := fs.top(key)
	if !ok {
		return nil, false
	}
	f, ok := ds[p.name]
	if !ok || p.index >= len(f.Screens) {
		return nil, false
	}
	return flowfile.InFile(f.Screens[p.index].UI, p.name), true
}

// hosting is where a design filling an outlet is drawn: the size the
// outlet lays out at and the device of the design at the top holding it.
type hosting struct {
	size   core.Size
	device string
}

// sizeOf is the size the design of f at i is drawn at: the one it sets,
// or else the outlet's it fills, as hosted measured it, or else its
// device's.
func sizeOf(f flowfile.File, i int, hosted map[int]hosting) (w, h int) {
	w, h = f.Size(i)
	host, ok := hosted[i]
	if _, in := f.HostOf(i); !in || !ok {
		return w, h
	}
	if f.Screens[i].Width == 0 {
		w = int(host.size.Width + 0.5)
	}
	if f.Screens[i].Height == 0 {
		h = int(host.size.Height + 0.5)
	}
	return w, h
}

// outletProblems are what is wrong with the outlets of the designs of f,
// named name, and those their in= names, found in ds.
func outletProblems(f flowfile.File, name string, ds designs) []string {
	if f.Kind != flowfile.Design {
		return nil
	}
	var out []string
	for i, s := range f.Screens {
		out = append(out, flowfile.OutletProblems(s.UI)...)
		if s.In == "" && i > 0 {
			continue
		}
		if _, msg := ds.hosts(place{name, i}); msg != "" && !slices.Contains(out, msg) {
			out = append(out, msg)
		}
	}
	return out
}

// composed is the design at the top of hops, with what its outlets show
// so d shows in the one of hops[0], drawn as the design at index 0 of a
// file of it alone, and the key of the outlet showing d.
func composed(ds designs, hops []hop, d place) (flowfile.File, fills, string) {
	top := hops[len(hops)-1]
	f := ds[top.name]
	steps := route(hops, len(hops)-1, screenPrefix(0), d)
	one := flowfile.File{Kind: flowfile.Design, Screens: []flowfile.Screen{f.Screens[top.index]}}
	one.Screens[0].UI = flowfile.InFile(one.Screens[0].UI, top.name)
	return one, fills{}.show(steps), steps[len(steps)-1].key
}
