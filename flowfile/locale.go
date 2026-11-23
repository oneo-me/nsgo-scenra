package flowfile

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
)

// LocalesDir is the folder of a project's catalogs, at its top: a YAML
// file of messages per language, named by its canonical BCP 47 tag as
// NSGo's catalogs are, such as locales/en.yaml and locales/zh-Hans.yaml,
// so a design's texts show in the language the editor shows and an app
// built from it can take the catalogs as they are. A text's key= names
// its message.
const LocalesDir = "locales"

// LocaleLanguage returns the language of the catalog at p, relative to
// the project's folder with slashes, and whether p is one: a file of the
// locales folder named by a canonical BCP 47 tag.
func LocaleLanguage(p string) (string, bool) {
	lang, ok := localeName(p)
	if !ok {
		return "", false
	}
	t, err := language.Parse(lang)
	return lang, err == nil && t.String() == lang && !t.IsRoot()
}

// IsLocale reports whether the file at p is a YAML file of the locales
// folder, a catalog unless LocaleLanguage says its name is no language.
func IsLocale(p string) bool {
	_, ok := localeName(p)
	return ok
}

func localeName(p string) (string, bool) {
	p = path.Clean(p)
	if path.Dir(p) != LocalesDir || path.Ext(p) != ".yaml" {
		return "", false
	}
	return strings.TrimSuffix(path.Base(p), ".yaml"), true
}

// LocaleProblem says what is wrong with the name of the catalog at p.
func LocaleProblem(p string) string {
	lang, _ := localeName(p)
	if t, err := language.Parse(lang); err == nil && !t.IsRoot() {
		return fmt.Sprintf("%s is not named by a canonical BCP 47 tag; name it %s/%s.yaml", p, LocalesDir, t)
	}
	return fmt.Sprintf("%s is not named by a BCP 47 tag, such as %s/en.yaml or %s/zh-Hans.yaml", p, LocalesDir, LocalesDir)
}

// Messages are a catalog's messages by their keys.
type Messages map[string]string

// ParseLocale reads a catalog as NSGo does: a mapping of keys to
// messages, each a string or a mapping whose message field holds it. A
// key of another package's catalog, an import path, is left out, as
// designs name only their own.
func ParseLocale(data []byte) (Messages, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	out := Messages{}
	if len(doc.Content) == 0 {
		return out, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, errors.New("want a mapping of keys to messages")
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		k, v := root.Content[i], root.Content[i+1]
		if strings.ContainsAny(k.Value, "/.") {
			continue
		}
		if !ValidKey(k.Value) {
			return nil, fmt.Errorf("line %d: key %q: want a letter, then letters, digits and _", k.Line, k.Value)
		}
		switch v.Kind {
		case yaml.ScalarNode:
			out[k.Value] = v.Value
		case yaml.MappingNode:
			found := false
			for j := 0; j+1 < len(v.Content); j += 2 {
				if v.Content[j].Value == "message" {
					out[k.Value], found = v.Content[j+1].Value, true
				}
			}
			if !found {
				return nil, fmt.Errorf("line %d: %s has no message", v.Line, k.Value)
			}
		default:
			return nil, fmt.Errorf("line %d: %s: want a message", v.Line, k.Value)
		}
	}
	return out, nil
}

// ValidKey reports whether key can name a message, as NSGo's catalogs
// allow: a letter, then letters, digits and _.
func ValidKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		if !letter && (i == 0 || r != '_' && (r < '0' || r > '9')) {
			return false
		}
	}
	return true
}

// Keys are the message keys els name, and those uses give the elements
// of their components, each once, in the order they are first written.
func Keys(els []Element) []string {
	var out []string
	add := func(key string) {
		if key != "" && !slices.Contains(out, key) {
			out = append(out, key)
		}
	}
	var walk func(els []Element)
	walk = func(els []Element) {
		for _, el := range els {
			add(el.Attr("key"))
			if el.Kind == "use" {
				for _, k := range overrideKeys(el.Attrs) {
					if strings.HasSuffix(k, ".key") {
						add(el.Attrs[k])
					}
				}
			}
			walk(el.Children)
		}
	}
	walk(els)
	return out
}

func keyHelp(key string) string {
	return fmt.Sprintf("key %q is not a message key of a letter, then letters, digits and _, such as signInTitle", key)
}
