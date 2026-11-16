package scenraview

import (
	"io/fs"
	"maps"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// flowSuffix ends the name of a design's flow, which lives beside it:
// sign-in-flow.scenra is the flow of sign-in.scenra.
const flowSuffix = "-flow"

// fileName is the file name of the design or flow named name.
func fileName(name string, kind flowfile.Kind) string {
	if kind == flowfile.Chart {
		return name + flowSuffix + ".scenra"
	}
	return name + ".scenra"
}

// stem is the name of the design that the file at path is or whose flow
// it is, and flow says it is named as a flow.
func stem(path string) (name string, flow bool) {
	name = strings.TrimSuffix(path, ".scenra")
	trimmed := strings.TrimSuffix(name, flowSuffix)
	return trimmed, trimmed != name
}

// pairOf is the design and the flow named alike in the folder of the file
// at path, which is one of them.
func pairOf(path string) (design, flow string) {
	name, _ := stem(path)
	return name + ".scenra", name + flowSuffix + ".scenra"
}

// scanLimit is how many files and folders a scan of the open folder looks
// at, so a folder as large as a home folder stays quick to read.
const scanLimit = 20000

// entry is a Scenra file found in the open folder: its path relative to
// the folder, with slashes, what it holds, no kind when it does not
// parse, the name it gives itself and the line of its first design,
// with a component's variants, read again with the file.
type entry struct {
	path string
	kind flowfile.Kind
	name string
	head *flowfile.Screen
}

// screen is the line of the file's first design, without its elements.
func (en entry) screen() flowfile.Screen {
	if en.head == nil {
		return flowfile.Screen{}
	}
	return *en.head
}

// scanned is what a scan read of a file, kept while the file stays the
// same: a component's elements too, or a picture's bytes.
type scanned struct {
	stamp stamp
	entry entry
	part  flowfile.Part
	file  *flowfile.File
	data  []byte
}

// components are the components the open folder's designs use, by name.
type components map[string]flowfile.Part

func sameComponents(a, b components) bool {
	return maps.EqualFunc(a, b, func(x, y flowfile.Part) bool {
		return flowfile.FormatUI(x.UI) == flowfile.FormatUI(y.UI) && slices.EqualFunc(x.Variants, y.Variants, func(v, w flowfile.Variant) bool {
			return v.ID == w.ID && maps.Equal(v.Attrs, w.Attrs)
		})
	})
}

// find finds a component of parts by its name.
func (parts components) find(name string) (flowfile.Part, bool) {
	p, ok := parts[name]
	return p, ok
}

// assets are the pictures of the open folder's assets folder, by their
// path in the folder with slashes.
type assets map[string]asset

// asset is a picture's bytes and the version of its file they are, which
// tells pictures apart without comparing their bytes. The bytes stay the
// same slice while the file does, so NSGo keeps the picture it decoded.
type asset struct {
	stamp stamp
	data  []byte
}

func sameAssets(a, b assets) bool {
	return maps.EqualFunc(a, b, func(x, y asset) bool { return x.stamp == y.stamp })
}

// assetLimit is the size of the largest picture a scan reads, so a stray
// video in the assets folder does not fill the memory.
const assetLimit = 20 << 20

// skipped says a scan leaves out the folder or file named name: hidden
// ones and installed packages.
func skipped(name string) bool { return strings.HasPrefix(name, ".") || name == "node_modules" }

// scan finds the Scenra files under root, those outside the project's
// layout too, which the server tells of, and the pictures of its assets
// folder, reading again only those that changed since the scan whose
// files were cache, and returns the Scenra files, the components among
// them, the pictures, the cache of this one and the files of designs.
func scan(root string, cache map[string]scanned) ([]entry, components, assets, map[string]scanned, designs) {
	var out []entry
	parts := components{}
	docs := designs{}
	pics := assets{}
	next := map[string]scanned{}
	seen := 0
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if seen++; seen > scanLimit {
			return fs.SkipAll
		}
		if p == root {
			return err
		}
		if err != nil || skipped(d.Name()) {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if flowfile.IsAsset(rel) {
			st := stampOf(p)
			c, ok := cache[p]
			if !ok || c.stamp != st {
				c = scanned{stamp: st}
				if st.size > 0 && st.size <= assetLimit {
					c.data, _ = os.ReadFile(p)
				}
			}
			next[p] = c
			if len(c.data) > 0 {
				pics[rel] = asset{c.stamp, c.data}
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".scenra") {
			return nil
		}
		st := stampOf(p)
		c, ok := cache[p]
		if !ok || c.stamp != st {
			c = scanned{stamp: st}
			if f, err := readFile(p); err == nil {
				c.entry.kind, c.entry.name = f.Kind, f.Title()
				if len(f.Screens) > 0 {
					head := f.Screens[0]
					head.UI = nil
					c.entry.head = &head
				}
				if f.Kind == flowfile.Design {
					c.file = &f
				}
				if f.Kind == flowfile.Component && len(f.Screens) > 0 {
					c.part = flowfile.Part{UI: f.Screens[0].UI, Variants: f.Screens[0].Variants}
				}
			}
		}
		c.entry.path = rel
		next[p] = c
		out = append(out, c.entry)
		if name, ok := flowfile.ComponentName(c.entry.path); ok && c.entry.kind == flowfile.Component {
			parts[name] = c.part
		}
		if c.file != nil && inLayout(rel) {
			docs[strings.TrimSuffix(rel, ".scenra")] = *c.file
		}
		return nil
	})
	return out, parts, pics, next, docs
}

// scanNow has the scan read the open folder now.
func (e *Editor) scanNow() {
	select {
	case e.rescan <- struct{}{}:
	default:
	}
}

// found adds a file the editor made to those found before the scan does.
func (e *Editor) found(en entry) {
	files := slices.DeleteFunc(slices.Clone(e.files.GetUntracked()), func(f entry) bool { return f.path == en.path })
	e.files.Set(append(files, en))
}

// scannedFiles takes the files a scan of dir found and what Git says
// changed in them, and opens those the
// agent made in its turn: a component only when it made nothing else, so
// the screen using it shows.
func (e *Editor) scannedFiles(dir string, files []entry, parts components, docs designs, pics assets, diffs changes, first bool) {
	if e.folder.GetUntracked() != dir {
		return
	}
	before := e.files.GetUntracked()
	e.cx.Batch(func() {
		e.files.Set(files)
		e.parts.Set(parts)
		e.designs.Set(docs)
		e.assets.Set(pics)
		e.changes.Set(diffs)
	})
	if first || !e.a.busy() {
		return
	}
	var made []entry
	for _, f := range files {
		if !slices.ContainsFunc(before, func(b entry) bool { return b.path == f.path }) && f.kind != "" && inLayout(f.path) {
			made = append(made, f)
		}
	}
	if slices.ContainsFunc(made, func(f entry) bool { return f.kind != flowfile.Component }) {
		made = slices.DeleteFunc(made, func(f entry) bool { return f.kind == flowfile.Component })
	}
	for _, f := range made {
		e.open(filepath.Join(dir, filepath.FromSlash(f.path)))
	}
}

// inLayout says the Scenra file at path, relative to the open folder with
// slashes, is where a project keeps its files: a design or flow at the
// folder's top, or a component in its components folder. The editor
// shows only those, and the server tells agents to move the others.
func inLayout(path string) bool {
	return !strings.Contains(path, "/") || isComponentFile(path)
}

// groupOf splits a file's name into the group before its slash, which
// the sidebar and the showcase list it under, and the name after it.
// Groups do not nest, so a name of several slashes, like one of none, is
// in no group.
func groupOf(name string) (group, rest string) {
	group, rest, ok := strings.Cut(name, "/")
	group, rest = strings.TrimSpace(group), strings.TrimSpace(rest)
	if !ok || group == "" || rest == "" || strings.Contains(rest, "/") {
		return "", name
	}
	return group, rest
}

// nestedGroups says name puts its file in a group of a group, which no
// list shows.
func nestedGroups(name string) bool { return strings.Count(name, "/") > 1 }

// keyName is the name of the file at path without its folder and
// extension.
func keyName(path string) string {
	return strings.TrimSuffix(pathpkg.Base(path), ".scenra")
}

// title is the name a file shows by: the one it gives itself, or its
// file name's.
func (en entry) title() string {
	if en.name != "" {
		return en.name
	}
	return keyName(en.path)
}

// fileGroup is a group of the sidebar: its name, "" for the files in
// none, and its files.
type fileGroup struct {
	name  string
	files []entry
}

// groupFiles groups the designs and flows of files, those in no group
// first and then the groups by name, each holding its files by the names
// they show, a flow after its design and in its group, so the two stay
// together whatever the flow calls itself.
func groupFiles(files []entry) []fileGroup {
	byPath := map[string]entry{}
	for _, f := range files {
		byPath[f.path] = f
	}
	// owner is the file a file sorts and groups by: a flow's design, when
	// the folder has it.
	owner := func(f entry) entry {
		if design, flow := stem(f.path); flow {
			if d, ok := byPath[design+".scenra"]; ok {
				return d
			}
		}
		return f
	}
	groups := map[string][]entry{}
	for _, f := range files {
		if !inLayout(f.path) || isComponentFile(f.path) {
			continue
		}
		g, _ := groupOf(owner(f).title())
		groups[g] = append(groups[g], f)
	}
	names := slices.SortedFunc(maps.Keys(groups), func(a, b string) int {
		if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
			return c
		}
		return strings.Compare(a, b)
	})
	out := make([]fileGroup, 0, len(names))
	for _, g := range names {
		list := groups[g]
		slices.SortFunc(list, func(a, b entry) int {
			_, na := groupOf(owner(a).title())
			_, nb := groupOf(owner(b).title())
			if c := strings.Compare(strings.ToLower(na), strings.ToLower(nb)); c != 0 {
				return c
			}
			sa, _ := stem(owner(a).path)
			sb, _ := stem(owner(b).path)
			if c := strings.Compare(sa, sb); c != 0 {
				return c
			}
			// A design's flow follows it.
			_, fa := stem(a.path)
			_, fb := stem(b.path)
			switch {
			case fa == fb:
				return strings.Compare(a.path, b.path)
			case fa:
				return 1
			}
			return -1
		})
		out = append(out, fileGroup{g, list})
	}
	return out
}
