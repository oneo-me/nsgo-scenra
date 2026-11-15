package scenraview

import (
	"os"
	"path/filepath"
	"time"

	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// watchEvery is how often the editor looks whether the open files
// changed, as agents write them.
const watchEvery = 250 * time.Millisecond

// stamp tells a file's versions apart.
type stamp struct {
	size int64
	mod  time.Time
}

func stampOf(path string) stamp {
	info, err := os.Stat(path)
	if err != nil {
		return stamp{}
	}
	return stamp{info.Size(), info.ModTime()}
}

// scanEvery is how many times the editor looks at the open files for
// each time it reads the open folder for its Scenra files.
const scanEvery = 4

// watch reads the open files again as they change, and the open folder
// for its Scenra files, off the UI's goroutine, until the editor's scope
// ends.
func (e *Editor) watch() {
	done := make(chan struct{})
	e.cx.Cleanup(func() { close(done) })
	go func() {
		tick := time.NewTicker(watchEvery)
		defer tick.Stop()
		var cache map[string]scanned
		ticks, last := 0, ""
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				e.cx.Post(func() { e.reload(e.shown()) })
				if ticks++; ticks%scanEvery != 0 {
					continue
				}
			case <-e.rescan:
			}
			e.mu.Lock()
			dir := e.scanning
			e.mu.Unlock()
			if dir == "" {
				continue
			}
			var files []entry
			files, cache = scan(dir, cache)
			first := dir != last
			last = dir
			e.cx.Post(func() { e.scannedFiles(dir, files, first) })
		}
	}()
}

// mark notes the version of d's file the document holds, after reading
// or writing it.
func (d *document) mark() {
	if d.path != "" {
		d.stamp = stampOf(d.path)
	}
}

// reload reads d's file again when it changed since it was last read or
// written, as an agent writing it does. The user's unsaved changes keep
// the document as it is. A file that does not parse leaves the document
// alone and tells of the error once, which the agent hears through the
// server too.
func (e *Editor) reload(d *document) {
	path := d.path
	if path == "" || d.dirty.GetUntracked() {
		return
	}
	now := stampOf(path)
	if now == d.stamp || now == (stamp{}) {
		return
	}
	d.stamp = now
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	f, err := flowfile.Parse(string(data))
	if err != nil {
		if d.broken != err.Error() {
			d.broken = err.Error()
			e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: locales.For(e.cx).OpenFailed(err.Error())})
		}
		return
	}
	d.broken = ""
	if f.Format() == d.doc.GetUntracked().Format() {
		return
	}
	d.change(f, e.a.group())
	d.dirty.Set(false)
}

// saveQuietly writes the document shown to its file when the user
// changed it, without telling of it, so what an agent reads is what the
// user sees.
func (e *Editor) saveQuietly() { e.SaveChanges() }

// locked says the user cannot change the documents now, as the agent is
// at work in the folder.
func (e *Editor) locked() bool { return e.a.busy() }

// Working says the agent is at work in the folder, while which the user
// cannot change its files; a caller reading it follows.
func (e *Editor) Working() bool { return e.a.working() }

// viewOf is what the Server tells an agent: what the user sees of the
// open folder, and the problems of its files.
func (e *Editor) viewOf() string {
	dir := e.folder.GetUntracked()
	if dir == "" {
		return "No folder is open in Scenra.\n"
	}
	var files []fileView
	// What the agent just wrote counts.
	e.reload(e.shown())
	found, _ := scan(dir, nil)
	e.scannedFiles(dir, found, false)
	shown := e.shown()
	for _, en := range found {
		path := filepath.Join(dir, filepath.FromSlash(en.path))
		fv := fileView{path: en.path}
		if shown.path == path {
			fv.file, fv.broken = shown.doc.GetUntracked(), shown.broken
		} else if f, err := readFile(path); err != nil {
			fv.broken = err.Error()
		} else {
			fv.file = f
		}
		if shown.path == path {
			at := shown.at()
			fv.shown = &at
		}
		files = append(files, fv)
	}
	return view(dir, files)
}
