package scenraview

import (
	"fmt"
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

// watch reads the open files again as they change, until the editor's
// scope ends.
func (e *Editor) watch() {
	done := make(chan struct{})
	e.cx.Cleanup(func() { close(done) })
	go func() {
		tick := time.NewTicker(watchEvery)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				e.cx.Post(func() {
					for _, d := range e.tabs.GetUntracked() {
						e.reload(d)
					}
				})
			}
		}
	}()
}

// mark notes the version of d's file the document holds, after reading
// or writing it.
func (d *document) mark() {
	if path := d.path.GetUntracked(); path != "" {
		d.stamp = stampOf(path)
	}
}

// reload reads d's file again when it changed since it was last read or
// written, as an agent writing it does. The user's unsaved changes keep
// the document as it is. A file that does not parse leaves the document
// alone and tells of the error once, which the agent hears through the
// server too.
func (e *Editor) reload(d *document) {
	path := d.path.GetUntracked()
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
	p, err := flowfile.ParseProject(string(data))
	if err != nil {
		if d.broken != err.Error() {
			d.broken = err.Error()
			e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: locales.For(e.cx).OpenFailed(err.Error())})
		}
		return
	}
	d.broken = ""
	if p.Format() == d.doc.GetUntracked().Format() {
		return
	}
	d.change(p, e.a.group(d))
	d.dirty.Set(false)
}

// saveQuietly writes d to its file when the user changed it, without
// telling of it, so what an agent reads is what the user sees.
func (e *Editor) saveQuietly(d *document) {
	path := d.path.GetUntracked()
	if path == "" || !d.dirty.GetUntracked() {
		return
	}
	if err := writeFile(path, d.doc.GetUntracked().Format()); err != nil {
		e.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: locales.For(e.cx).SaveFailed(err.Error())})
		return
	}
	d.mark()
	d.dirty.Set(false)
}

// locked says the user cannot change d now, as its agent is at work on
// it.
func (e *Editor) locked(d *document) bool { return e.a.busy(d) }

// Working says the agent is at work on the document shown, which the
// user cannot change meanwhile; a caller reading it follows.
func (e *Editor) Working() bool { return e.a.working(e.cur()) }

// viewOf is what the Server tells an agent of the file at path: what the
// user sees of it and its problems.
func (e *Editor) viewOf(path string) (string, bool) {
	if !filepath.IsAbs(path) {
		return fmt.Sprintf("error: %q is not an absolute path", path), true
	}
	path = filepath.Clean(path)
	for _, d := range e.tabs.GetUntracked() {
		if d.path.GetUntracked() != path {
			continue
		}
		// What the agent just wrote counts.
		e.reload(d)
		text := view(d.doc.GetUntracked(), d.at())
		if d.broken != "" {
			text = "The file does not parse, so the editor still shows its last version: " + d.broken + "\n" + text
		}
		return text, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "error: " + err.Error(), true
	}
	p, err := flowfile.ParseProject(string(data))
	if err != nil {
		return "The file is not open in the editor, and does not parse: " + err.Error() + "\n", false
	}
	return "The file is not open in the editor.\n" + view(p, location{}), false
}
