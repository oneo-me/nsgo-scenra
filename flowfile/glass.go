package flowfile

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Liquid Glass styles glass= takes, matching NSGo's core.GlassEffect,
// with optional GlassInteractive for core.GlassInteractive.
const (
	GlassRegular     = "regular"
	GlassClear       = "clear"
	GlassNone        = "none"
	GlassInteractive = "interactive"
)

// GlassStyles are the styles glass= takes.
var GlassStyles = []string{GlassRegular, GlassClear, GlassNone}

// Glass is an element's Liquid Glass effect: its style (regular, clear
// or none) and whether it responds to presses on platforms that support
// it.
type Glass struct {
	Style       string
	Interactive bool
}

// String writes g as a glass= attribute value.
func (g Glass) String() string {
	if g.Interactive && g.Style != "" && g.Style != GlassNone {
		return g.Style + " " + GlassInteractive
	}
	return g.Style
}

var errGlass = errors.New(`glass= takes regular, clear or none, with optional interactive, such as glass=regular or glass="regular interactive"`)

// ParseGlass reads a glass= attribute: regular, clear or none, with
// optional interactive for regular and clear.
func ParseGlass(s string) (Glass, error) {
	words := strings.Fields(s)
	if len(words) == 0 {
		return Glass{}, errGlass
	}
	var g Glass
	for _, w := range words {
		switch {
		case slices.Contains(GlassStyles, w):
			if g.Style != "" {
				return Glass{}, fmt.Errorf("glass=%q sets two styles; use one of %s", s, strings.Join(GlassStyles, ", "))
			}
			g.Style = w
		case w == GlassInteractive:
			if g.Interactive {
				return Glass{}, fmt.Errorf("glass=%q repeats %s", s, GlassInteractive)
			}
			g.Interactive = true
		default:
			return Glass{}, fmt.Errorf("unknown glass %q; use %s, with optional %s", w, strings.Join(GlassStyles, ", "), GlassInteractive)
		}
	}
	if g.Style == "" {
		return Glass{}, fmt.Errorf("glass=%q needs %s or %s", s, GlassRegular, GlassClear)
	}
	if g.Style == GlassNone && g.Interactive {
		return Glass{}, fmt.Errorf("glass=%q cannot be %s; use %s or %s", s, GlassInteractive, GlassRegular, GlassClear)
	}
	return g, nil
}

// Glass is the element's Liquid Glass effect, none when unset.
func (el Element) Glass() (Glass, bool) {
	g, err := ParseGlass(el.Attr("glass"))
	return g, err == nil
}
