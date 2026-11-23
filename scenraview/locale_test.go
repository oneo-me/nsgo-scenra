package scenraview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The catalogs' language is matched to the one shown as NSGo matches an
// app's, and a message missing in it falls back to the text, then the
// source.
func TestTextsSayTheirMessagesInTheLanguageShown(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"locales/en.yaml":      "hi: Hello\nbye: Bye\nonly: Only English\n",
		"locales/zh-Hans.yaml": "hi: 你好\nbye: 再见\n",
		"locales/de.yaml":      "hi: Hallo\n",
	})
	_, _, pics, _, _ := scan(dir, nil)
	for _, c := range []struct{ shown, key, text, want string }{
		{"en", "hi", "", "Hello"},
		{"en-GB", "hi", "", "Hello"},
		{"zh-Hans", "hi", "", "你好"},
		{"zh-CN", "bye", "", "再见"},
		{"de-CH", "hi", "", "Hallo"},
		{"de", "bye", "Tschüss", "Tschüss"},
		{"de", "bye", "", "Bye"},
		{"ja", "hi", "", "Hello"},
		{"zh-Hans", "gone", "", "gone"},
	} {
		if got := pics.say(c.shown, c.key, c.text); got != c.want {
			t.Errorf("say(%q, %q, %q) = %q, want %q", c.shown, c.key, c.text, got, c.want)
		}
	}
	if got := (assets{}).say("en", "hi", "Hi"); got != "Hi" {
		t.Fatalf("without catalogs says %q", got)
	}
}

func TestCheckTellsOfMessagesAndCatalogs(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":            "design\ntext key=hi\ntext key=gone\nuse card title.key=bye\n",
		"components/card.scenra": "component\ntext key=hi name=title\nvariant other title.key=late\n",
		"locales/en.yaml":        "hi: Hello\nbye: Bye\nlate: Late\n",
		"locales/zh-Hans.yaml":   "hi: 你好\n",
		"locales/pt-br.yaml":     "hi: Olá\n",
		"locales/fr.yaml":        "- x\n",
	})
	got, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"home.scenra: key=gone: no message in locales/en.yaml, locales/zh-Hans.yaml",
		"home.scenra: key=bye: no message in locales/zh-Hans.yaml",
		"components/card.scenra: key=late: no message in locales/zh-Hans.yaml",
		"locales/pt-br.yaml is not named by a canonical BCP 47 tag; name it locales/pt-BR.yaml",
		"locales/fr.yaml: want a mapping",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "key=hi") {
		t.Errorf("a key every catalog has:\n%s", got)
	}
	bare := folderOfFiles(t, map[string]string{"home.scenra": "design\ntext key=hi\n"})
	if got, _ := Check(bare); !strings.Contains(got, "key=hi: there are no catalogs; write the messages in locales/, such as locales/en.yaml") {
		t.Errorf("without catalogs:\n%s", got)
	}
}

func TestDesignsShowTheMessagesOfTheCatalogs(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":            "design\ntext key=hi\nuse card\nuse card title=Mocha\n",
		"components/card.scenra": "component\ntext Latte name=title key=latte\n",
		"locales/en.yaml":        "hi: Hello\nlatte: Latte macchiato\n",
	})
	app, e := startEditor(t)
	openPath(t, app, e, filepath.Join(dir, "home.scenra"))
	app.WaitFor("design >> text:Hello", 5*time.Second)
	app.Find("design >> text:Latte macchiato")
	app.Find("design >> text:Mocha")
	// Changing a catalog redraws the designs showing its messages.
	os.WriteFile(filepath.Join(dir, "locales", "en.yaml"), []byte("hi: Hello there\nlatte: Latte macchiato\n"), 0o644)
	app.WaitFor("design >> text:Hello there", 5*time.Second)
}
