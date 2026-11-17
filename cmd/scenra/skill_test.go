package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/oneo-me/nsgo-scenra/flowfile"
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
	words = append(words, "name=", "<a>/<b>=", "`width=fill`", "`preview=`", `backdrop="…"`, "`>`", "`variant <id>`", "`variant=`", "`state=`", "`in=\"<file>#<outlet>\"`", "`get_layout`", `\"`)
	words = append(words, flowfile.States...)
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

// The skill's references ship beside it, so each it links must be there.
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
