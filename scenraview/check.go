package scenraview

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Check reports the problems of the Scenra files of the folder at path,
// or of the file at path, as the editor's Server does, without an editor
// or what the user sees there, for hosts whose agents edit the files
// elsewhere. A file's whole project counts, though only its own problems
// are told. It leaves out how the designs lay out, as NSGo lays out only
// in a window. It reads the files on the goroutine calling it.
func Check(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	dir := path
	if !info.IsDir() {
		dir = ProjectOf(path)
	}
	root := projectDir(dir)
	found, parts, pics, _, _ := scan(root, nil)
	var files []fileView
	for _, en := range found {
		files = append(files, fileViewOf(root, en))
	}
	keep := func(fileView) bool { return true }
	var b strings.Builder
	if info.IsDir() {
		if root == path {
			fmt.Fprintf(&b, "These are the problems of the Scenra files of %s; the paths below are relative to it.\n", path)
		} else {
			fmt.Fprintf(&b, "These are the problems of the Scenra files of %s, whose project is in %s, as it has none of its own; the paths below are relative to %[2]s.\n", path, root)
		}
		if len(files) == 0 {
			b.WriteString("It has no Scenra files.\n")
			return b.String(), nil
		}
	} else {
		rel, err := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if err != nil || !slices.ContainsFunc(files, func(f fileView) bool { return f.path == rel }) {
			return "", fmt.Errorf("%s is no Scenra file of the project in %s", path, root)
		}
		keep = func(f fileView) bool { return f.path == rel }
		fmt.Fprintf(&b, "%s is %s of the project in %s; the paths below are relative to it.\n", path, rel, root)
	}
	if p := problems(files, parts, pics, keep); p != "" {
		b.WriteString(p)
	} else {
		b.WriteString("No problems found.\n")
	}
	return b.String(), nil
}
