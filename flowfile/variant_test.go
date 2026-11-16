package flowfile

import (
	"slices"
	"strings"
	"testing"
)

// flowStep is a component of the user's kind: a row whose variants
// change its icon and texts.
const flowStep = `component width=fill preview=244 backdrop="bg-background > bg-foreground/5"
box name=row class="flex-row gap-2"
  icon circle-check name=icon class="size-3.5 text-emerald-600"
  text "Done" name=title class="text-sm"
end
variant doing icon=loader-circle icon.class="text-accent" title.class="font-medium" row.class="bg-accent/10"
variant next icon=circle-dashed icon.class="text-accent"
variant late variant=next icon.class="text-rose-600"
`

func partOf(t *testing.T, src string) Part {
	t.Helper()
	f, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return Part{UI: f.Screens[0].UI, Variants: f.Screens[0].Variants}
}

func TestUsesPickVariantsByName(t *testing.T) {
	step := partOf(t, flowStep)
	parts := func(name string) (Part, bool) { return step, name == "flow-step" }
	use := func(src string) Element {
		t.Helper()
		els, err := ParseUI(src)
		if err != nil {
			t.Fatal(err)
		}
		return els[0]
	}
	// The variants apply in order, then the use's own changes.
	drawn := Instance(use(`use flow-step variant="next doing" title="Open the PR" icon.class="size-4"`), step)
	if got := FormatUI(drawn); got != "box name=\"row\" class=\"flex-row gap-2 bg-accent/10\"\n"+
		"  icon \"loader-circle\" name=\"icon\" class=\"size-3.5 text-emerald-600 text-accent text-accent size-4\"\n"+
		"  text \"Open the PR\" name=\"title\" class=\"text-sm font-medium\"\nend\n" {
		t.Fatalf("drawn as\n%s", got)
	}
	// A variant starts from those it picks.
	if got := Instance(use(`use flow-step variant=late`), step)[0].Children[0]; got.Text != "circle-dashed" || !strings.HasSuffix(got.Attr("class"), "text-accent text-rose-600") {
		t.Fatalf("late drawn as %+v", got)
	}
	if p := OverrideProblems(use(`use flow-step variant="next nxt"`), parts); len(p) != 1 || !strings.Contains(p[0], `use flow-step: no variant "nxt"; it has doing, next, late`) {
		t.Fatalf("problems %v", p)
	}
	// A use writing out a variant's changes hears it could pick it.
	if h := VariantHints(use(`use flow-step title="x" icon=circle-dashed icon.class="text-accent"`), step); len(h) != 1 || !strings.Contains(h[0], "is variant next; write variant=next instead") {
		t.Fatalf("hints %v", h)
	}
	if h := VariantHints(use(`use flow-step variant=next`), step); len(h) != 0 {
		t.Fatalf("hints %v", h)
	}
	// A use passes variants down by names.
	card := partOf(t, "component\nbox\n  use flow-step name=step\nend\n")
	nested := Instance(use(`use card step.variant=next`), card)
	if got := FormatUI(Instance(nested[0].Children[0], step)); !strings.Contains(got, "circle-dashed") {
		t.Fatalf("the nested step drawn as\n%s", got)
	}
	both := func(name string) (Part, bool) {
		if name == "card" {
			return card, true
		}
		return parts(name)
	}
	if p := OverrideProblems(use(`use card step.variant=nope`), both); len(p) != 1 || !strings.Contains(p[0], `no variant "nope"`) {
		t.Fatalf("problems %v", p)
	}
}

func TestVariantsPickingEachOtherInALoop(t *testing.T) {
	f, err := Parse("component\nbox name=a\nend\nvariant x variant=y\nvariant y variant=x class=\"p-1\"\nvariant z variant=gone\n")
	if err != nil {
		t.Fatal(err)
	}
	p := strings.Join(VariantProblems(f.Screens[0], "c", func(string) (Part, bool) { return Part{}, false }), "\n")
	for _, want := range []string{"variant x: variants pick each other in a loop, x picks y picks x", `variant z: no variant "gone"`} {
		if !strings.Contains(p, want) {
			t.Errorf("no problem %q in\n%s", want, p)
		}
	}
	// Drawing one still ends.
	Instance(Element{Kind: "use", Attrs: map[string]string{"variant": "x"}}, Part{UI: f.Screens[0].UI, Variants: f.Screens[0].Variants})
}

func TestStatesApplyTheirClasses(t *testing.T) {
	tab := partOf(t, `component
box name=tab class="group h-6 rounded-md px-2 selected:bg-foreground/10 hover:bg-muted"
  text "Pipeline" name=label class="text-muted-foreground group-selected:font-semibold group-selected:text-foreground"
  use dot name=dot
end
variant selected state=selected
`)
	dot := partOf(t, "component\nbox class=\"size-1 group-selected:bg-accent dark:selected:p-1\"\nend\n")
	parts := func(name string) (Part, bool) {
		switch name {
		case "tab":
			return tab, true
		case "dot":
			return dot, true
		}
		return Part{}, false
	}
	drawn := Instance(Element{Kind: "use", Text: "tab", Attrs: map[string]string{"variant": "selected"}}, tab)
	if got := drawn[0].Attr("class"); !strings.HasSuffix(got, " bg-foreground/10") || strings.Contains(got, " bg-muted") {
		t.Fatalf("the selected tab's classes %q", got)
	}
	if got := drawn[0].Children[0].Attr("class"); !strings.HasSuffix(got, " font-semibold text-foreground") {
		t.Fatalf("the selected label's classes %q", got)
	}
	// A use inside a group follows its group-*: classes, not its own state.
	inner := Instance(drawn[0].Children[1], dot)
	if got := inner[0].Attr("class"); !strings.HasSuffix(got, " bg-accent") || strings.Contains(got, " dark:p-1") {
		t.Fatalf("the dot's classes %q", got)
	}
	// One element by name, and states with nothing to draw.
	if got := Instance(Element{Kind: "use", Text: "tab", Attrs: map[string]string{"state": "hover@tab"}}, tab)[0].Attr("class"); !strings.HasSuffix(got, " bg-muted") {
		t.Fatalf("the hovered tab's classes %q", got)
	}
	p := OverrideProblems(Element{Kind: "use", Text: "tab", Attrs: map[string]string{"state": "disabled selected@nobody"}}, parts)
	if all := strings.Join(p, "\n"); len(p) != 2 || !strings.Contains(all, "state=selected@nobody names nothing") || !strings.Contains(all, "state=disabled draws nothing") {
		t.Fatalf("problems %v", p)
	}
	for src, want := range map[string]string{
		"use tab state=sleepy":    `unknown state "sleepy"`,
		"use tab state=selected@": "names no element",
		"use tab variant=\"\"":    "variant= takes the IDs",
	} {
		if _, err := ParseUI(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
	if _, err := ParseUI("use tab state=\"selected hover@label\" variant=selected tab.state=pressed"); err != nil {
		t.Fatal(err)
	}
}

func TestOutletsAndHosts(t *testing.T) {
	f, err := Parse("design repos in=\"main#content\" name=\"Repos\"\nbox\nend\n\ndesign empty\nbox\nend\n\ndesign other in=\"settings#general/content\"\nbox outlet=side\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []Host{{"main", "", "content"}, {"main", "", "content"}, {"settings", "general", "content"}} {
		if h, ok := f.HostOf(i); !ok || h != want {
			t.Errorf("design %d fills %+v", i, h)
		}
	}
	if !strings.Contains(f.Format(), "design repos name=\"Repos\" in=\"main#content\"\n") {
		t.Fatalf("formatted as\n%s", f.Format())
	}
	if h, err := ParseHost("#general/content"); err != nil || h != (Host{"", "general", "content"}) {
		t.Fatalf("read %+v %v", h, err)
	}
	for _, bad := range []string{"main", "main#", "main#a/b/c", "#"} {
		if _, err := ParseHost(bad); err == nil {
			t.Errorf("%q read", bad)
		}
	}
	main, _ := ParseUI("box\n  box outlet=content\n  end\n  box outlet=content\n  end\nend\n")
	if at, ok := Outlet(main, "content"); !ok || !slices.Equal(at, []int{0, 0}) {
		t.Fatalf("outlet at %v", at)
	}
	if p := OutletProblems(main); len(p) != 1 || !strings.Contains(p[0], "outlet content is written twice") {
		t.Fatalf("problems %v", p)
	}
	for src, want := range map[string]string{
		"design in=main":             "names no outlet",
		"component in=\"main#c\"":    "unknown attribute \"in\"",
		"design\ntext x outlet=a":    "text takes no attribute \"outlet\"",
		"design\nbox outlet=\"a.b\"": "is not a name",
	} {
		if _, err := Parse(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
	// A design shown in another file's outlet keeps opening its own.
	els, _ := ParseUI("text x tap=\"open #b\"\nuse row item.tap=\"open #c\"\ntext y tap=\"open menu\"\n")
	got := InFile(els, "issues")
	if got[0].Attr("tap") != "open issues#b" || got[1].Attr("item.tap") != "open issues#c" || got[2].Attr("tap") != "open menu" || els[0].Attr("tap") != "open #b" {
		t.Fatalf("taps %+v", got)
	}
}

func TestPreviewsAndBackdrops(t *testing.T) {
	f, err := Parse(flowStep)
	if err != nil {
		t.Fatal(err)
	}
	if w, _ := f.Size(0); w != 244 {
		t.Fatalf("previews %d wide", w)
	}
	if !strings.HasPrefix(f.Format(), "component width=fill preview=244 backdrop=") {
		t.Fatalf("formatted as\n%s", f.Format())
	}
	if got := f.Screens[0].Layers(); !slices.Equal(got, []string{"bg-background", "bg-foreground/5"}) {
		t.Fatalf("layers %q", got)
	}
	if got := (Screen{Backdrop: "bg-inset"}).Layers(); !slices.Equal(got, []string{"bg-background", "bg-inset"}) {
		t.Fatalf("one layer is %q", got)
	}
	if got := (Screen{}).Layers(); got != nil {
		t.Fatalf("no backdrop is %q", got)
	}
	if _, err := Parse("component preview=200"); err == nil || !strings.Contains(err.Error(), "add width=fill") {
		t.Fatalf("preview without fill: %v", err)
	}
}

func TestUsesNoteAndSetPanels(t *testing.T) {
	f, err := Parse("design\nbox\n  use row row.note=\"Opens the task\" title=x\n  use card option.panel=a\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := f.UseNotes(); !slices.Equal(got, []UseNote{{0, "0.0", "row", "Opens the task"}}) {
		t.Fatalf("notes %v", got)
	}
	card := Part{UI: []Element{{Kind: "box", Attrs: map[string]string{"name": "option"}}}}
	if got := Instance(f.Screens[0].UI[0].Children[1], card)[0].Attr("panel"); got != "a" {
		t.Fatalf("panel %q", got)
	}
	if _, err := ParseUI("text x name=variant"); err == nil || !strings.Contains(err.Error(), "name, class, tap, panel, comment, note, variant, state, outlet, src") {
		t.Fatalf("a reserved name: %v", err)
	}
}

// settingsItem is a component of the user's kind: a row whose control
// is a slot its stacked variant widens by the slot's own name.
const settingsItem = `component width=fill preview=535
box name=row class="flex-row items-center gap-3 px-3"
  text "Label" name=label class="flex-1"
  slot control class="flex-row items-center"
    text "Pick"
  end
  slot name=extra class="gap-1"
  end
end
variant stacked row.class="flex-col items-stretch" control.class="w-full"
`

func TestUsesShapeSlotsByTheirNames(t *testing.T) {
	item := partOf(t, settingsItem)
	parts := func(name string) (Part, bool) { return item, name == "settings-item" }
	use := func(src string) Element {
		t.Helper()
		els, err := ParseUI(src)
		if err != nil {
			t.Fatal(err)
		}
		return els[0]
	}
	if p := VariantProblems(Screen{UI: item.UI, Variants: item.Variants}, "settings-item", parts); len(p) != 0 {
		t.Fatalf("problems %v", p)
	}
	// The component's classes, the variant's and the use's add up in
	// order; the slot keeps its name and what it holds.
	drawn := Instance(use(`use settings-item variant=stacked control.class="justify-end" extra.class="p-2"`), item)
	slot := drawn[0].Children[1]
	if slot.Kind != "slot" || slot.Text != "control" || slot.Attr("class") != "flex-row items-center w-full justify-end" || slot.Children[0].Text != "Pick" {
		t.Fatalf("the slot draws as %+v", slot)
	}
	if got := drawn[0].Children[2].Attr("class"); got != "gap-1 p-2" {
		t.Fatalf("the slot named extra takes %q", got)
	}
	// A slot takes nothing else.
	p := OverrideProblems(use(`use settings-item control="x" control.tap="close" control/label="y"`), parts)
	all := strings.Join(p, "\n")
	for _, want := range []string{`control="x" sets the text of control, a slot, which has none`,
		`control.tap="close" sets the tap of control, a slot, which takes only class`,
		`control/label="y" goes inside control, a slot of components/settings-item.scenra, not a use`} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in\n%s", want, all)
		}
	}
	if drawn := Instance(use(`use settings-item control="x" control.tap="close"`), item); drawn[0].Children[1].Text != "control" || drawn[0].Children[1].Attr("tap") != "" {
		t.Fatalf("the slot changed: %+v", drawn[0].Children[1])
	}
}
