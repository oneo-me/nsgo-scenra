package scenraview

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"
	uiflow "github.com/nsgo-dev/nsgo/ui/flow"
	"github.com/oneo-me/nsgo-agio/chatview"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// historyLimit is how many changes undo keeps.
const historyLimit = 200

// Options configure an Editor.
type Options struct {
	// Agents, when set, runs the assistant: a conversation about the open
	// folder, working in it, whose agent edits its files as the app's
	// skill teaches and asks the app's Server what the user sees. Without
	// it the editor has no assistant.
	Agents *chatview.Model
	// OpenSettings, when set, shows where the app sets up the agents,
	// which the assistant's header offers.
	OpenSettings func()
	// Toaster shows what failed to open or save, and what was saved; one
	// of the editor's own when nil, which Toaster returns for the app to
	// show.
	Toaster *ui.Toaster
	// Remember, when set, prefixes the storage keys under which the open
	// folder, the files open in tabs and the one shown open again at the
	// next start.
	Remember string
	// OnSend, when set, hears of each message sent to the assistant.
	OnSend func()
}

// document is a Scenra file open in a tab: its content and path, its
// history and what the user selected in it.
type document struct {
	e *Editor
	// id keys the document's tab.
	id  int
	doc *core.Signal[flowfile.File]
	// path is the file the document came from, which saving writes
	// again; dirty says it changed since.
	path  string
	dirty *core.Signal[bool]
	// selected is the node of a flow the user or the assistant selected,
	// and element the path of a design's selected element.
	selected, element *core.Signal[string]
	// undos and redos are documents before and after changes.
	undos, redos *core.Signal[[]flowfile.File]
	// group is the key of the last change, whose undo later changes with
	// the same key join.
	group string
	// stamp is the version of the file the document was last read from
	// or written to, and broken why the file's latest version does not
	// parse, "" when it does.
	stamp  stamp
	broken string
}

// Editor is a folder of Scenra files, the ones open for editing, a tab
// each, and the one shown, with an assistant that works in the folder.
// The views show an editor; an app showing one file opens just it and
// leaves out the TabBar.
type Editor struct {
	cx   *core.Scope
	opts Options
	// folder is the open folder, "" before one is, and files the Scenra
	// files found in it.
	folder *core.Signal[string]
	files  *core.Signal[[]entry]
	tabs   *core.Signal[[]*document]
	active *core.Signal[*document]
	// blank stands for the document shown while none is open.
	blank *document
	// naming is the kind of file the new file dialog makes, "" while it
	// is closed, and into the folder it goes in, relative to the open one.
	naming *core.Signal[flowfile.Kind]
	into   string
	camera *uiflow.State
	toasts *ui.Toaster
	a      *assistant
	ids    int
	// scanning is the folder the scan reads, shared with its goroutine,
	// and rescan asks it to read the folder now.
	mu       sync.Mutex
	scanning string
	rescan   chan struct{}
}

// New is an editor with no folder open, or with the one opts.Remember
// kept.
func New(cx *core.Scope, opts Options) *Editor {
	toasts := opts.Toaster
	if toasts == nil {
		toasts = ui.NewToaster(cx)
	}
	e := &Editor{cx: cx, opts: opts, folder: core.NewSignal(cx, ""), files: core.NewSignalFunc(cx, []entry(nil), slices.Equal[[]entry]),
		tabs:   core.NewSignalFunc(cx, []*document(nil), func(a, b []*document) bool { return false }),
		active: core.NewSignal[*document](cx, nil), naming: core.NewSignal[flowfile.Kind](cx, ""),
		camera: &uiflow.State{}, toasts: toasts, rescan: make(chan struct{}, 1)}
	e.blank = e.newDocument(flowfile.File{}, "")
	e.a = newAssistant(cx, e)
	e.watch()
	// Another flow shows fitted, once the batch that opened it is over.
	cx.Effect(func() {
		e.cur()
		cx.Post(e.camera.Fit)
	})
	if opts.Remember != "" {
		e.remember(opts.Remember+".folder", opts.Remember+".openFiles", opts.Remember+".activeFile")
	}
	return e
}

// remember opens the folder and the files open in tabs when the app last
// quit, under the storage keys folder, list and active, and keeps them
// there.
func (e *Editor) remember(folder, list, active string) {
	cx := e.cx
	e.restore(folder, list, active)
	cx.Effect(func() {
		dir := e.folder.Get()
		var paths []string
		for _, d := range e.tabs.Get() {
			paths = append(paths, d.path)
		}
		shown := e.cur().path
		core.Untracked(cx, func() bool {
			st := cx.Storage()
			st.SetItem(folder, dir)
			st.SetItem(list, strings.Join(paths, "\n"))
			st.SetItem(active, shown)
			return true
		})
	})
}

// Toaster shows the editor's messages.
func (e *Editor) Toaster() *ui.Toaster { return e.toasts }

// restore opens the folder the storage keeps under folderKey, and its
// files under listKey, leaving out those that are gone, and shows the
// one under activeKey.
func (e *Editor) restore(folderKey, listKey, activeKey string) {
	st := e.cx.Storage()
	dir, _ := st.GetItem(folderKey)
	if info, err := os.Stat(dir); dir == "" || err != nil || !info.IsDir() {
		return
	}
	e.setFolder(dir)
	list, _ := st.GetItem(listKey)
	active, _ := st.GetItem(activeKey)
	var shown *document
	for _, path := range strings.Split(list, "\n") {
		if path == "" || !e.inside(path) {
			continue
		}
		f, err := readFile(path)
		if err != nil {
			continue
		}
		d := e.newDocument(f, path)
		e.tabs.Set(append(e.tabs.GetUntracked(), d))
		if shown == nil || path == active {
			shown = d
		}
	}
	if shown != nil {
		e.active.Set(shown)
	}
}

// readFile reads the Scenra file at path.
func readFile(path string) (flowfile.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return flowfile.File{}, err
	}
	return flowfile.Parse(string(data))
}

// newDocument is a document of f, from the file at path.
func (e *Editor) newDocument(f flowfile.File, path string) *document {
	cx := e.cx
	never := func(a, b flowfile.File) bool { return false }
	stacks := func(a, b []flowfile.File) bool { return false }
	e.ids++
	d := &document{e: e, id: e.ids, doc: core.NewSignalFunc(cx, f, never), path: path, dirty: core.NewSignal(cx, false),
		selected: core.NewSignal(cx, ""), element: core.NewSignal(cx, ""),
		undos: core.NewSignalFunc(cx, []flowfile.File(nil), stacks), redos: core.NewSignalFunc(cx, []flowfile.File(nil), stacks)}
	d.mark()
	return d
}

// cur is the document shown, blank while none is open, which a caller
// reading it follows.
func (e *Editor) cur() *document {
	if d := e.active.Get(); d != nil {
		return d
	}
	return e.blank
}

// shown is the document shown, for actions.
func (e *Editor) shown() *document { return core.Untracked(e.cx, e.cur) }

// empty says no document is open.
func (e *Editor) empty() bool { return len(e.tabs.Get()) == 0 }

// activate shows d's tab.
func (e *Editor) activate(d *document) {
	if e.active.GetUntracked() == d {
		return
	}
	e.active.Set(d)
	e.cx.Post(e.camera.Fit)
}

// inside says path is in the open folder.
func (e *Editor) inside(path string) bool {
	dir := e.folder.GetUntracked()
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// rel is path relative to the open folder, with slashes.
func (e *Editor) rel(path string) string {
	rel, err := filepath.Rel(e.folder.GetUntracked(), path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// setFolder opens the folder at dir, whose files the scan finds.
func (e *Editor) setFolder(dir string) {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	e.mu.Lock()
	e.scanning = dir
	e.mu.Unlock()
	e.cx.Batch(func() {
		e.folder.Set(dir)
		e.files.Set(nil)
	})
	e.scanNow()
}

// openFolder opens the folder at dir in place of the open one, once the
// changes of the files open are saved or discarded, and runs then, when
// set, after; the open folder stays as it is.
func (e *Editor) openFolder(dir string, then func()) {
	if abs, err := filepath.Abs(dir); err == nil && abs == e.folder.GetUntracked() {
		if then != nil {
			then()
		}
		return
	}
	e.KeepAllChanges(func() {
		e.a.remove()
		e.cx.Batch(func() {
			e.tabs.Set(nil)
			e.active.Set(nil)
			e.setFolder(dir)
		})
		if then != nil {
			then()
		}
	})
}

// OpenFolder asks for a folder and opens it.
func (e *Editor) OpenFolder() {
	e.cx.PickDirectories(core.PickDirectoriesOptions{}, func(dirs []core.Directory, err error) {
		switch {
		case err != nil:
			e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: locales.For(e.cx).OpenFailed(err.Error())})
		case len(dirs) > 0 && dirs[0].Path != "":
			e.openFolder(dirs[0].Path, nil)
		}
	})
}

// OpenPath opens the folder at path, or the Scenra file at path in a tab,
// opening the folder holding it first when it is not in the open one.
func (e *Editor) OpenPath(path string) error {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		e.openFolder(path, nil)
		return nil
	}
	f, err := readFile(path)
	if err != nil {
		return err
	}
	if e.inside(path) {
		e.openDoc(f, path)
		return nil
	}
	e.openFolder(filepath.Dir(path), func() { e.openDoc(f, path) })
	return nil
}

// open shows the file at path, which the sidebar chose, in a tab.
func (e *Editor) open(path string) {
	for _, d := range e.tabs.GetUntracked() {
		if d.path == path {
			e.activate(d)
			return
		}
	}
	f, err := readFile(path)
	if err != nil {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: locales.For(e.cx).OpenFailed(err.Error())})
		return
	}
	e.openDoc(f, path)
}

// openDoc opens f from the file at path in a tab: the file's tab when it
// is open already, else a new one.
func (e *Editor) openDoc(f flowfile.File, path string) {
	for _, d := range e.tabs.GetUntracked() {
		if d.path == path {
			e.activate(d)
			return
		}
	}
	d := e.newDocument(f, path)
	e.cx.Batch(func() {
		e.tabs.Set(append(slices.Clone(e.tabs.GetUntracked()), d))
		e.activate(d)
	})
}

// NewDesign and NewFlow ask for the name of a new design or flow, which
// they make in the folder the sidebar has selected and open.
func (e *Editor) NewDesign() { e.askName(flowfile.Design) }
func (e *Editor) NewFlow()   { e.askName(flowfile.Chart) }

func (e *Editor) askName(kind flowfile.Kind) {
	if e.folder.GetUntracked() == "" {
		return
	}
	e.naming.Set(kind)
}

// errExists and errName are the errors of a new file whose name is taken
// or cannot name a file.
var (
	errExists = errors.New("a file of that name exists")
	errName   = errors.New("not a file name")
)

// create makes a file of kind named name in the folder into, relative to
// the open one, and opens it.
func (e *Editor) create(kind flowfile.Kind, into, name string) error {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".scenra")
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return errName
	}
	path := filepath.Join(e.folder.GetUntracked(), filepath.FromSlash(into), name+".scenra")
	f := flowfile.File{Kind: kind}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return errExists
	}
	if err != nil {
		return err
	}
	_, err = file.WriteString(f.Format())
	if cerr := file.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	e.found(entry{path: e.rel(path), kind: kind})
	e.openDoc(f, path)
	return nil
}

// name is the document's file name.
func (d *document) name() string {
	if d.path == "" {
		return ""
	}
	return filepath.Base(d.path)
}

// at is what the user selected.
func (d *document) at() location {
	return location{node: d.selected.GetUntracked(), element: d.element.GetUntracked()}
}

// show selects the node of l, and the element, dropping what is gone.
func (d *document) show(l location) {
	f := d.doc.GetUntracked()
	if _, ok := f.Flow.Node(l.node); !ok || f.Kind != flowfile.Chart {
		l.node = ""
	}
	if _, ok := flowfile.ElementAt(f.UI, l.element); !ok || f.Kind != flowfile.Design {
		l.element = ""
	}
	d.e.cx.Batch(func() {
		d.selected.Set(l.node)
		d.element.Set(l.element)
	})
	if l.node != "" && d.e.shown() == d {
		d.e.cx.Post(func() { d.e.camera.Reveal(l.node) })
	}
}

// change replaces the document with an edit of it, which undo reverts.
// Consecutive changes of one nonempty group, such as an assistant's
// turn, undo together.
func (d *document) change(f flowfile.File, group string) {
	before := d.doc.GetUntracked()
	joined := group != "" && group == d.group && len(d.undos.GetUntracked()) > 0
	d.group = group
	d.e.cx.Batch(func() {
		if !joined {
			undos := append(d.undos.GetUntracked(), before)
			if len(undos) > historyLimit {
				undos = undos[len(undos)-historyLimit:]
			}
			d.undos.Set(undos)
		}
		d.redos.Set(nil)
		d.doc.Set(f)
		d.dirty.Set(true)
		d.show(d.at())
	})
}

// setComment sets the comment on the element at path of the design, or
// removes it when text is "", as a change undo reverts.
func (d *document) setComment(path, text string) {
	if d.e.locked() {
		return
	}
	f := d.doc.GetUntracked()
	els, err := flowfile.SetComment(f.UI, path, text)
	if err != nil {
		return
	}
	f.UI = els
	d.change(f, "")
}

func (d *document) undo() { d.step(d.undos, d.redos) }
func (d *document) redo() { d.step(d.redos, d.undos) }

// step moves the document from one history stack to the other.
func (d *document) step(from, to *core.Signal[[]flowfile.File]) {
	list := from.GetUntracked()
	if len(list) == 0 || d.e.locked() {
		return
	}
	d.group = ""
	d.e.cx.Batch(func() {
		to.Set(append(to.GetUntracked(), d.doc.GetUntracked()))
		from.Set(list[: len(list)-1 : len(list)-1])
		d.doc.Set(list[len(list)-1])
		d.dirty.Set(true)
		d.show(d.at())
	})
}

// Undo and Redo step through the history of the document shown.
func (e *Editor) Undo() { e.shown().undo() }
func (e *Editor) Redo() { e.shown().redo() }

// closeTab closes d's tab once its changes are saved or discarded.
func (e *Editor) closeTab(d *document) {
	e.keepChanges(d, func() {
		tabs := slices.Clone(e.tabs.GetUntracked())
		i := slices.Index(tabs, d)
		if i < 0 {
			return
		}
		tabs = slices.Delete(tabs, i, i+1)
		e.cx.Batch(func() {
			e.tabs.Set(tabs)
			switch {
			case len(tabs) == 0:
				e.active.Set(nil)
			case e.active.GetUntracked() == d:
				e.activate(tabs[min(i, len(tabs)-1)])
			}
		})
	})
}

// CloseShown closes the tab shown.
func (e *Editor) CloseShown() {
	if !e.empty() {
		e.closeTab(e.shown())
	}
}

// keepChanges asks what becomes of d's unsaved changes before it goes,
// showing its tab, and runs then once they are saved or discarded;
// canceling, or a save that fails, keeps the document.
func (e *Editor) keepChanges(d *document, then func()) {
	if !d.dirty.GetUntracked() {
		then()
		return
	}
	e.activate(d)
	tr := locales.For(e.cx)
	save, discard := tr.Save(), tr.DontSave()
	base.ShowAlertDialog(e.cx, base.AlertDialog{
		Title: tr.SaveChanges(d.name()), Description: tr.ChangesLost(),
		Actions: []core.AlertAction{{Label: save}, {Label: discard, Role: core.AlertDestructive}, {Label: tr.Cancel(), Role: core.AlertCancel}},
	}, func(label string) {
		switch label {
		case save:
			if e.save(d) {
				then()
			}
		case discard:
			then()
		}
	})
}

// KeepAllChanges asks about each document's unsaved changes in turn, and
// runs then once every one is saved or discarded.
func (e *Editor) KeepAllChanges(then func()) { e.keepChangesFrom(0, then) }

func (e *Editor) keepChangesFrom(i int, then func()) {
	tabs := e.tabs.GetUntracked()
	for ; i < len(tabs); i++ {
		if d := tabs[i]; d.dirty.GetUntracked() {
			e.keepChanges(d, func() { e.keepChangesFrom(i+1, then) })
			return
		}
	}
	then()
}

// Save writes the document shown to its file.
func (e *Editor) Save() {
	if !e.empty() && e.save(e.shown()) {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: locales.For(e.cx).Saved()})
	}
}

// save writes d to its file, and says it did.
func (e *Editor) save(d *document) bool {
	if err := writeFile(d.path, d.doc.GetUntracked().Format()); err != nil {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: locales.For(e.cx).SaveFailed(err.Error())})
		return false
	}
	d.mark()
	d.dirty.Set(false)
	return true
}

// writeFile replaces the file at path with text, leaving the old one
// whole if writing fails.
func writeFile(path, text string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".scenra-*")
	if err != nil {
		return err
	}
	_, err = tmp.WriteString(text)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}

// The editor's state, which a caller reading it follows, such as to name
// the window or turn menu items on and off.

// Folder is the open folder, "" before one is.
func (e *Editor) Folder() string { return e.folder.Get() }

// Empty says no file is open.
func (e *Editor) Empty() bool { return e.empty() }

// Name is the file name of the document shown, "" while none is.
func (e *Editor) Name() string { return e.cur().name() }

// Edited says the document shown changed since it was last saved.
func (e *Editor) Edited() bool { return e.cur().dirty.Get() }

// Unsaved says some open document changed since it was last saved.
func (e *Editor) Unsaved() bool {
	return slices.ContainsFunc(e.tabs.Get(), func(d *document) bool { return d.dirty.Get() })
}

// File is the document shown.
func (e *Editor) File() flowfile.File { return e.cur().doc.Get() }

// CanUndo and CanRedo say the document shown has a change to undo or to
// redo, which it does not while the agent works.
func (e *Editor) CanUndo() bool { return len(e.cur().undos.Get()) > 0 && !e.Working() }
func (e *Editor) CanRedo() bool { return len(e.cur().redos.Get()) > 0 && !e.Working() }

// FlowOpen says the document shown is a flow, and LeftToRight that it
// points right.
func (e *Editor) FlowOpen() bool { return e.cur().doc.Get().Kind == flowfile.Chart }
func (e *Editor) LeftToRight() bool {
	return e.cur().doc.Get().Flow.Direction == flowfile.Right
}

// ZoomIn, ZoomOut and Fit move the flow chart's camera.
func (e *Editor) ZoomIn()  { e.camera.ZoomIn() }
func (e *Editor) ZoomOut() { e.camera.ZoomOut() }
func (e *Editor) Fit()     { e.camera.Fit() }

// NewChat starts a new conversation about the folder.
func (e *Editor) NewChat() { e.a.remove() }
