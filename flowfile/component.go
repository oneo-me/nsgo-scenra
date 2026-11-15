package flowfile

import (
	"path"
	"slices"
	"strings"
)

// ComponentsDir is the folder of a project's components, at its top: the
// component tab-bar is components/tab-bar.scenra, and nav/tab-bar is
// components/nav/tab-bar.scenra.
const ComponentsDir = "components"

// ValidComponentName reports whether name can name a component: a path
// of the components folder, its parts IDs, without .scenra.
func ValidComponentName(name string) bool {
	for _, part := range strings.Split(name, "/") {
		if !ValidID(part) || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// ComponentPath is the path of the component named name, relative to the
// project's folder, with slashes.
func ComponentPath(name string) string { return ComponentsDir + "/" + name + ".scenra" }

// ComponentName is the name of the component at the path, relative to
// the project's folder with slashes, when the path is in the components
// folder.
func ComponentName(p string) (string, bool) {
	rest, ok := strings.CutPrefix(path.Clean(p), ComponentsDir+"/")
	name, scenra := strings.CutSuffix(rest, ".scenra")
	if !ok || !scenra || !ValidComponentName(name) {
		return "", false
	}
	return name, true
}

// Uses are the names of the components els use, each once, in the order
// they are first written.
func Uses(els []Element) []string {
	var out []string
	var walk func(els []Element)
	walk = func(els []Element) {
		for _, el := range els {
			if el.Kind == "use" && !slices.Contains(out, el.Text) {
				out = append(out, el.Text)
			}
			walk(el.Children)
		}
	}
	walk(els)
	return out
}
