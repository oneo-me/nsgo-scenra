package scenraview

import (
	"bufio"
	"bytes"
	"context"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// change is how many lines a file of the open folder added and removed
// since the last commit of the Git repository holding the folder.
type change struct{ added, removed int }

// changes are the changes of the open folder's Scenra files, by their
// path in the folder with slashes; nil when the folder is in no
// repository, or Git is not installed.
type changes map[string]change

func sameChanges(a, b changes) bool { return (a == nil) == (b == nil) && maps.Equal(a, b) }

// gitTimeout bounds a run of Git, so a stuck repository never holds up
// the scan.
const gitTimeout = 5 * time.Second

// emptyTree is Git's empty tree, which a repository without a commit
// yet compares with.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// git runs Git in dir with args, never taking the locks a command the
// user runs at the same time needs.
func git(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-c", "core.quotePath=false"}, args...)...)
	cmd.Dir = dir
	hideConsole(cmd)
	return cmd.Output()
}

// gitChanges are the changes of the Scenra files under dir, against the
// last commit of the repository holding it, or nil when none does. A file
// Git does not track yet adds all its lines.
func gitChanges(dir string) changes {
	if _, err := git(dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil
	}
	base := "HEAD"
	if _, err := git(dir, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		base = emptyTree
	}
	out := changes{}
	// --relative names the files from dir and leaves out the others.
	stat, err := git(dir, "diff", "--numstat", "--relative", "--no-renames", base, "--", "*.scenra")
	if err != nil {
		return nil
	}
	for line := range bytes.Lines(stat) {
		fields := strings.SplitN(strings.TrimRight(string(line), "\r\n"), "\t", 3)
		if len(fields) != 3 {
			continue
		}
		added, _ := strconv.Atoi(fields[0])
		removed, _ := strconv.Atoi(fields[1])
		out[filepath.ToSlash(fields[2])] = change{added, removed}
	}
	untracked, err := git(dir, "ls-files", "--others", "--exclude-standard", "--", "*.scenra")
	if err != nil {
		return out
	}
	sc := bufio.NewScanner(bytes.NewReader(untracked))
	for sc.Scan() {
		rel := filepath.ToSlash(sc.Text())
		if rel == "" {
			continue
		}
		out[rel] = change{added: countLines(filepath.Join(dir, filepath.FromSlash(rel)))}
	}
	return out
}

// countLines is how many lines the file at path holds.
func countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return 0
	}
	n := bytes.Count(data, []byte{'\n'})
	if data[len(data)-1] != '\n' {
		n++
	}
	return n
}
