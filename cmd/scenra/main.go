// Command scenra is a desktop app designer with an AI assistant: a folder
// of Scenra files, each a design or a flow chart, as a tree on the left,
// the file chosen beside it, and on the right a conversation with a coding
// agent, such as Claude Code or Codex, run through Agio, which edits the
// folder's files as the scenra skill among the app's resources teaches.
// It is scenraview's Window with the app's menu, settings and usage
// statistics.
package main

import (
	"os"
	"path/filepath"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/ui"
	"github.com/oneo-me/nsgo-agio/agent"
	"github.com/oneo-me/nsgo-agio/host"

	"github.com/oneo-me/nsgo-scenra/cmd/scenra/locales"
	"github.com/oneo-me/nsgo-scenra/scenraview"
	words "github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

func main() {
	agent.Client = agent.ClientInfo{Name: "scenra", Title: "Scenra", Version: "1"}
	nsgo.Run(Root, nsgo.WindowSize(1440, 900), nsgo.WindowConstraints(core.WindowConstraints{MinWidth: 1000, MinHeight: 480}))
}

// shown is the editor of the window shown last, which tests reach.
var shown *scenraview.Editor

// Root is the app's window: the name of the file shown in its title bar
// over the folder's files and the file, and the assistant.
func Root(cx *core.Scope) core.Node {
	home, _ := os.UserHomeDir()
	return root(cx, host.Options{Dir: home})
}

// root is Root with the agents agents runs.
func root(cx *core.Scope, agents host.Options) core.Node {
	colors := ui.ColorsOf(cx)
	toasts := ui.NewToaster(cx)
	settingsOpen := core.NewSignal(cx, false)
	stats := newUsage(cx)
	m, srv := startAgents(cx, agents)
	e := scenraview.New(cx, scenraview.Options{
		Agents:       m,
		OpenSettings: func() { settingsOpen.Set(true) },
		Toaster:      toasts,
		Remember:     "editor",
		OnSend:       stats.sent,
	})
	if srv != nil {
		srv.Attach(e)
	}
	shown = e
	stats.start(e)
	// Scenra files or folders opened from the Finder, or named on the
	// command line.
	desktop.OnOpenFiles(cx, func(paths []string) {
		for _, path := range paths {
			if err := e.OpenPath(path); err != nil {
				toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: words.For(cx).OpenFailed(err.Error())})
			}
		}
	})
	// Closing the window saves the changes, and stays open when saving
	// fails.
	if w := desktop.CurrentWindow(cx); w != nil {
		w.OnClosing(cx, func(c *desktop.WindowClosing) {
			if !e.SaveChanges() {
				c.Cancel()
			}
		})
	}

	titleBar := desktop.IsAvailable(cx)
	if titleBar {
		desktop.SetTitleBar(cx, desktop.TitleBarHidden, scenraview.HeaderHeight)
		cx.Effect(func() {
			title := e.Name()
			switch {
			case e.Empty() && e.Folder() != "":
				title = filepath.Base(e.Folder())
			case e.Empty():
				title = locales.For(cx).AppName()
			case e.Edited():
				title += " — " + words.For(cx).Edited()
			}
			desktop.SetTitle(cx, title)
		})
		cx.Effect(func() { desktop.SetApplicationMenu(cx, menu(cx, e, settingsOpen)) })
	} else {
		cx.OnKey("mod+n", e.NewDesign)
		cx.OnKey("mod+shift+n", e.NewFlow)
		cx.OnKey("mod+o", e.OpenFolder)
		cx.OnKey("mod+s", e.Save)
	}

	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
		scenraview.Window{Editor: e, TitleBar: titleBar},
		settingsDialog(cx, settingsOpen, m),
		ui.Toasts{Toaster: toasts, Placement: ui.ToastBottomStart},
	}}
}

// menu is the application menu.
func menu(cx *core.Scope, e *scenraview.Editor, settingsOpen *core.Signal[bool]) []desktop.MenuItem {
	tr, w := locales.For(cx), words.For(cx)
	settings := func() { settingsOpen.Set(true) }
	return []desktop.MenuItem{
		{Label: tr.AppName(), Items: []desktop.MenuItem{
			{Role: desktop.MenuAbout}, {Separator: true},
			{Label: w.Settings(), Shortcut: "mod+,", OnSelect: settings}, {Separator: true},
			{Role: desktop.MenuHide}, {Role: desktop.MenuHideOthers}, {Role: desktop.MenuShowAll}, {Separator: true},
			{Role: desktop.MenuQuit},
		}},
		{Label: tr.FileMenu(), Items: []desktop.MenuItem{
			{Label: w.NewDesign(), Shortcut: "mod+n", Disabled: e.Folder() == "", OnSelect: e.NewDesign},
			{Label: w.NewFlow(), Shortcut: "mod+shift+n", Disabled: e.Folder() == "", OnSelect: e.NewFlow},
			{Label: w.OpenFolder(), Shortcut: "mod+o", OnSelect: e.OpenFolder},
			{Separator: true},
			{Label: w.Save(), Shortcut: "mod+s", OnSelect: e.Save},
			{Separator: true},
			{Label: w.NewChat(), Shortcut: "mod+alt+n", OnSelect: e.NewChat},
			{Separator: true},
			{Role: desktop.MenuClose},
		}},
		{Label: tr.EditMenu(), Items: []desktop.MenuItem{
			{Role: desktop.MenuUndo}, {Role: desktop.MenuRedo}, {Separator: true},
			{Role: desktop.MenuCut}, {Role: desktop.MenuCopy}, {Role: desktop.MenuPaste}, {Role: desktop.MenuSelectAll},
			{Separator: true},
			{Label: tr.UndoChange(), Shortcut: "mod+alt+z", Disabled: !e.CanUndo(), OnSelect: e.Undo},
			{Label: tr.RedoChange(), Shortcut: "mod+alt+shift+z", Disabled: !e.CanRedo(), OnSelect: e.Redo},
		}},
		{Label: tr.ViewMenu(), Items: []desktop.MenuItem{
			{Label: w.ZoomIn(), Shortcut: "mod+=", OnSelect: e.ZoomIn},
			{Label: w.ZoomOut(), Shortcut: "mod+-", OnSelect: e.ZoomOut},
			{Label: w.Fit(), Shortcut: "mod+0", OnSelect: e.Fit},
			{Separator: true},
			{Label: w.LeftToRight(), Checkable: true, Checked: e.LeftToRight(), Disabled: !e.FlowOpen(), OnSelect: e.ToggleDirection},
		}},
	}
}
