package scenraview

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/nsgo-dev/nsgo/i18n"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// The catalogs of a project's locales folder are read with its pictures,
// among its assets, so every view drawing designs from them redraws as
// one changes. A text naming a message by key= shows it in the language
// the app shows, matched against the catalogs' languages as NSGo matches
// an app's, so a design reads as the app will in the user's language.

// localeSource is the language a project's messages fall back to when the
// language shown has none, as NSGo's catalogs are written in English,
// else the first of its languages.
const localeSource = "en"

// languages returns the languages of the project's catalogs, the one its
// messages fall back to first, nil without any.
func (a assets) languages() []string {
	var out []string
	for p, pic := range a {
		if lang, ok := flowfile.LocaleLanguage(p); ok && pic.msgs != nil {
			out = append(out, lang)
		}
	}
	slices.Sort(out)
	if i := slices.Index(out, localeSource); i > 0 {
		out = slices.Insert(slices.Delete(out, i, i+1), 0, localeSource)
	}
	return out
}

// messages returns the messages of the catalog of lang, nil without one.
func (a assets) messages(lang string) flowfile.Messages {
	return a[localePath(lang)].msgs
}

// localePath is the path in the project of the catalog of lang.
func localePath(lang string) string { return flowfile.LocalesDir + "/" + lang + ".yaml" }

// catalogs keeps a catalog per set of languages, which remembers the
// locales it matched.
var catalogs sync.Map // string → *i18n.Catalog

func catalogOf(langs []string) *i18n.Catalog {
	id := strings.Join(langs, " ")
	if c, ok := catalogs.Load(id); ok {
		return c.(*i18n.Catalog)
	}
	c, _ := catalogs.LoadOrStore(id, i18n.NewCatalog(langs[0], langs[1:]...))
	return c.(*i18n.Catalog)
}

// say returns the message key names in the catalog whose language suits
// shown best, or in the languages it falls back to by CLDR's parents, as
// NSGo's messages do; else text, the element's own, else the message in
// the source language, else the key, so a missing message shows.
func (a assets) say(shown, key, text string) string {
	langs := a.languages()
	if len(langs) > 0 {
		for lang := catalogOf(langs).Match(shown); lang != ""; lang = i18n.Parent(lang) {
			if m, ok := a.messages(lang)[key]; ok {
				return m
			}
		}
	}
	if text != "" {
		return text
	}
	if len(langs) > 0 {
		if m, ok := a.messages(langs[0])[key]; ok {
			return m
		}
	}
	return key
}

// localeProblems are what is wrong with the catalogs of a project's
// locales folder: names that are no language and files that do not read.
func localeProblems(pics assets) []string {
	var out []string
	for _, p := range slices.Sorted(maps.Keys(pics)) {
		if !flowfile.IsLocale(p) {
			continue
		}
		if _, ok := flowfile.LocaleLanguage(p); !ok {
			out = append(out, flowfile.LocaleProblem(p))
		} else if err := pics[p].err; err != "" {
			out = append(out, p+": "+err)
		}
	}
	return out
}

// keyProblems are what is wrong with the message keys els and the
// variants vars name: keys of a project without catalogs, and keys some
// catalog lacks.
func keyProblems(els []flowfile.Element, vars []flowfile.Variant, pics assets) []string {
	all := els
	for _, v := range vars {
		all = append(all, v.Use(""))
	}
	keys := flowfile.Keys(all)
	if len(keys) == 0 {
		return nil
	}
	langs := pics.languages()
	if len(langs) == 0 {
		return []string{fmt.Sprintf("key=%s: there are no catalogs; write the messages in %s, such as %s", strings.Join(keys, ", key="), flowfile.LocalesDir+"/", localePath(localeSource))}
	}
	var out []string
	for _, key := range keys {
		var lacking []string
		for _, lang := range langs {
			if _, ok := pics.messages(lang)[key]; !ok {
				lacking = append(lacking, localePath(lang))
			}
		}
		if len(lacking) > 0 {
			out = append(out, fmt.Sprintf("key=%s: no message in %s", key, strings.Join(lacking, ", ")))
		}
	}
	return out
}
