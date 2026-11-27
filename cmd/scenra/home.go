package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/cmd/scenra/locales"
	"github.com/oneo-me/nsgo-scenra/scenraview"
	words "github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// Sizes of a recent folder's card and of its cover.
const (
	cardWidth   core.Points = 248
	coverHeight core.Points = 156
)

// homePage is the tab always first: the folders opened lately as cards,
// the latest first, each opening in a tab of its own, with a way to open
// another.
func homePage(cx *core.Scope, t *tabs) core.Node {
	tr, w := locales.For(cx), words.For(cx)
	colors := ui.ColorsOf(cx)
	empty := core.View{TestID: "recent-empty", Style: core.Style{AlignItems: core.AlignCenter, Gap: 12, Padding: core.Symmetric(64, 0)}, Children: []core.Node{
		ui.Icon(lucide.FolderOpen, 32, colors.Muted),
		core.Text{Content: tr.RecentEmpty(), Style: core.Style{FontSize: 13, LineHeight: 19, TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(360)},
			StyleFunc: ui.Colored(colors.Muted)},
	}}
	cards := core.View{TestID: "recent", Style: core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 20}, Children: []core.Node{
		core.Each(t.recent.list.Get, func(f recentFolder) string { return f.Folder }, func(cx *core.Scope, item func() recentFolder, _ func() int) core.Node {
			return recentCard(cx, t, item)
		}),
	}}
	// A folder or a Scenra file dropped on the page opens as the Finder's
	// do, tinting the page while one is dragged over it.
	dragging := core.NewSignal(cx, false)
	drop := func(uris []string) {
		dragging.Set(false)
		for _, uri := range uris {
			path, err := fileURLPath(uri)
			if err == nil {
				err = t.openPath(path)
			}
			if err != nil {
				t.toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: w.OpenFailed(err.Error())})
			}
		}
	}
	return core.View{TestID: "home-drop", Style: ui.FillRest, OnFileDrop: drop,
		OnFileDragEnter: func() { dragging.Set(true) }, OnFileDragLeave: func() { dragging.Set(false) },
		StyleFunc: func() core.Style {
			if dragging.Get() {
				return core.Style{Background: ui.Alpha(colors.Accent(), .06)}
			}
			return core.Style{}
		},
		Children: []core.Node{ui.ScrollView{TestID: "home", Style: ui.FillRest, Children: []core.Node{
			core.View{Style: core.Style{Padding: core.All(32), Gap: 20}, Children: []core.Node{
				core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}, Children: []core.Node{
					ui.Text(tr.Recent(), 20, ui.Semibold, colors.Text),
					ui.Spacer{},
					ui.Button{Text: w.OpenFolder(), Icon: lucide.FolderOpen, Variant: ui.ButtonOutline, OnPress: t.pick, TestID: "home-open-folder"},
				}},
				core.Show{When: func() bool { return len(t.recent.list.Get()) == 0 },
					Then: func(*core.Scope) core.Node { return empty },
					Else: func(*core.Scope) core.Node { return cards }},
			}},
		}}}}
}

// fileURLPath is the local path a file: URL dropped from another app
// names, which may be a folder, unlike what core.FileFromURL opens.
func fileURLPath(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("%s is not a local file", raw)
	}
	path := u.Path
	if runtime.GOOS == "windows" {
		path = strings.TrimPrefix(path, "/")
		if u.Host != "" && u.Host != "localhost" {
			path = "//" + u.Host + "/" + path
		}
	}
	return filepath.FromSlash(path), nil
}

// recentCard is a folder opened lately: its cover, a design of the file
// it showed last, over its name and where it is, which opens it in its
// tab, and from its context menu leaves the list.
func recentCard(cx *core.Scope, t *tabs, item func() recentFolder) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	dir := item().Folder
	file := core.NewMemo(cx, func() string { return item().File })
	cover := core.Dynamic(func(cx *core.Scope) core.Node {
		return scenraview.Cover{Folder: dir, File: file.Get(), TestID: "recent-cover",
			Style: core.Style{Height: core.Pt(coverHeight)}}
	})
	name := core.Text{Content: filepath.Base(dir), MaxLines: 1, Style: ui.TypeBody.Merge(core.Style{FontWeight: ui.Medium}), StyleFunc: ui.Colored(colors.Text)}
	where := core.Text{Content: homeRelative(filepath.Dir(dir)), MaxLines: 1, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)}
	face := ui.Press{OnPress: func() { t.openFolder(dir) }, Label: filepath.Base(dir), TestID: "recent-" + filepath.Base(dir), Scale: 0.99,
		Style: core.Style{Width: core.Pt(cardWidth), CornerRadius: 10, Borders: core.All(1), Overflow: core.Hidden},
		StyleFunc: func() core.Style {
			return core.Style{Background: colors.Surface(), BorderColor: colors.Line(),
				Hover: &core.Style{BorderColor: colors.Accent()}}
		},
		Children: []core.Node{
			core.View{StyleFunc: func() core.Style {
				return core.Style{Background: colors.Inset(), BorderColors: core.EdgeColors{Bottom: colors.Line()}}
			}, Children: []core.Node{cover}},
			core.View{Style: core.Style{Padding: core.Symmetric(10, 12), Gap: 2}, Children: []core.Node{name, where}},
		}}
	return ui.ContextMenu{Trigger: face, Children: []core.Node{
		ui.MenuItem{Text: tr.RemoveRecent(), Icon: lucide.Trash, OnSelect: func() { t.recent.remove(dir) }, TestID: "recent-remove"},
	}}
}

// homeRelative is path with the home folder written ~.
func homeRelative(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if rest, ok := strings.CutPrefix(path, home); ok && (rest == "" || strings.HasPrefix(rest, string(filepath.Separator))) {
		return "~" + rest
	}
	return path
}
