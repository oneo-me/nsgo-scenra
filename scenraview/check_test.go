package scenraview

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckTellsTheProblemsWithoutAnEditor(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra":           "design\nbox\n  text \"Go\" tap=\"open nowhere\" class=\"text-glow\"\n  use card\n  image src=\"assets/logo.png\"\nend\n",
		"settings.scenra":       "design\nbox\n  text \"Back\" tap=\"open gone\"\nend\n",
		"flows/broken.scenra":   "flow\nnode a \"A\nnode b\n",
		"misc/stray.scenra":     "design\ntext \"Stray\"\n",
		"components/tab.scenra": "component\ntext \"Tab\" comment=\"Make it larger\"\n",
	})
	all, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`home.scenra: tap "open nowhere": there is no design nowhere`,
		`settings.scenra: tap "open gone"`,
		"flows/broken.scenra: line ",
		"- misc/stray.scenra\n",
		`home.scenra: use "card": there is no components/card.scenra`,
		`home.scenra: src "assets/logo.png": there is no assets/logo.png`,
		"home.scenra: text-glow",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in:\n%s", want, all)
		}
	}
	// What the user sees in an editor is not told.
	if strings.Contains(all, "Make it larger") || strings.Contains(all, "The user") {
		t.Errorf("the check tells of the editor:\n%s", all)
	}
	// A file tells only its own problems, its project's files counting.
	one, err := Check(filepath.Join(dir, "settings.scenra"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(one, `settings.scenra: tap "open gone"`) || strings.Contains(one, "home.scenra") || strings.Contains(one, "broken") {
		t.Errorf("the file's check:\n%s", one)
	}
	clean, err := Check(filepath.Join(dir, "components", "tab.scenra"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(clean, "components/tab.scenra of the project in "+dir) || !strings.HasSuffix(clean, "No problems found.\n") {
		t.Errorf("the component's check:\n%s", clean)
	}
	// A folder without a project of its own checks its design folder's.
	repo := folderOfFiles(t, map[string]string{"design/home.scenra": "design\ntext \"Home\"\n"})
	if got, err := Check(repo); err != nil || !strings.Contains(got, "project is in "+filepath.Join(repo, "design")) || !strings.HasSuffix(got, "No problems found.\n") {
		t.Errorf("the repository's check: %v\n%s", err, got)
	}
	if _, err := Check(filepath.Join(dir, "missing.scenra")); err == nil {
		t.Error("a missing file checked")
	}
}
