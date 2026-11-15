package main

import (
	"os"
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
	words = append(words, "`"+string(flowfile.Component)+"`", "`"+flowfile.ComponentsDir+"/`")
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
