package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview"
)

// The skill is written by hand, so it must keep naming what the format
// knows.
func TestTheSkillTeachesTheWholeFormat(t *testing.T) {
	data, err := os.ReadFile("resources/skills/scenra/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	skill := string(data)
	if !strings.HasPrefix(skill, "---\nname: scenra\n") {
		t.Fatal("the skill does not name itself scenra")
	}
	var words []string
	for _, spec := range flowfile.Elements {
		words = append(words, "`"+spec.Kind+"`")
		for _, a := range spec.Attrs {
			words = append(words, a+`="…"`)
		}
	}
	words = append(words, "`"+string(flowfile.Component)+"`", "`"+flowfile.ComponentsDir+"/`", "`flows/`", "`views/`", "`"+flowfile.AssetsDir+"/`")
	words = append(words, "name=", "<a>/<b>=", "`width=fill`", "`preview=`", `backdrop="…"`, "`>`", "`variant <id>`", "`variant=`", "`state=`", "`in=\"<file>#<outlet>\"`", "`design device=desktop class=", `\"`)
	words = append(words, flowfile.States...)
	words = append(words, flowfile.GlassStyles...)
	words = append(words, flowfile.GlassInteractive)
	words = append(words, flowfile.ReservedNames()...)
	for _, a := range flowfile.OverrideAttrs {
		words = append(words, "`<name>."+a+`="…"`+"`")
	}
	words = append(words, `comment="…"`, `note="…"`, `tap="…"`, "panel=", "selected:", "group-selected:", "gap=", `tap="open #`)
	for _, a := range flowfile.Taps {
		words = append(words, `tap="`+a)
	}
	words = append(words, flowfile.Kinds...)
	words = append(words, flowfile.Statuses...)
	words = append(words, flowfile.Devices...)
	words = append(words, flowfile.CommonIcons...)
	for _, w := range words {
		if !strings.Contains(skill, w) {
			t.Errorf("the skill does not mention %s", w)
		}
	}
}

// The app's skill must name the server's tools and what they take.
func TestTheAppSkillTeachesTheServer(t *testing.T) {
	data, err := os.ReadFile("resources/skills/scenra-app/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	skill := string(data)
	if !strings.HasPrefix(skill, "---\nname: scenra-app\n") {
		t.Fatal("the skill does not name itself scenra-app")
	}
	for _, w := range []string{"`" + scenraview.ServerName + "`", "`" + scenraview.ToolView + "`", "`" + scenraview.ToolLayout + "`", "`tab`",
		`notes="` + scenraview.NotesOpen + `"`, `notes="` + scenraview.NotesNone + `"`, "`element=`", "`depth=`"} {
		if !strings.Contains(skill, w) {
			t.Errorf("the skill does not mention %s", w)
		}
	}
	// The format's skill stays the same wherever the files are edited.
	format, _ := os.ReadFile("resources/skills/scenra/SKILL.md")
	if strings.Contains(string(format), scenraview.ToolView) {
		t.Error("the format's skill names the app's tools")
	}
}

// The skills' references ship beside them, so each one links must be
// there.
func TestTheSkillLinksItsReferences(t *testing.T) {
	data, err := os.ReadFile("resources/skills/scenra/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	links := regexp.MustCompile(`\]\((references/[^)]+)\)`).FindAllStringSubmatch(string(data), -1)
	if len(links) == 0 {
		t.Fatal("the skill links no references")
	}
	for _, l := range links {
		if _, err := os.Stat(filepath.Join("resources/skills/scenra", l[1])); err != nil {
			t.Errorf("the skill links %s, which is not there", l[1])
		}
	}
}
