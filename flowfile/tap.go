package flowfile

import (
	"errors"
	"fmt"
	"strings"
)

// What pressing an element does, its tap attribute, as the editor plays
// a design: open another design, close the one shown, returning to the
// design that opened it, or show a tab, the panels of that name.
const (
	TapOpen  = "open"
	TapClose = "close"
	TapTab   = "tab"
)

// Taps are the actions a tap takes.
var Taps = []string{TapOpen, TapClose, TapTab}

// Tap is an element's tap: its action, and the design it opens, by its
// file's path in the project's folder without .scenra, a # and its ID
// when the file holds several, or # and the ID alone for a design of the
// same file, or the tab it shows.
type Tap struct{ Action, Target string }

// Opens is the file and the ID of the design an open tap opens: "" for
// the file of the design tapped, and for the file's first design.
func (t Tap) Opens() (file, id string) {
	file, id, _ = strings.Cut(t.Target, "#")
	return file, id
}

// String writes t as a tap attribute.
func (t Tap) String() string {
	if t.Target == "" {
		return t.Action
	}
	return t.Action + " " + t.Target
}

var errTap = errors.New(`tap takes "open <design>", close or "tab <panel>"`)

// ParseTap reads a tap attribute: open with a design's path, close, or
// tab with a panel's name.
func ParseTap(s string) (Tap, error) {
	action, target, _ := strings.Cut(strings.TrimSpace(s), " ")
	t := Tap{action, strings.TrimSpace(target)}
	switch t.Action {
	case TapClose:
		if t.Target != "" {
			return Tap{}, errTap
		}
	case TapOpen:
		file, id, hash := strings.Cut(t.Target, "#")
		if file == "" && !hash || file != "" && !ValidDesignName(file) || hash && !ValidID(id) {
			return Tap{}, fmt.Errorf("tap %q names no design; open one by its path in the folder without .scenra, such as menu or orders/detail, and #<id> for one of several in a file, such as auth#sign-up or #sign-up in the same file", s)
		}
	case TapTab:
		if !ValidID(t.Target) {
			return Tap{}, fmt.Errorf("tap %q names no panel; show one by its name, of letters, digits, _, - or .", s)
		}
	default:
		return Tap{}, errTap
	}
	return t, nil
}

// Tap is what pressing the element does, none when it has no tap.
func (el Element) Tap() (Tap, bool) {
	t, err := ParseTap(el.Attr("tap"))
	return t, err == nil
}

// ValidDesignName reports whether name can name a design of the project:
// a path of its folder with slashes, without .scenra, whose parts are
// names of files.
func ValidDesignName(name string) bool {
	if name == "" || strings.ContainsAny(name, `\#`) || strings.HasSuffix(name, ".scenra") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.TrimSpace(part) == "" || strings.HasPrefix(part, ".") {
			return false
		}
	}
	return true
}

// DesignPath is the path of the design named name, relative to the
// project's folder, with slashes.
func DesignPath(name string) string { return name + ".scenra" }
