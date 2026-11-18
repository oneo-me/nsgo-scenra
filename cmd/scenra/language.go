package main

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/cmd/scenra/locales"
)

// keyLanguage keeps the language chosen in the settings; without it the
// app follows the system's.
const keyLanguage = "language"

// rememberLanguage shows the language chosen last time.
func rememberLanguage(cx *core.Scope) {
	if lang, ok := cx.Storage().GetItem(keyLanguage); ok && lang != "" {
		cx.SetLocale(lang)
	}
}

// setLanguage shows lang, "" the system's, and keeps the choice.
func setLanguage(cx *core.Scope, lang string) {
	cx.SetLocale(lang)
	if lang == "" {
		cx.Storage().RemoveItem(keyLanguage)
	} else {
		cx.Storage().SetItem(keyLanguage, lang)
	}
}

// languageSection chooses the language the app shows: the system's or one
// of its catalog's, each named in itself.
func languageSection(cx *core.Scope) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	initial, _ := cx.Storage().GetItem(keyLanguage)
	chosen := core.NewSignal(cx, initial)
	options := []ui.Option[string]{{Value: "", Label: tr.LanguageSystem()}}
	for _, lang := range locales.Catalog.Languages() {
		options = append(options, ui.Option[string]{Value: lang, Label: locales.In(lang).LanguageName()})
	}
	return core.View{Style: core.Style{Gap: 8}, Children: []core.Node{
		ui.Text(tr.LanguageSection(), 15, ui.Semibold, colors.Text),
		ui.Select[string]{Options: options, Value: chosen.Get, OnChange: func(lang string) { chosen.Set(lang); setLanguage(cx, lang) },
			Label: tr.LanguageSection(), TestID: "language"},
	}}
}
