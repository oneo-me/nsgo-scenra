package scenraview

import (
	"fmt"
	"strings"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// location is what the user sees: a screen, one of its flows, "" for
// none, and a selected node of that flow, or the screen's design when
// design says, and the path of a selected element of it.
type location struct {
	screen, flow, node string
	design             bool
	element            string
}

// view tells an agent what the user sees of p, open at at, and the
// problems of the file: where the user looks, what is selected, the
// user's comments and the classes that draw nothing.
func view(p flowfile.Project, at location) string {
	var b strings.Builder
	if at.screen != "" {
		fmt.Fprintf(&b, "The user has screen %q open", at.screen)
		switch {
		case at.design:
			b.WriteString(" on its design")
		case at.flow != "":
			fmt.Fprintf(&b, " on flow %q", at.flow)
		}
		b.WriteString(".\n")
	} else if len(p.Screens) == 0 {
		b.WriteString("The file has no screens yet.\n")
	}
	if at.node != "" {
		fmt.Fprintf(&b, "The user has selected node %q of the flow.\n", at.node)
	}
	if i := p.ScreenIndex(at.screen); i >= 0 && at.design && at.element != "" {
		if el, ok := flowfile.ElementAt(p.Screens[i].UI, at.element); ok {
			fmt.Fprintf(&b, "The user has selected element %s of the design:\n\n```\n%s```\n", at.element, flowfile.FormatUI([]flowfile.Element{el}))
		}
	}
	if comments := p.Comments(); len(comments) > 0 {
		b.WriteString("The user's open comments:\n")
		for _, c := range comments {
			fmt.Fprintf(&b, "- screen %s, element %s: %q\n", c.Screen, c.Path, c.Text)
		}
	}
	if classes := unknownClasses(p); len(classes) > 0 {
		fmt.Fprintf(&b, "These classes draw nothing: %s.\n", strings.Join(classes, ", "))
	}
	return b.String()
}

// cmp is a, or b when a is empty.
func cmp(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
