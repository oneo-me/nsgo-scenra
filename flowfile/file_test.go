package flowfile

import (
	"slices"
	"strings"
	"testing"
)

const design = `design device=tablet
text "Sign in" class="text-2xl font-bold"
image "A cup" src="https://example.com/cup.png"
box class="flex-row gap-2"
  icon lock class="size-4"
  text "Remember me"
end
svg "<svg viewBox='0 0 10 10'><circle cx='5' cy='5' r='4'/></svg>"
`

func TestParseDesign(t *testing.T) {
	f, err := Parse(design)
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != Design || f.Device != "tablet" || len(f.UI) != 4 || len(f.UI[2].Children) != 2 || f.UI[1].Attr("src") != "https://example.com/cup.png" {
		t.Fatalf("design: %+v", f)
	}
	again, err := Parse(f.Format())
	if err != nil {
		t.Fatalf("%v\n%s", err, f.Format())
	}
	if again.Format() != f.Format() {
		t.Fatalf("not stable:\n%s\n---\n%s", f.Format(), again.Format())
	}
	if !strings.HasPrefix(f.Format(), "design device=tablet\n") || !strings.Contains(f.Format(), "box class=\"flex-row gap-2\"\n  icon \"lock\" class=\"size-4\"\n") {
		t.Fatalf("format:\n%s", f.Format())
	}
	if empty, err := Parse("design\n"); err != nil || empty.Kind != Design || empty.Format() != "design\n" {
		t.Fatalf("an empty design: %+v %v", empty, err)
	}
}

func TestParseFileErrors(t *testing.T) {
	for src, want := range map[string]string{
		"":                                 "line 1: a file starts with design, component or flow",
		"text \"x\"":                       "a file starts with design, component or flow",
		"design\nwidget":                   "line 2: unknown element",
		"design\nbox\ntext \"x\"":          "line 3: box has no end",
		"design\nend":                      "end closes no",
		"design\ntext \"x\" src=y":         "text takes no attribute \"src\"; use class",
		"design\ntext x y":                 "quote text with spaces",
		"design\nicon nope":                "unknown icon \"nope\"",
		"design\nicon cofee-cup":           "such as cup-soda",
		"design\nsvg \"<div/>\"":           "svg needs a document",
		"design\nsvg \"<svg>\"":            "svg XML syntax error",
		"design device=watch":              "unknown device",
		"design width=wide":                "width=\"wide\" is not a size",
		"design height=20":                 "height=\"20\" is not a size",
		"design direction=right":           "a design takes name, device",
		"flow device=phone":                "a flow takes name and direction",
		"design\nflow":                     "line 2: a file holds one design, component or flow",
		"flow\na -> b\ndesign":             "line 3: a file holds one design, component or flow",
		"design\ncomponent":                "line 2: a file holds one design, component or flow",
		"component direction=down":         "a component takes name, device",
		"component\nnode a":                "a component holds no flow",
		"design\nuse":                      "\"\" names no component",
		"design\nuse \"a b\"":              "names no component",
		"design\nuse ../secret":            "names no component",
		"design\nuse x\n  text \"y\"\nend": "end closes no box",
		"design\nnode a":                   "a design holds no flow",
		"design\na -> b":                   "a design holds no flow",
	} {
		if _, err := Parse(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
}

func TestDesignSize(t *testing.T) {
	f, err := Parse("design device=tablet width=1024 height=1366\n")
	if err != nil {
		t.Fatal(err)
	}
	if w, h := f.Size(); w != 1024 || h != 1366 || f.Format() != "design device=tablet width=1024 height=1366\n" {
		t.Fatalf("%d×%d:\n%s", w, h, f.Format())
	}
	if f, _ := Parse("design height=600\n"); !sizeIs(f, 390, 600) {
		t.Fatalf("a phone 600 high: %+v", f)
	}
	if f, _ := Parse("component width=320\n"); !sizeIs(f, 320, 0) {
		t.Fatalf("a component 320 wide: %+v", f)
	}
}

func TestFileName(t *testing.T) {
	for src, want := range map[string]string{
		"design device=tablet name=\"Sign in\"\n":  "design name=\"Sign in\" device=tablet\n",
		"component name=Tabs\n":                    "component name=\"Tabs\"\n",
		"flow direction=right name=\"Checkout\"\n": "flow name=\"Checkout\" direction=right\n",
		"design name=\" \"\n":                      "design\n",
	} {
		f, err := Parse(src)
		if err != nil || f.Format() != want {
			t.Errorf("%q: %v\n%s", src, err, f.Format())
		}
	}
	if f, _ := Parse("flow name=\"Checkout\"\n"); f.Name != "Checkout" {
		t.Fatalf("name %q", f.Name)
	}
}

func sizeIs(f File, w, h int) bool {
	gw, gh := f.Size()
	return gw == w && gh == h
}

func TestComponents(t *testing.T) {
	src := "component device=tablet\nbox class=\"flex-row\"\n  use nav/item class=\"flex-1\"\nend\n"
	f, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != Component || f.Device != "tablet" || f.Format() != strings.Replace(src, "use nav/item", `use "nav/item"`, 1) {
		t.Fatalf("component %+v:\n%s", f, f.Format())
	}
	if got := Uses(f.UI); !slices.Equal(got, []string{"nav/item"}) {
		t.Fatalf("uses %v", got)
	}
	for p, want := range map[string]string{"components/tab-bar.scenra": "tab-bar", "components/nav/item.scenra": "nav/item", "tab-bar.scenra": "", "components/x.txt": ""} {
		if got, ok := ComponentName(p); got != want || ok != (want != "") {
			t.Errorf("%s names %q %v", p, got, ok)
		}
	}
	if ComponentPath("nav/item") != "components/nav/item.scenra" {
		t.Fatal(ComponentPath("nav/item"))
	}
}

func TestElementPaths(t *testing.T) {
	els, err := ParseUI("text \"a\"\nbox\n  text \"b\"\n  text \"c\"\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	if el, ok := ElementAt(els, "1.1"); !ok || el.Text != "c" {
		t.Fatalf("1.1 = %+v", el)
	}
	if _, ok := ElementAt(els, "0.0"); ok {
		t.Fatal("a text holds an element")
	}
	with, _ := ParseUI("text \"x\"\ntext \"y\"\n")
	out, err := ReplaceElement(els, "1.0", with)
	if err != nil || FormatUI(out) != "text \"a\"\nbox\n  text \"x\"\n  text \"y\"\n  text \"c\"\nend\n" {
		t.Fatalf("%v\n%s", err, FormatUI(out))
	}
	if FormatUI(els) != "text \"a\"\nbox\n  text \"b\"\n  text \"c\"\nend\n" {
		t.Fatal("replacing changed the elements")
	}
	if _, err := ReplaceElement(els, "5", nil); err == nil {
		t.Fatal("replaced a missing element")
	}
}

func TestComments(t *testing.T) {
	p, err := Parse("design\n  box comment=\"Roomier\" class=\"p-2\"\n    text \"a\"\n  end\n  icon mail comment=\"Bigger\"\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []Comment{{"0", "Roomier"}, {"1", "Bigger"}}
	if got := p.Comments(); !slices.Equal(got, want) {
		t.Fatalf("comments %v", got)
	}
	if got := FormatUI(p.UI[:1]); got != "box class=\"p-2\" comment=\"Roomier\"\n  text \"a\"\nend\n" {
		t.Fatalf("formatted as\n%s", got)
	}
	els, err := SetComment(p.UI, "0.0", "Bolder")
	if err != nil {
		t.Fatal(err)
	}
	if els, _ = SetComment(els, "0", ""); FormatUI(els[:1]) != "box class=\"p-2\"\n  text \"a\" comment=\"Bolder\"\nend\n" {
		t.Fatalf("set as\n%s", FormatUI(els))
	}
	if p.UI[0].Attr("comment") != "Roomier" || p.UI[0].Children[0].Attr("comment") != "" {
		t.Fatal("setting a comment changed the elements")
	}
	if _, err := SetComment(els, "3", "x"); err == nil {
		t.Fatal("commented a missing element")
	}
}

func TestNotes(t *testing.T) {
	p, err := Parse("design\nbox note=\"Scrolls\" comment=\"Roomier\"\n  text \"a\" note=\"Live\"\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := p.Notes(), []Comment{{"0", "Scrolls"}, {"0.0", "Live"}}; !slices.Equal(got, want) {
		t.Fatalf("notes %v", got)
	}
	els, _ := SetNote(p.UI, "0", "")
	if els, _ = SetNote(els, "0.0", "Updates live"); FormatUI(els) != "box comment=\"Roomier\"\n  text \"a\" note=\"Updates live\"\nend\n" {
		t.Fatalf("set as\n%s", FormatUI(els))
	}
}

func TestTaps(t *testing.T) {
	p, err := Parse("design\nbox tap=\"open orders/detail\" class=\"p-2\"\n  text \"x\" tap=close\nend\n" +
		"box tap=\"tab past\"\nend\nbox panel=past\nend\nbox tap=\"open Sign in\"\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]Tap{"0": {TapOpen, "orders/detail"}, "0.0": {TapClose, ""}, "1": {TapTab, "past"}, "3": {TapOpen, "Sign in"}} {
		el, _ := ElementAt(p.UI, path)
		if got, ok := el.Tap(); !ok || got != want {
			t.Errorf("%s taps %+v", path, got)
		}
	}
	if el, _ := ElementAt(p.UI, "2"); el.Attr("panel") != "past" {
		t.Fatalf("the panel is %q", el.Attr("panel"))
	}
	if got := FormatUI(p.UI[:1]); got != "box class=\"p-2\" tap=\"open orders/detail\"\n  text \"x\" tap=\"close\"\nend\n" {
		t.Fatalf("formatted as\n%s", got)
	}
	for _, src := range []string{"box tap=jump\nend\n", "box tap=open\nend\n", "box tap=\"open ../x\"\nend\n", "box tap=\"close now\"\nend\n",
		"box tap=\"tab a b\"\nend\n", "box panel=\"a b\"\nend\n"} {
		if _, err := ParseUI(src); err == nil {
			t.Errorf("%q parsed", src)
		}
	}
}

func TestIconsTakeLucideNamesAndAliases(t *testing.T) {
	els, err := ParseUI("icon \"alert-circle\"\nicon \"badge-dollar-sign\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if els[0].Text != "circle-alert" || els[1].Text != "badge-dollar-sign" {
		t.Fatalf("icons read as %q and %q", els[0].Text, els[1].Text)
	}
	for alias, name := range iconAliases {
		if _, ok := IconName(name); !ok {
			t.Errorf("alias %s names no icon %s", alias, name)
		}
	}
	for _, name := range CommonIcons {
		if _, ok := IconName(name); !ok {
			t.Errorf("no common icon %s", name)
		}
	}
}
