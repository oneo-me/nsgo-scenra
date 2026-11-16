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
	ui := f.Screens[0].UI
	if f.Kind != Design || f.Screens[0].Device != "tablet" || len(ui) != 4 || len(ui[2].Children) != 2 || ui[1].Attr("src") != "https://example.com/cup.png" {
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
		"design\ntext \"x\" src=y":         "text takes no attribute \"src\"; use name, class",
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
		"design\nflow":                     "line 2: a file holds designs, a component or flows, never two kinds",
		"flow\na -> b\ndesign":             "line 3: a file holds designs, a component or flows, never two kinds",
		"design\ncomponent":                "line 2: a file holds designs, a component or flows, never two kinds",
		"component\ncomponent":             "line 2: a component file holds one component",
		"component tabs":                   "unexpected \"tabs\"",
		"design a\ndesign":                 "line 1: a file of several designs gives each an ID",
		"flow a\nflow a":                   "two flows have the ID \"a\"",
		"design \"a b\"":                   "unexpected \"a b\"",
		"design a.b/c":                     "is not an ID",
		"design\ndesign b gap=10":          "line 2: gap is the file's",
		"design gap=-4":                    "gap=\"-4\" is not a gap",
		"design\nbox\ndesign b":            "line 3: box has no end",
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
	if w, h := f.Size(0); w != 1024 || h != 1366 || f.Format() != "design device=tablet width=1024 height=1366\n" {
		t.Fatalf("%d×%d:\n%s", w, h, f.Format())
	}
	if f, _ := Parse("design height=600\n"); !sizeIs(f, 390, 600) {
		t.Fatalf("a phone 600 high: %+v", f)
	}
	if f, _ := Parse("component width=320\n"); !sizeIs(f, 320, 0) {
		t.Fatalf("a component 320 wide: %+v", f)
	}
	// A component is as large as it needs unless it fills the width.
	if f, _ := Parse("component device=tablet\n"); !sizeIs(f, 0, 0) {
		t.Fatalf("a component of no size: %+v", f)
	}
	f, err = Parse("component device=tablet width=fill backdrop=\"bg-inset p-2\"\n")
	if err != nil || !sizeIs(f, 820, 0) || f.Screens[0].Backdrop != "bg-inset p-2" ||
		f.Format() != "component device=tablet width=fill backdrop=\"bg-inset p-2\"\n" {
		t.Fatalf("a component filling the width: %v %+v\n%s", err, f, f.Format())
	}
}

func TestVariants(t *testing.T) {
	src := "component\nvariant off\nbox name=track class=\"w-10\"\n  box name=knob\n  end\nend\nvariant on track.class=\"bg-blue-600\" class=\"mt-1\" knob.class=\"ml-auto\"\n"
	f, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	s := f.Screens[0]
	if len(s.Variants) != 2 || s.Variants[1].ID != "on" || s.Variants[1].Attrs["knob.class"] != "ml-auto" {
		t.Fatalf("variants %+v", s.Variants)
	}
	want := "component\nbox name=\"track\" class=\"w-10\"\n  box name=\"knob\"\n  end\nend\nvariant off\nvariant on class=\"mt-1\" knob.class=\"ml-auto\" track.class=\"bg-blue-600\"\n"
	if f.Format() != want {
		t.Fatalf("formatted as\n%s", f.Format())
	}
	// A variant draws as a use changing the component so would.
	if got := FormatUI(Instance(s.Variants[1].Use(""), Part{UI: s.UI, Variants: s.Variants})); got != "box name=\"track\" class=\"w-10 mt-1 bg-blue-600\"\n  box name=\"knob\" class=\"ml-auto\"\n  end\nend\n" {
		t.Fatalf("drawn as\n%s", got)
	}
	f.Screens[0].Variants = append(f.Screens[0].Variants, Variant{ID: "x", Attrs: map[string]string{"gone.class": "p-1"}})
	if p := VariantProblems(f.Screens[0], "switch", func(string) (Part, bool) { return Part{}, false }); len(p) != 1 ||
		!strings.Contains(p[0], "variant x: gone.class=\"p-1\" names nothing: no element of components/switch.scenra has name=gone") {
		t.Fatalf("problems %v", p)
	}
	for src, want := range map[string]string{
		"component\nvariant":                     "variant needs an ID",
		"component\nvariant a\nvariant a":        "two variants have the ID \"a\"",
		"component\nvariant a tap=close":         "a variant takes no tap",
		"component\nvariant a title":             "unexpected \"title\"",
		"component\nbox\n  variant a\nend":       "a variant is a line of the component's file, outside its boxes",
		"design\nvariant a":                      "only a component has variants",
		"component backdrop=x\nvariant a ti:t=x": "takes no attribute \"ti:t\"",
	} {
		if _, err := Parse(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
	if _, err := Parse("design width=fill\n"); err == nil {
		t.Error("a design fills a width")
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
	if f, _ := Parse("flow name=\"Checkout\"\n"); f.Title() != "Checkout" {
		t.Fatalf("name %q", f.Title())
	}
}

func sizeIs(f File, w, h int) bool {
	gw, gh := f.Size(0)
	return gw == w && gh == h
}

func TestComponents(t *testing.T) {
	src := "component device=tablet\nbox class=\"flex-row\"\n  use nav/item class=\"flex-1\"\nend\n"
	f, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != Component || f.Screens[0].Device != "tablet" || f.Format() != strings.Replace(src, "use nav/item", `use "nav/item"`, 1) {
		t.Fatalf("component %+v:\n%s", f, f.Format())
	}
	if got := Uses(f.Screens[0].UI); !slices.Equal(got, []string{"nav/item"}) {
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
	want := []Comment{{0, "0", "Roomier"}, {0, "1", "Bigger"}}
	if got := p.Comments(); !slices.Equal(got, want) {
		t.Fatalf("comments %v", got)
	}
	if got := FormatUI(p.Screens[0].UI[:1]); got != "box class=\"p-2\" comment=\"Roomier\"\n  text \"a\"\nend\n" {
		t.Fatalf("formatted as\n%s", got)
	}
	els, err := SetComment(p.Screens[0].UI, "0.0", "Bolder")
	if err != nil {
		t.Fatal(err)
	}
	if els, _ = SetComment(els, "0", ""); FormatUI(els[:1]) != "box class=\"p-2\"\n  text \"a\" comment=\"Bolder\"\nend\n" {
		t.Fatalf("set as\n%s", FormatUI(els))
	}
	if p.Screens[0].UI[0].Attr("comment") != "Roomier" || p.Screens[0].UI[0].Children[0].Attr("comment") != "" {
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
	if got, want := p.Notes(), []Comment{{0, "0", "Scrolls"}, {0, "0.0", "Live"}}; !slices.Equal(got, want) {
		t.Fatalf("notes %v", got)
	}
	els, _ := SetNote(p.Screens[0].UI, "0", "")
	if els, _ = SetNote(els, "0.0", "Updates live"); FormatUI(els) != "box comment=\"Roomier\"\n  text \"a\" note=\"Updates live\"\nend\n" {
		t.Fatalf("set as\n%s", FormatUI(els))
	}
}

func TestTaps(t *testing.T) {
	p, err := Parse("design\nbox tap=\"open orders/detail\" class=\"p-2\"\n  text \"x\" tap=close\nend\n" +
		"box tap=\"tab past\"\nend\nbox panel=past\nend\nbox tap=\"open Sign in\"\nend\nbox tap=\"open auth#sign-up\"\nend\nbox tap=\"open #reset\"\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]Tap{"0": {TapOpen, "orders/detail"}, "0.0": {TapClose, ""}, "1": {TapTab, "past"}, "3": {TapOpen, "Sign in"}, "4": {TapOpen, "auth#sign-up"}, "5": {TapOpen, "#reset"}} {
		el, _ := ElementAt(p.Screens[0].UI, path)
		if got, ok := el.Tap(); !ok || got != want {
			t.Errorf("%s taps %+v", path, got)
		}
	}
	if el, _ := ElementAt(p.Screens[0].UI, "2"); el.Attr("panel") != "past" {
		t.Fatalf("the panel is %q", el.Attr("panel"))
	}
	if got := FormatUI(p.Screens[0].UI[:1]); got != "box class=\"p-2\" tap=\"open orders/detail\"\n  text \"x\" tap=\"close\"\nend\n" {
		t.Fatalf("formatted as\n%s", got)
	}
	for _, src := range []string{"box tap=jump\nend\n", "box tap=open\nend\n", "box tap=\"open ../x\"\nend\n", "box tap=\"close now\"\nend\n",
		"box tap=\"tab a b\"\nend\n", "box panel=\"a b\"\nend\n", "box tap=\"open #\"\nend\n", "box tap=\"open a#b c\"\nend\n", "box tap=\"open a#b#c\"\nend\n"} {
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

func TestOverrides(t *testing.T) {
	card, err := ParseUI("box name=card class=\"p-4\"\n  icon coffee name=icon class=\"size-5\"\n  text \"Title\" name=title class=\"font-bold\"\n  use badge name=badge\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	badge, _ := ParseUI("text \"New\" name=label class=\"text-xs\"\n")
	parts := map[string]Part{"card": {UI: card}, "badge": {UI: badge}, "pill": {UI: badge}}
	find := func(name string) (Part, bool) { p, ok := parts[name]; return p, ok }
	use, err := ParseUI("use card title=\"Latte\" title.class=\"text-rose-600\" icon=home badge/label=Hot badge/label.class=\"font-bold\" card.tap=\"open menu\" comment=\"x\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatUI(use); got != "use \"card\" badge/label=\"Hot\" badge/label.class=\"font-bold\" card.tap=\"open menu\" icon=\"home\" title=\"Latte\" title.class=\"text-rose-600\" comment=\"x\"\n" {
		t.Fatalf("formatted as\n%s", got)
	}
	inst := Instance(use[0], Part{UI: card})
	if got := FormatUI(inst); got != "box name=\"card\" class=\"p-4\" tap=\"open menu\"\n  icon \"house\" name=\"icon\" class=\"size-5\"\n  text \"Latte\" name=\"title\" class=\"font-bold text-rose-600\"\n"+
		"  use \"badge\" name=\"badge\" label=\"Hot\" label.class=\"font-bold\"\nend\n" {
		t.Fatalf("drawn as\n%s", got)
	}
	// The use's classes go to the element at the top of its component,
	// after its own, or to none when it has several.
	merged, _ := ParseUI("use card class=\"w-full p-2\"\n")
	if got := Instance(merged[0], Part{UI: card})[0].Attr("class"); got != "p-4 w-full p-2" {
		t.Fatalf("the use's classes go to %q", got)
	}
	if two := slices.Concat(card, badge); Instance(merged[0], Part{UI: two})[0].Attr("class") != "p-4" {
		t.Fatal("the use's classes went to one of several elements")
	}
	if FormatUI(card) != "box name=\"card\" class=\"p-4\"\n  icon \"coffee\" name=\"icon\" class=\"size-5\"\n  text \"Title\" name=\"title\" class=\"font-bold\"\n  use \"badge\" name=\"badge\"\nend\n" {
		t.Fatal("drawing a use changed its component")
	}
	if got := FormatUI(Instance(inst[0].Children[2], Part{UI: badge})); got != "text \"Hot\" name=\"label\" class=\"text-xs font-bold\"\n" {
		t.Fatalf("the badge drawn as\n%s", got)
	}
	if p := OverrideProblems(use[0], find); len(p) != 0 {
		t.Fatalf("problems %v", p)
	}
	bad, _ := ParseUI("use card nope=x title.src=y card=z icon=nope badge=gone title/label=x\nuse card badge/size=x\n")
	got := strings.Join(slices.Concat(OverrideProblems(bad[0], find), OverrideProblems(bad[1], find)), "\n")
	for _, want := range []string{"nope=\"x\" names nothing", "title.src=\"y\" sets the src of title, a text", "card=\"z\" sets the text of card, a box",
		"icon=\"nope\" unknown icon", "badge=\"gone\" swaps badge for components/gone.scenra", "title/label=\"x\" goes inside title, a text",
		"badge/size=\"x\" names nothing: no element of components/badge.scenra"} {
		if !strings.Contains(got, want) {
			t.Errorf("no problem %q in\n%s", want, got)
		}
	}
	for src, want := range map[string]string{
		"use card ti:tle=x":       "use takes no attribute \"ti:tle\"",
		"use card title.size=x":   "use cannot change \"size\" of title",
		"use card title.a.b=x":    "use cannot change \"a.b\"",
		"use card title.tap=jump": "tap takes",
		"use card src=x":          "use takes no attribute \"src\"",
		"text \"x\" name=class":   "name \"class\" is not a name",
		"text \"x\" name=\"a.b\"": "is not a name",
		"text \"x\" title=y":      "text takes no attribute \"title\"",
	} {
		if _, err := ParseUI(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
}

func TestSlots(t *testing.T) {
	card, err := ParseUI("box class=\"p-4\"\n  slot header\n    text \"Title\"\n  end\n  slot\n  end\n  use row\n    fill trailing\n      slot actions\n      end\n    end\n  end\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := Slots(card); !slices.Equal(got, []string{"header", "", "actions"}) {
		t.Fatalf("slots %v", got)
	}
	src := "use \"card\" class=\"m-2\"\n  fill \"header\" note=\"Varies\"\n    text \"Latte\"\n  end\n  fill\n    use \"card\"\n  end\nend\nuse \"card\"\n"
	use, err := ParseUI(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatUI(use); got != src {
		t.Fatalf("formatted as\n%s", got)
	}
	if len(use) != 2 || len(use[0].Children) != 2 || use[0].Children[1].Children[0].Kind != "use" {
		t.Fatalf("read %+v", use)
	}
	nested, _ := ParseUI("use a\n  fill\n    use b\n      fill\n        text x\n      end\n    end\n  end\nend\n")
	if inner := nested[0].Children[0].Children; len(inner) != 1 || len(inner[0].Children) != 1 {
		t.Fatalf("a use in a fill read as %+v", inner)
	}
	if got := Uses(use); !slices.Equal(got, []string{"card"}) {
		t.Fatalf("uses %v", got)
	}
	parts := map[string]Part{"card": {UI: card}}
	find := func(name string) (Part, bool) { p, ok := parts[name]; return p, ok }
	if p := FillProblems(use[0], find); len(p) != 0 {
		t.Fatalf("problems %v", p)
	}
	bad, _ := ParseUI("use card\n  fill footer\n  end\nend\n")
	if p := FillProblems(bad[0], find); len(p) != 1 || !strings.Contains(p[0], "the fill of slot footer fills nothing") || !strings.Contains(p[0], "it has slot header, the default slot, slot actions") {
		t.Fatalf("problems %v", p)
	}
	twice, _ := ParseUI("slot a\nend\nbox\n  slot a\n  end\nend\n")
	if p := SlotProblems(twice); len(p) != 1 || !strings.Contains(p[0], "slot a is written twice") {
		t.Fatalf("problems %v", p)
	}
	for src, want := range map[string]string{
		"fill x\nend":         "fill fills a slot of the component a use draws",
		"text a\nfill x\nend": "put it right after the use",
		"use a\n  fill x\n  end\nend\nfill y\nend":     "put it right after the use",
		"use a\n  fill x\n  end\n  text b\nend":        "a use holds only fill lines",
		"use a\n  fill x\n  end\n  fill x\n  end\nend": "the use fills slot x twice",
		"use a\n  fill\n  end\n  fill\n  end\nend":     "the use fills the default slot twice",
		"use a\n  fill x class=p-2\n  end\nend":        "fill takes no attribute \"class\"",
		"use a\n  fill x\n  end":                       "use has no end",
		"slot \"a b\"\nend":                            "is not a name",
	} {
		if _, err := ParseUI(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
	if _, err := Parse("design\nslot\nend\n"); err == nil || !strings.Contains(err.Error(), "a design has no slots") {
		t.Errorf("a design's slot: %v", err)
	}
	if _, err := Parse("component\nslot\nend\n"); err != nil {
		t.Errorf("a component's slot: %v", err)
	}
}

func TestImagesNameThePicturesOfTheAssetsFolder(t *testing.T) {
	for src, want := range map[string]string{"assets/a.png": "assets/a.png", "./assets/b/c.JPG": "assets/b/c.JPG", "https://x.io/a.png": "", "data:image/png;base64,AA": ""} {
		if got, _ := AssetPath(src); got != want {
			t.Errorf("AssetPath(%q) = %q, want %q", src, got, want)
		}
	}
	for p, want := range map[string]bool{"assets/a.png": true, "assets/b/c.JPG": true, "assets/a.mp4": false, "a/assets/a.png": false, "assets/../a.png": false} {
		if IsAsset(p) != want {
			t.Errorf("IsAsset(%q) is %v", p, !want)
		}
	}
	els, _ := ParseUI("image \"a\" src=assets/a.png\nbox\n  use card pic.src=assets/b.png\n  image \"a\" src=assets/a.png\nend\n")
	if got := Images(els); !slices.Equal(got, []string{"assets/a.png", "assets/b.png"}) {
		t.Fatalf("images %v", got)
	}
}

func TestOpensNamesFileAndDesign(t *testing.T) {
	for target, want := range map[string][2]string{"menu": {"menu", ""}, "auth#sign-up": {"auth", "sign-up"}, "#reset": {"", "reset"}} {
		if file, id := (Tap{TapOpen, target}).Opens(); file != want[0] || id != want[1] {
			t.Errorf("%s opens %q %q", target, file, id)
		}
	}
}

const designs = `design sign-in name="Sign in" gap=120
text "In" tap="open #sign-up"

design sign-up name="Sign up" device=desktop
box
  text "Up" comment="Shorter"
end
`

func TestSeveralDesigns(t *testing.T) {
	f, err := Parse(designs)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Screens) != 2 || f.Gap != 120 || f.Spacing() != 120 || f.Title() != "Sign in" || !f.Several() {
		t.Fatalf("designs: %+v", f)
	}
	if i, ok := f.Screen("sign-up"); !ok || i != 1 || f.Screens[1].Device != "desktop" || len(f.Screens[1].UI[0].Children) != 1 {
		t.Fatalf("sign-up: %d %v %+v", i, ok, f.Screens[1])
	}
	if i, ok := f.Screen(""); !ok || i != 0 {
		t.Fatal("no first design")
	}
	if _, ok := f.Screen("menu"); ok {
		t.Fatal("found a missing design")
	}
	if w, h := f.Size(1); w != 1200 || h != 760 {
		t.Fatalf("sign-up is %d×%d", w, h)
	}
	if got := f.Comments(); !slices.Equal(got, []Comment{{1, "0.0", "Shorter"}}) {
		t.Fatalf("comments %v", got)
	}
	if f.Format() != strings.ReplaceAll(designs, `text "Up"`, `text "Up"`) {
		t.Fatalf("format:\n%s", f.Format())
	}
	if (File{}).Spacing() != DefaultGap {
		t.Fatal("no default gap")
	}
}

func TestSeveralFlows(t *testing.T) {
	src := "flow pay name=\"Pay\" direction=right\n\nnode a \"A\"\n\na -> b\n\nflow refund\n\na -> c\n"
	f, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Flows) != 2 || f.Flows[0].Direction != Right || f.Flows[1].ID != "refund" || len(f.Flows[1].Nodes) != 2 || f.Flows[1].Nodes[0].Label != "a" {
		t.Fatalf("flows: %+v", f.Flows)
	}
	again, err := Parse(f.Format())
	if err != nil || again.Format() != f.Format() || !strings.Contains(f.Format(), "a -> b\n\nflow refund\n") {
		t.Fatalf("format: %v\n%s", err, f.Format())
	}
	clone := f.Clone()
	clone.Flows[0].Nodes[0].Label = "Z"
	if f.Flows[0].Nodes[0].Label != "A" {
		t.Fatal("a clone shares its flows")
	}
}
