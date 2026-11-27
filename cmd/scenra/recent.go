package main

import (
	"encoding/json"
	"os"
	"slices"
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

// keyRecent is the storage key of the folders opened lately.
const keyRecent = "recent"

// recentLimit is how many folders the home page keeps.
const recentLimit = 30

// recentFolder is a folder opened lately, and the file shown there last,
// which its cover draws.
type recentFolder struct {
	Folder string    `json:"folder"`
	File   string    `json:"file,omitempty"`
	Opened time.Time `json:"opened"`
}

// recents are the folders opened lately, the latest first, kept in the
// storage.
type recents struct {
	cx   *core.Scope
	list *core.Signal[[]recentFolder]
}

func newRecents(cx *core.Scope) *recents {
	var list []recentFolder
	if data, ok := cx.Storage().GetItem(keyRecent); ok {
		json.Unmarshal([]byte(data), &list)
	}
	r := &recents{cx: cx, list: core.NewSignalFunc(cx, []recentFolder(nil), slices.Equal[[]recentFolder])}
	// Folders moved or deleted since leave the list.
	kept := slices.DeleteFunc(slices.Clone(list), func(f recentFolder) bool { return !isDir(f.Folder) })
	if len(kept) != len(list) {
		r.set(kept)
	} else {
		r.list.Set(list)
	}
	return r
}

// isDir says a folder is at path.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// set keeps list as the recent folders.
func (r *recents) set(list []recentFolder) {
	if len(list) > recentLimit {
		list = list[:recentLimit]
	}
	r.list.Set(list)
	if data, err := json.Marshal(list); err == nil {
		r.cx.Storage().SetItem(keyRecent, string(data))
	}
}

// touch moves the folder dir first, as it opens.
func (r *recents) touch(dir string) {
	list := slices.Clone(r.list.GetUntracked())
	f := recentFolder{Folder: dir}
	if i := slices.IndexFunc(list, func(f recentFolder) bool { return f.Folder == dir }); i >= 0 {
		f = list[i]
		list = slices.Delete(list, i, i+1)
	}
	f.Opened = r.cx.Now()
	r.set(append([]recentFolder{f}, list...))
}

// cover keeps file, unless "", as the file the folder dir shows last.
func (r *recents) cover(dir, file string) {
	list := slices.Clone(r.list.GetUntracked())
	i := slices.IndexFunc(list, func(f recentFolder) bool { return f.Folder == dir })
	if file == "" || i < 0 || list[i].File == file {
		return
	}
	list[i].File = file
	r.set(list)
}

// remove forgets the folder dir.
func (r *recents) remove(dir string) {
	r.set(slices.DeleteFunc(slices.Clone(r.list.GetUntracked()), func(f recentFolder) bool { return f.Folder == dir }))
}
