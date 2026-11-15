package scenraview

import (
	"io/fs"
	"maps"
	"path"
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
// the folder, with slashes, and what it holds, no kind when it does not
// parse.
type entry struct {
	path   string
	kind   flowfile.Kind
	device string
}

// scanned is what a scan read of a file, kept while the file stays the
// same: a component's elements too.
type scanned struct {
	stamp stamp
	entry entry
	ui    []flowfile.Element
}

// components are the elements of the components the open folder's
// designs use, by name.
type components map[string][]flowfile.Element

func sameComponents(a, b components) bool {
	return maps.EqualFunc(a, b, func(x, y []flowfile.Element) bool { return flowfile.FormatUI(x) == flowfile.FormatUI(y) })
}

// skipped says a scan leaves out the folder or file named name: hidden
// ones and installed packages.
func skipped(name string) bool { return strings.HasPrefix(name, ".") || name == "node_modules" }

// scan finds the Scenra files under root, reading again only those that
// changed since the scan whose files were cache, and returns them, the
// components among them and the cache of this one.
func scan(root string, cache map[string]scanned) ([]entry, components, map[string]scanned) {
	var out []entry
	parts := components{}
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
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".scenra") {
			return nil
		}
		st := stampOf(p)
		c, ok := cache[p]
		if !ok || c.stamp != st {
			c = scanned{stamp: st}
			if f, err := readFile(p); err == nil {
				c.entry.kind, c.entry.device = f.Kind, f.Device
				if f.Kind == flowfile.Component {
					c.ui = f.UI
				}
			}
		}
		rel, _ := filepath.Rel(root, p)
		c.entry.path = filepath.ToSlash(rel)
		next[p] = c
		out = append(out, c.entry)
		if name, ok := flowfile.ComponentName(c.entry.path); ok && c.entry.kind == flowfile.Component {
			parts[name] = c.ui
		}
		return nil
	})
	return out, parts, next
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

// scannedFiles takes the files a scan of dir found, and opens those the
// agent made in its turn: a component only when it made nothing else, so
// the screen using it shows.
func (e *Editor) scannedFiles(dir string, files []entry, parts components, first bool) {
	if e.folder.GetUntracked() != dir {
		return
	}
	before := e.files.GetUntracked()
	e.cx.Batch(func() {
		e.files.Set(files)
		e.parts.Set(parts)
	})
	if first || !e.a.busy() {
		return
	}
	var made []entry
	for _, f := range files {
		if !slices.ContainsFunc(before, func(b entry) bool { return b.path == f.path }) && f.kind != "" {
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

// fileTree is the open folder's files as a tree: the keys of each
// folder's folders, then its files, by name. A folder's key is its path
// and a slash, the open folder's "", and a file's its path.
type fileTree struct {
	children map[string][]string
	files    map[string]entry
}

func treeOf(files []entry) fileTree {
	t := fileTree{children: map[string][]string{}, files: map[string]entry{}}
	add := func(parent, key string) {
		if !slices.Contains(t.children[parent], key) {
			t.children[parent] = append(t.children[parent], key)
		}
	}
	for _, f := range files {
		t.files[f.path] = f
		key := f.path
		for {
			dir := path.Dir(strings.TrimSuffix(key, "/"))
			parent := ""
			if dir != "." {
				parent = dir + "/"
			}
			add(parent, key)
			if parent == "" {
				break
			}
			key = parent
		}
	}
	for _, keys := range t.children {
		slices.SortFunc(keys, func(a, b string) int {
			if da, db := strings.HasSuffix(a, "/"), strings.HasSuffix(b, "/"); da != db {
				if da {
					return -1
				}
				return 1
			}
			// A design's flow follows it.
			sa, fa := stem(strings.ToLower(a))
			sb, fb := stem(strings.ToLower(b))
			if c := strings.Compare(sa, sb); c != 0 || fa == fb {
				return c
			}
			if fa {
				return 1
			}
			return -1
		})
	}
	return t
}

// keyName is the name a tree's row shows for key: a folder's, or a
// file's without its extension.
func keyName(key string) string {
	return strings.TrimSuffix(path.Base(strings.TrimSuffix(key, "/")), ".scenra")
}

// folderOf is the folder key the file or folder key is in or is.
func folderOf(key string) string {
	if key == "" || strings.HasSuffix(key, "/") {
		return key
	}
	if dir := path.Dir(key); dir != "." {
		return dir + "/"
	}
	return ""
}
