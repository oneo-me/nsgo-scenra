// Command scenra is a desktop app designer with an AI assistant: flow
// files open in tabs, each listing its screens' designs and flow charts on
// the left, and on the right a conversation with an OpenAI-compatible
// model that draws them with tools. It is scenraview's Window with the
// app's menu, settings and usage statistics.
package main

import (
	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/agent"
	"github.com/oneo-me/nsgo-scenra/cmd/scenra/locales"
	"github.com/oneo-me/nsgo-scenra/scenraview"
	words "github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

func main() {
	nsgo.Run(Root, nsgo.WindowSize(1440, 900), nsgo.WindowConstraints(core.WindowConstraints{MinWidth: 1000, MinHeight: 480}))
}

// shown is the editor of the window shown last, which tests reach.
var shown *scenraview.Editor

// Root is the app's window: the open documents' tabs in its title bar
// over the entries of the one shown, the entry open and the assistant.
func Root(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	toasts := ui.NewToaster(cx)
	settingsOpen := core.NewSignal(cx, false)
	stats := newUsage(cx)
	e := scenraview.New(cx, scenraview.Options{
		Provider:     func() agent.Provider { return provider(cx) },
		OpenSettings: func() { settingsOpen.Set(true) },
		Toaster:      toasts,
		Remember:     "editor",
		OnSend:       stats.sent,
	})
	shown = e
	stats.start(e)
	// Flow files opened from the Finder, or named on the command line.
	desktop.OnOpenFiles(cx, func(paths []string) {
		for _, path := range paths {
			if err := e.OpenPath(path); err != nil {
				toasts.Show(ui.ToastContent{Type: ui.ToastError, Title: words.For(cx).OpenFailed(err.Error())})
			}
		}
	})
	// Closing the window with unsaved changes asks first.
	if w := desktop.CurrentWindow(cx); w != nil {
		w.OnClosing(cx, func(c *desktop.WindowClosing) {
			if core.Untracked(cx, e.Unsaved) {
				c.Cancel()
				e.KeepAllChanges(w.Close)
			}
		})
	}

	titleBar := desktop.IsAvailable(cx)
	if titleBar {
		desktop.SetTitleBar(cx, desktop.TitleBarHidden, scenraview.HeaderHeight)
		cx.Effect(func() {
			title := e.Name()
			if e.Empty() {
				title = locales.For(cx).AppName()
			} else if e.Edited() {
				title += " — " + words.For(cx).Edited()
			}
			desktop.SetTitle(cx, title)
		})
		cx.Effect(func() { desktop.SetApplicationMenu(cx, menu(cx, e, settingsOpen)) })
	} else {
		cx.OnKey("mod+n", e.NewFile)
		cx.OnKey("mod+o", e.Open)
		cx.OnKey("mod+s", e.Save)
		cx.OnKey("mod+shift+s", e.SaveAs)
		cx.OnKey("mod+w", e.CloseShown)
	}

	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{
		scenraview.Window{Editor: e, TitleBar: titleBar},
		settingsDialog(cx, settingsOpen),
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
			{Label: w.NewFlow(), Shortcut: "mod+n", OnSelect: e.NewFile},
			{Label: w.Open(), Shortcut: "mod+o", OnSelect: e.Open},
			{Separator: true},
			{Label: w.Save(), Shortcut: "mod+s", OnSelect: e.Save},
			{Label: w.SaveAs(), Shortcut: "mod+shift+s", OnSelect: e.SaveAs},
			{Separator: true},
			{Label: w.NewChat(), Shortcut: "mod+shift+n", OnSelect: e.NewChat},
			{Separator: true},
			{Label: w.CloseTab(), Shortcut: "mod+w", OnSelect: e.CloseShown},
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
