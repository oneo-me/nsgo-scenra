package scenraview

import (
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// pinSize is a pin's diameter on screen, and cardWidth the card's width.
const (
	pinSize   = 22
	cardWidth = 320
)

// remarkPins mark the elements of the designs of s the user commented on or
// noted with a pin on each one's top right corner, a note's left of a
// comment's, their size on screen the same at any zoom.
func remarkPins(cx *core.Scope, s flowfile.File, sel selection, zoom func() float64) []core.Node {
	colors := ui.ColorsOf(cx)
	commented := map[string]bool{}
	for _, c := range s.Comments() {
		commented[elementKey(c.Screen, c.Path)] = true
	}
	pin := func(path, testID string, shift bool, icon func() *core.SVG, background, color func() core.Color) core.Node {
		return core.View{TestID: testID, PointerEvents: core.PointerEventsNone,
			StyleFunc: func() core.Style {
				sel.laidOut.Get()
				b, ok := sel.bounds(path)
				if !ok {
					return core.Style{Display: core.DisplayNone}
				}
				x := b.Origin.X + b.Size.Width - pinSize/2
				if shift {
					x -= core.Points(float64(pinSize+4) / zoom())
				}
				return core.Style{Position: core.Absolute, Left: core.Pt(x), Top: core.Pt(b.Origin.Y - pinSize/2),
					Width: core.Pt(pinSize), Height: core.Pt(pinSize), CornerRadius: pinSize / 2, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
					Background: background(), Transform: core.Transform{Scale: float32(1 / zoom())}}
			},
			Children: []core.Node{ui.Icon(icon, 12, color)}}
	}
	var out []core.Node
	for _, c := range s.Comments() {
		key := elementKey(c.Screen, c.Path)
		out = append(out, pin(key, "comment-pin-"+key, false, lucide.MessageSquare, colors.Accent, colors.OnAccent))
	}
	for _, n := range s.Notes() {
		key := elementKey(n.Screen, n.Path)
		out = append(out, pin(key, "note-pin-"+key, commented[key], lucide.StickyNote, colors.Text, colors.Background))
	}
	return out
}

// remarkOptions configure the card of an element's comment and note.
type remarkOptions struct {
	// path is the key of the element of the designs shown the card is
	// about.
	path func() string
	// ask, when set, asks the assistant to resolve the comment.
	ask func(prompt string)
	// place says where a card of a size goes in the design pane.
	place func(core.Size) (core.Point, bool)
	// over hears whether the pointer is over the card, and held whether
	// it has changes to keep it.
	over, held func(bool)
}

// remarkCard edits the comment and the note on an element of the design
// shown, saving them together, says what pressing the element plays, and
// asks the assistant to resolve the comment, saving it first. The card
// moves by its position, not a transform, as platforms hit-test views
// without their transforms.
func remarkCard(cx *core.Scope, e *Editor, o remarkOptions) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	fin := ui.FinOf(cx)
	element := func() flowfile.Element {
		el, _ := elementOf(e.cur().doc.Get(), o.path())
		return el
	}
	path := func() string { return core.Untracked(cx, o.path) }
	// tapping is the tap pressing the element plays, its own or that of
	// the innermost element holding it with one.
	tapping := func() (flowfile.Tap, bool) {
		screen, path, ok := splitKey(o.path())
		f := e.cur().doc.Get()
		if !ok || screen >= len(f.Screens) {
			return flowfile.Tap{}, false
		}
		els := f.Screens[screen].UI
		at, err := flowfile.ParsePath(path)
		for i := len(at); err == nil && i > 0; i-- {
			if el, ok := flowfile.ElementAt(els, flowfile.FormatPath(at[:i])); ok {
				if t, ok := el.Tap(); ok {
					return t, true
				}
			}
		}
		return flowfile.Tap{}, false
	}
	comment := core.NewMemo(cx, func() string { return element().Attr("comment") })
	note := core.NewMemo(cx, func() string { return element().Attr("note") })
	draft, noteDraft := core.NewSignal(cx, ""), core.NewSignal(cx, "")
	// The drafts start from the element's comment and note, and follow
	// them when they change, such as when the assistant resolves one.
	cx.Effect(func() {
		o.path()
		c, n := comment.Get(), note.Get()
		core.Untracked(cx, func() bool { draft.Set(c); noteDraft.Set(n); return true })
	})
	text := func() string { return strings.TrimSpace(draft.Get()) }
	noteText := func() string { return strings.TrimSpace(noteDraft.Get()) }
	changed := func() bool { return text() != comment.Get() || noteText() != note.Get() }
	// The card stays while it has changes.
	cx.Effect(func() {
		keep := changed()
		core.Untracked(cx, func() bool { o.held(keep); return true })
	})
	cx.Cleanup(func() { o.held(false); o.over(false) })
	save := func() {
		e.shown().annotate(path(), map[string]string{"comment": core.Untracked(cx, text), "note": core.Untracked(cx, noteText)})
	}
	size := core.NewSignal(cx, core.Size{})
	section := func(icon func() *core.SVG, title string, has func() bool, remove core.Node, input core.Node) core.Node {
		return core.View{Style: core.Style{Gap: 6}, Children: []core.Node{
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, MinHeight: core.Pt(24)}, Children: []core.Node{
				ui.Icon(icon, 14, colors.Muted),
				ui.Text(title, 13, ui.Semibold, colors.Text),
				ui.Spacer{},
				core.Show{When: has, Then: func(*core.Scope) core.Node { return remove }},
			}},
			input,
		}}
	}
	return core.View{TestID: "comment-card",
		Style:        core.Style{Position: core.Absolute, Width: core.Pt(cardWidth), Padding: core.All(12), Gap: 10, CornerRadius: ui.RadiusCard, BorderWidth: 1},
		OnLayout:     func(b core.Bounds) { size.Set(b.Size) },
		OnMouseEnter: func(core.MouseEvent) { o.over(true) },
		OnMouseLeave: func(core.MouseEvent) { o.over(false) },
		StyleFunc: func() core.Style {
			at, ok := o.place(size.Get())
			if !ok {
				return core.Style{Display: core.DisplayNone}
			}
			return fin.Raised().Over(core.Style{Left: core.Pt(at.X), Top: core.Pt(at.Y), Background: colors.Surface(), BorderColor: colors.Border()})
		},
		Children: []core.Node{
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
				core.Text{ContentFunc: func() string { return element().Kind + " " + keyLabel(e.cur().doc.Get(), o.path()) }, MaxLines: 1,
					Style: ui.TypeCaption.Merge(core.Style{FlexShrink: 1}), StyleFunc: ui.Colored(colors.Muted)},
				ui.Spacer{},
				core.Show{When: func() bool { _, ok := tapping(); return ok }, Then: func(*core.Scope) core.Node {
					return core.View{TestID: "comment-tap", Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4, FlexShrink: 1}, Children: []core.Node{
						ui.Icon(lucide.MousePointerClick, 12, colors.Muted),
						core.Text{ContentFunc: func() string { t, _ := tapping(); return tapLabel(tr, t) }, MaxLines: 1,
							Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
					}}
				}},
			}},
			section(lucide.MessageSquare, tr.Comment(), func() bool { return comment.Get() != "" },
				ui.IconButton{Icon: lucide.Trash, Label: tr.RemoveComment(), TestID: "comment-remove",
					OnPress: func() { e.shown().setComment(path(), "") }},
				ui.Input{Value: draft.Get, OnChange: draft.Set, OnSubmit: func(string) { save() }, Placeholder: tr.CommentPlaceholder(),
					Label: tr.Comment(), TestID: "comment-input"}),
			section(lucide.StickyNote, tr.Note(), func() bool { return note.Get() != "" },
				ui.IconButton{Icon: lucide.Trash, Label: tr.RemoveNote(), TestID: "note-remove",
					OnPress: func() { e.shown().setNote(path(), "") }},
				ui.Input{Value: noteDraft.Get, OnChange: noteDraft.Set, OnSubmit: func(string) { save() }, Placeholder: tr.NotePlaceholder(),
					Label: tr.Note(), TestID: "note-input"}),
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
				ui.Spacer{},
				ui.Button{Text: tr.SaveComment(), Variant: ui.ButtonOutline, Size: ui.SizeSmall, TestID: "comment-save",
					Disabled: func() bool { return !changed() }, OnPress: save},
				core.Show{When: func() bool { return o.ask != nil }, Then: func(*core.Scope) core.Node {
					return ui.Button{Text: tr.AskAssistant(), Icon: lucide.Sparkles, Variant: ui.ButtonPrimary, Size: ui.SizeSmall, TestID: "comment-ask",
						Disabled: func() bool { return text() == "" },
						OnPress: func() {
							save()
							o.ask(resolvePrompt(tr, e.shown().doc.GetUntracked(), path(), core.Untracked(cx, text)))
						}}
				}},
			}},
		}}
}

// elementOf is the element of f's designs at key.
func elementOf(f flowfile.File, key string) (flowfile.Element, bool) {
	screen, path, ok := splitKey(key)
	if !ok || screen >= len(f.Screens) {
		return flowfile.Element{}, false
	}
	return flowfile.ElementAt(f.Screens[screen].UI, path)
}

// keyLabel names the element of f's designs at key as the agent and the
// card do: its path, after its design's ID in a file of several.
func keyLabel(f flowfile.File, key string) string {
	screen, path, ok := splitKey(key)
	if !ok || !f.Several() || screen >= len(f.Screens) {
		return path
	}
	return f.Screens[screen].ID + " " + path
}

// resolvePrompt asks the assistant to resolve the comment on the element
// of f's designs at key.
func resolvePrompt(tr locales.Messages, f flowfile.File, key, comment string) string {
	screen, path, _ := splitKey(key)
	if f.Several() && screen < len(f.Screens) {
		return tr.ResolveDesignComment(path, f.Screens[screen].ID, comment)
	}
	return tr.ResolveComment(path, comment)
}

// tapLabel says what pressing an element with the tap t does.
func tapLabel(tr locales.Messages, t flowfile.Tap) string {
	switch t.Action {
	case flowfile.TapOpen:
		return tr.TapOpens(t.Target)
	case flowfile.TapClose:
		return tr.TapCloses()
	}
	return tr.TapShows(t.Target)
}
