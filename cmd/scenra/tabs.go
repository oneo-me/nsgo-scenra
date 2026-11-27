package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/scenraview"
	words "github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// The storage keys of the tabs open at quit, and of the one shown, by its
// folder.
const (
	keyTabs  = "tabs"
	keyShown = "tabs.shown"
)

// tab is a folder open beside the home page, in an editor of its own
// whose scope lasts until the tab closes, so each folder keeps its
// document, history and conversations.
type tab struct {
	folder string
	cx     *core.Scope
	e      *scenraview.Editor
}

// savedTab is what opens a tab again: its folder and the file shown.
type savedTab struct {
	Folder string `json:"folder"`
	File   string `json:"file,omitempty"`
}

// tabs are the window's tabs after the home page, which is always first.
type tabs struct {
	cx     *core.Scope
	toasts *ui.Toaster
	open   *core.Signal[[]*tab]
	// shown is the tab shown, nil for the home page.
	shown *core.Signal[*tab]
	// editor is an editor of the folder, made in the tab's scope.
	editor func(cx *core.Scope, folder string) *scenraview.Editor
	recent *recents
}

func newTabs(cx *core.Scope, toasts *ui.Toaster, recent *recents, editor func(cx *core.Scope, folder string) *scenraview.Editor) *tabs {
	t := &tabs{cx: cx, toasts: toasts, recent: recent, editor: editor,
		open: core.NewSignalFunc(cx, []*tab(nil), slices.Equal[[]*tab]), shown: core.NewSignal[*tab](cx, nil)}
	t.restore()
	// The tabs and their files open again at the next start.
	cx.Effect(func() {
		var saved []savedTab
		for _, tb := range t.open.Get() {
			saved = append(saved, savedTab{Folder: tb.folder, File: tb.e.Path()})
		}
		shown := ""
		if tb := t.shown.Get(); tb != nil {
			shown = tb.folder
		}
		core.Untracked(cx, func() bool {
			st := cx.Storage()
			data, _ := json.Marshal(saved)
			st.SetItem(keyTabs, string(data))
			st.SetItem(keyShown, shown)
			return true
		})
	})
	return t
}

// restore opens the tabs kept at the last quit whose folders are still
// there, and shows the one shown then.
func (t *tabs) restore() {
	st := t.cx.Storage()
	data, _ := st.GetItem(keyTabs)
	var saved []savedTab
	json.Unmarshal([]byte(data), &saved)
	shown, _ := st.GetItem(keyShown)
	for _, s := range saved {
		if info, err := os.Stat(s.Folder); err != nil || !info.IsDir() {
			continue
		}
		tb := t.add(s.Folder)
		if s.File != "" && tb.e.Contains(s.File) {
			tb.e.OpenPath(s.File)
		}
		if s.Folder == shown {
			t.shown.Set(tb)
		}
	}
}

// editorShown is the editor of the tab shown, nil on the home page, which
// a caller reading it follows.
func (t *tabs) editorShown() *scenraview.Editor {
	if tb := t.shown.Get(); tb != nil {
		return tb.e
	}
	return nil
}

// editors are the editors of the open tabs, for actions.
func (t *tabs) editors() []*scenraview.Editor {
	var out []*scenraview.Editor
	for _, tb := range t.open.GetUntracked() {
		out = append(out, tb.e)
	}
	return out
}

// add opens a tab of the folder dir after the others, without showing
// it.
func (t *tabs) add(dir string) *tab {
	cx := t.cx.Child()
	tb := &tab{folder: dir, cx: cx, e: t.editor(cx, dir)}
	t.open.Set(append(slices.Clone(t.open.GetUntracked()), tb))
	t.recent.touch(dir)
	// The file shown is the folder's cover among the recent ones.
	cx.Effect(func() {
		path := tb.e.Path()
		cx.Untrack(func() { t.recent.cover(dir, path) })
	})
	return tb
}

// openFolder shows the tab of the folder dir, opening one first unless
// there is.
func (t *tabs) openFolder(dir string) *tab {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		if err == nil {
			err = os.ErrNotExist
		}
		// A recent folder gone since the list was read leaves it.
		t.recent.remove(dir)
		t.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: words.For(t.cx).OpenFailed(err.Error())})
		return nil
	}
	i := slices.IndexFunc(t.open.GetUntracked(), func(tb *tab) bool { return tb.folder == dir })
	var tb *tab
	t.cx.Batch(func() {
		if i >= 0 {
			tb = t.open.GetUntracked()[i]
			t.recent.touch(dir)
		} else {
			tb = t.add(dir)
		}
		t.shown.Set(tb)
	})
	return tb
}

// openPath shows the folder at path in its tab, or the Scenra file at
// path in the tab whose project holds it, else in a new tab of its
// project.
func (t *tabs) openPath(path string) error {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		t.openFolder(path)
		return nil
	}
	i := slices.IndexFunc(t.open.GetUntracked(), func(tb *tab) bool { return core.Untracked(t.cx, func() bool { return tb.e.Contains(path) }) })
	var tb *tab
	if i >= 0 {
		tb = t.open.GetUntracked()[i]
		t.shown.Set(tb)
	} else if tb = t.openFolder(scenraview.ProjectOf(path)); tb == nil {
		return nil
	}
	return tb.e.OpenPath(path)
}

// pick asks for a folder and shows its tab.
func (t *tabs) pick() {
	t.cx.PickDirectories(core.PickDirectoriesOptions{}, func(dirs []core.Directory, err error) {
		switch {
		case err != nil:
			t.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: words.For(t.cx).OpenFailed(err.Error())})
		case len(dirs) > 0 && dirs[0].Path != "":
			t.openFolder(dirs[0].Path)
		}
	})
}

// close closes tb once its changes are saved, showing the tab after it,
// else the one before or the home page, and stops its folder's agents;
// it stays open when saving fails.
func (t *tabs) close(tb *tab) {
	if !tb.e.SaveChanges() {
		return
	}
	open := t.open.GetUntracked()
	i := slices.Index(open, tb)
	if i < 0 {
		return
	}
	rest := slices.Delete(slices.Clone(open), i, i+1)
	t.cx.Batch(func() {
		t.open.Set(rest)
		if t.shown.GetUntracked() == tb {
			switch {
			case i < len(rest):
				t.shown.Set(rest[i])
			case len(rest) > 0:
				t.shown.Set(rest[len(rest)-1])
			default:
				t.shown.Set(nil)
			}
		}
	})
	// Its views go first, so none reads the editor once it is gone.
	t.cx.Post(tb.cx.Dispose)
}

// closeShown closes the tab shown, and the window on the home page.
func (t *tabs) closeShown() {
	if tb := t.shown.GetUntracked(); tb != nil {
		t.close(tb)
		return
	}
	t.cx.CloseWindow()
}

// saveAll saves the changes of every tab, and says none are left.
func (t *tabs) saveAll() bool {
	ok := true
	for _, e := range t.editors() {
		ok = e.SaveChanges() && ok
	}
	return ok
}
