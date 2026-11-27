package main

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
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

// settingsDialog is the settings over the window, whichever tab shows:
// the language and theme, the agents and their models, the model titling
// conversations and the usage statistics. open and category live outside
// the views a new language builds again, so the dialog stays on its page.
func settingsDialog(cx *core.Scope, m *chatview.Model, n *namer, open *core.Signal[bool], category *core.Signal[string]) core.Node {
	return chatview.SettingsDialog{Model: m, Open: open, Category: category,
		Settings: func(agents []ui.SettingsCategory) []ui.SettingsCategory { return settingsCategories(cx, n, agents) }}
}

// settingsCategories are Scenra's own categories around the agents'. The
// controls hold their state in scopes of their own, as the page is built
// again when the agents change.
func settingsCategories(cx *core.Scope, n *namer, agents []ui.SettingsCategory) []ui.SettingsCategory {
	tr := locales.For(cx)
	general := ui.SettingsCategory{ID: "general", Title: tr.SettingsGeneral(), Icon: lucide.Settings, Groups: []ui.SettingsGroup{{
		ID: "general", Items: []ui.SettingsItem{{
			ID: "language", Title: tr.LanguageSection(), Keywords: append([]string{"language"}, languageNames()...), Control: core.Func(languageSelect),
		}, {
			ID: "theme", Title: tr.Theme(), Keywords: []string{"theme", "appearance", "dark", "light", tr.ThemeLight(), tr.ThemeDark()},
			Control: core.Func(themeSelect),
		}}}}}
	stats := ui.SettingsCategory{ID: "stats", Title: tr.StatsSection(), Icon: lucide.ChartColumn, Groups: []ui.SettingsGroup{{
		ID: "stats", Items: []ui.SettingsItem{{
			ID: "send", Title: tr.StatsSend(), Description: tr.StatsDetail(), Keywords: []string{"statistics", "privacy"}, Control: core.Func(statsSwitch),
		}}}}}
	out := append([]ui.SettingsCategory{general}, agents...)
	return append(out, namingSettings(cx, n), stats)
}
