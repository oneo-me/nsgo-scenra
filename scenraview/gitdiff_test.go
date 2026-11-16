package scenraview

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// gitIn runs Git in dir, skipping the test when Git is not installed.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	cmd := exec.Command("git", append([]string{"-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestGitTellsTheLinesFilesChanged(t *testing.T) {
	root := folderOfFiles(t, map[string]string{
		"app/home.scenra":  "design\ntext \"Welcome\"\ntext \"Hello\"\n",
		"app/plan.scenra":  "flow\nnode a \"A\"\n",
		"app/notes.txt":    "one\n",
		"other/out.scenra": "flow\n",
	})
	if got := gitChanges(filepath.Join(root, "app")); got != nil {
		t.Fatalf("a folder in no repository has changes %v", got)
	}
	gitIn(t, root, "init", "-q")
	dir := filepath.Join(root, "app")
	// Before the first commit, every file adds its lines.
	if got := gitChanges(dir); got["home.scenra"] != (change{added: 3}) || len(got) != 2 {
		t.Fatalf("before a commit the changes are %v", got)
	}
	gitIn(t, root, "add", ".")
	gitIn(t, root, "commit", "-q", "-m", "first")
	if got := gitChanges(dir); got == nil || len(got) != 0 {
		t.Fatalf("after a commit the changes are %v", got)
	}
	os.WriteFile(filepath.Join(dir, "home.scenra"), []byte("design\ntext \"Welcome back\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("two\n"), 0o644)
	os.WriteFile(filepath.Join(root, "other", "out.scenra"), []byte("design\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "screens"), 0o755)
	os.WriteFile(filepath.Join(dir, "screens", "new.scenra"), []byte("design\ntext \"New\"\n"), 0o644)
	got := gitChanges(dir)
	want := changes{"home.scenra": {added: 1, removed: 2}, "screens/new.scenra": {added: 2}}
	if !sameChanges(got, want) {
		t.Fatalf("the changes are %v, not %v", got, want)
	}
}

func TestTheTreeShowsTheLinesFilesChanged(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"home.scenra": "design\ntext \"Welcome\"\n",
		"plan.scenra": "flow\n",
	})
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "add", "plan.scenra")
	gitIn(t, dir, "commit", "-q", "-m", "first")
	app, e := startEditor(t)
	openPath(t, app, e, dir)
	app.WaitFor("id:files >> id:change-added", 5*time.Second)
	if got := app.Find("id:files >> id:change-added").Text; got != "+2" {
		t.Fatalf("the tree shows %q added", got)
	}
	if app.Exists("id:files >> id:change-removed") {
		t.Fatal("the tree shows lines removed")
	}
}

// A name too long for the tree is cut short before the lines it changed.
func TestTheLinesChangedStayBesideALongName(t *testing.T) {
	dir := folderOfFiles(t, map[string]string{
		"a design whose name is far too long for the sidebar to show whole.scenra": "design\ntext \"Welcome\"\n",
	})
	gitIn(t, dir, "init", "-q")
	app, e := startEditor(t)
	openPath(t, app, e, dir)
	app.WaitFor("id:files >> id:change-added", 5*time.Second)
	v := app.Find("id:files >> id:change-added")
	right := v.Frame.Origin.X + v.Frame.Size.Width
	for v = v.Parent; v.Props.TestID != "file"; v = v.Parent {
		right += v.Frame.Origin.X
	}
	if right > v.Frame.Size.Width {
		t.Fatalf("the lines added end at %v, past the row's %v", right, v.Frame.Size.Width)
	}
}
