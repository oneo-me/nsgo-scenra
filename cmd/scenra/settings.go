package main

import (
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-agio/chatview"
	"github.com/oneo-me/nsgo-scenra/cmd/scenra/locales"
)

// statsSwitch turns the usage statistics on or off.
func statsSwitch(cx *core.Scope) core.Node {
	on := core.NewSignal(cx, statsOn(cx))
	return ui.Switch{Checked: on.Get, OnChange: func(v bool) { on.Set(v); setStatsOn(cx, v) },
		Label: locales.For(cx).StatsSend()}
}

// settingsDialog turns the agents and their models on or off, and the
// usage statistics.
func settingsDialog(cx *core.Scope, open *core.Signal[bool], m *chatview.Model) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return ui.Dialog{Open: open.Get, OnOpenChange: func(o bool, _ base.Reason) { open.Set(o) }, TestID: "settings",
		Title: tr.SettingsTitle(), Style: core.Style{Width: core.Pt(520)},
		Children: []core.Node{
			chatview.Agents{Model: m},
			core.View{Style: core.Style{Gap: 4}, Children: []core.Node{
				ui.Text(tr.StatsSection(), 15, ui.Semibold, colors.Text),
				core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}, Children: []core.Node{
					core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 2}, Children: []core.Node{
						ui.Text(tr.StatsSend(), 13, ui.Medium, colors.Text),
						core.Text{Content: tr.StatsDetail(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
					}},
					statsSwitch(cx),
				}},
			}},
		},
		Footer: []core.Node{ui.Button{Text: tr.Done(), TestID: "settings-done", OnPress: func() { open.Set(false) }}}}
}
