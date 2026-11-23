package flowfile

import (
	"slices"
	"strings"
	"testing"
)

func TestCatalogsAreNamedByTheirLanguages(t *testing.T) {
	for p, want := range map[string]string{"locales/en.yaml": "en", "locales/zh-Hans.yaml": "zh-Hans", "locales/pt-BR.yaml": "pt-BR",
		"locales/zh-hans.yaml": "", "locales/english.yaml": "", "locales/en.yml": "", "a/locales/en.yaml": "", "locales/x/en.yaml": ""} {
		if got, _ := LocaleLanguage(p); got != want && want != "" || want == "" && func() bool { _, ok := LocaleLanguage(p); return ok }() {
			t.Errorf("LocaleLanguage(%q) = %q, want %q", p, got, want)
		}
	}
	if !IsLocale("locales/zh-hans.yaml") || IsLocale("locales/en.yml") {
		t.Fatal("IsLocale")
	}
	if got := LocaleProblem("locales/zh-hans.yaml"); !strings.Contains(got, "locales/zh-Hans.yaml") {
		t.Fatalf("LocaleProblem says %q", got)
	}
}

func TestParseLocaleReadsNSGoCatalogs(t *testing.T) {
	msgs, err := ParseLocale([]byte("signIn: Sign in\nwelcome:\n  message: \"Welcome, {name}!\"\n  description: The greeting.\ngithub.com/x/locales:\n  ok: OK\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs["signIn"] != "Sign in" || msgs["welcome"] != "Welcome, {name}!" {
		t.Fatalf("messages %v", msgs)
	}
	for src, want := range map[string]string{"- a\n": "mapping", "9lives: x\n": "want a letter", "a:\n  description: x\n": "no message", "a: [x]\n": "want a message"} {
		if _, err := ParseLocale([]byte(src)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
}

func TestTextsNameMessagesByKey(t *testing.T) {
	els, err := ParseUI("text key=signIn class=\"font-bold\"\ntext \"Hi\" key=hi\nuse card title.key=latte\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatUI(els); got != "text key=\"signIn\" class=\"font-bold\"\ntext \"Hi\" key=\"hi\"\nuse \"card\" title.key=\"latte\"\n" {
		t.Fatalf("formatted as\n%s", got)
	}
	if got := Keys(els); !slices.Equal(got, []string{"signIn", "hi", "latte"}) {
		t.Fatalf("keys %v", got)
	}
	card, _ := ParseUI("box\n  text \"Title\" name=title key=title\n  icon coffee name=icon\nend\n")
	parts := map[string]Part{"card": {UI: card}}
	find := func(name string) (Part, bool) { p, ok := parts[name]; return p, ok }
	// A use gives a text another key, or a text of its own in place of
	// the message.
	if got := Instance(els[2], parts["card"])[0].Children[0].Attr("key"); got != "latte" {
		t.Fatalf("the use's key is %q", got)
	}
	own, _ := ParseUI("use card title=Mocha\n")
	if got := Instance(own[0], parts["card"])[0].Children[0]; got.Text != "Mocha" || got.Attr("key") != "" {
		t.Fatalf("the use's text drew as %v", got)
	}
	bad, _ := ParseUI("use card icon.key=x\n")
	if p := OverrideProblems(bad[0], find); len(p) != 1 || !strings.Contains(p[0], "sets the key of icon, a icon") {
		t.Fatalf("problems %v", p)
	}
	for src, want := range map[string]string{
		"text key=\"a-b\"":      "not a message key",
		"use card title.key=1a": "not a message key",
		"icon coffee key=x":     "icon takes no attribute \"key\"",
		"text \"x\" name=key":   "name \"key\" is not a name",
	} {
		if _, err := ParseUI(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
}
