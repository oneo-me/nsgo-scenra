package flowfile

import (
	"strings"
	"testing"
)

func TestParseGlass(t *testing.T) {
	good := []struct {
		in   string
		want Glass
		str  string
	}{
		{"regular", Glass{Style: GlassRegular}, "regular"},
		{"clear", Glass{Style: GlassClear}, "clear"},
		{"none", Glass{Style: GlassNone}, "none"},
		{"regular interactive", Glass{Style: GlassRegular, Interactive: true}, "regular interactive"},
		{"interactive clear", Glass{Style: GlassClear, Interactive: true}, "clear interactive"},
	}
	for _, tc := range good {
		got, err := ParseGlass(tc.in)
		if err != nil {
			t.Fatalf("ParseGlass(%q): %v", tc.in, err)
		}
		if got != tc.want || got.String() != tc.str {
			t.Fatalf("ParseGlass(%q) = %+v (%q), want %+v (%q)", tc.in, got, got.String(), tc.want, tc.str)
		}
	}
	bad := []string{"", "frosted", "interactive", "none interactive", "regular clear", "regular interactive interactive"}
	for _, in := range bad {
		if got, err := ParseGlass(in); err == nil {
			t.Fatalf("ParseGlass(%q) = %+v, want error", in, got)
		}
	}
}

func TestGlassParseFormatAndOverrides(t *testing.T) {
	src := `design
box class="rounded-full bg-white/60 backdrop-blur-lg" glass="regular"
  box class="rounded-full" glass="clear interactive"
    text "Play"
  end
  use "pill" glass="regular interactive"
end
`
	f, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := f.Format(); got != src {
		t.Fatalf("Format:\n%s\nwant:\n%s", got, src)
	}

	// Non-box elements reject glass=.
	for _, line := range []string{
		`text "Hi" glass=regular`,
		`icon star glass=regular`,
		`image "Pic" glass=regular`,
		`svg "<svg/>" glass=regular`,
	} {
		if _, err := Parse("design\n" + line + "\n"); err == nil {
			t.Fatalf("Parse accepted %s", line)
		}
	}

	// Component variants and overrides can set or clear glass=.
	compSrc := `component
box name="bar" class="rounded-full" glass="regular"
  slot "body" glass="clear"
    text "Item" name="label"
  end
end
variant flat glass="none"
variant clear bar.glass="clear interactive" body.glass="none"
`
	comp, err := Parse(compSrc)
	if err != nil {
		t.Fatalf("Parse component: %v", err)
	}
	if got := comp.Format(); got != compSrc {
		t.Fatalf("Format component:\n%s\nwant:\n%s", got, compSrc)
	}
	part := Part{UI: comp.Screens[0].UI, Variants: comp.Screens[0].Variants}
	parts := func(name string) (Part, bool) {
		if name == "pill" {
			return part, true
		}
		return Part{}, false
	}

	if probs := VariantProblems(comp.Screens[0], "pill", parts); len(probs) > 0 {
		t.Fatalf("VariantProblems: %v", probs)
	}

	// variant=flat overrides root glass=regular with none.
	useFlat, err := ParseUI("use pill variant=flat\n")
	if err != nil {
		t.Fatal(err)
	}
	drawn := Instance(useFlat[0], part)
	if got := drawn[0].Attr("glass"); got != "none" {
		t.Fatalf("flat root glass = %q, want none", got)
	}

	// variant=clear sets bar.glass and body.glass on the slot.
	useClear, err := ParseUI("use pill variant=clear\n")
	if err != nil {
		t.Fatal(err)
	}
	drawn = Instance(useClear[0], part)
	if got := drawn[0].Attr("glass"); got != "clear interactive" {
		t.Fatalf("clear bar glass = %q, want clear interactive", got)
	}
	if got := drawn[0].Children[0].Attr("glass"); got != "none" {
		t.Fatalf("clear slot glass = %q, want none", got)
	}

	// Overriding glass on a text element reports a problem.
	useBad, err := ParseUI("use pill label.glass=regular\n")
	if err != nil {
		t.Fatal(err)
	}
	probs := OverrideProblems(useBad[0], parts)
	if len(probs) == 0 || !strings.Contains(probs[0], "only boxes take") {
		t.Fatalf("OverrideProblems(label.glass) = %v, want box-only error", probs)
	}
}
