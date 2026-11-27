package main

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/cmd/scenra/locales"
)

// keyTheme keeps the appearance chosen in the settings; without it the
// app follows the system's.
const keyTheme = "theme"

// rememberAppearance shows the appearance chosen last time.
func rememberAppearance(cx *core.Scope) {
	switch theme, _ := cx.Storage().GetItem(keyTheme); theme {
	case "light":
		cx.SetAppearance(core.AppearanceLight)
	case "dark":
		cx.SetAppearance(core.AppearanceDark)
	}
}

// setAppearance shows a, core.AppearanceSystem the system's, and keeps the
// choice.
func setAppearance(cx *core.Scope, a core.Appearance) {
	cx.SetAppearance(a)
	switch a {
	case core.AppearanceLight:
		cx.Storage().SetItem(keyTheme, "light")
	case core.AppearanceDark:
		cx.Storage().SetItem(keyTheme, "dark")
	default:
		cx.Storage().RemoveItem(keyTheme)
	}
}

// themeSelect chooses the appearance: the system's, light or dark.
func themeSelect(cx *core.Scope) core.Node {
	tr := locales.For(cx)
	return ui.Select[core.Appearance]{Options: []ui.Option[core.Appearance]{
		{Value: core.AppearanceSystem, Label: tr.ThemeSystem()},
		{Value: core.AppearanceLight, Label: tr.ThemeLight()},
		{Value: core.AppearanceDark, Label: tr.ThemeDark()},
	}, Value: cx.Env().Appearance, OnChange: func(a core.Appearance) { setAppearance(cx, a) },
		Label: tr.Theme(), TestID: "theme", Style: core.Style{Width: core.Pt(180)}}
}
