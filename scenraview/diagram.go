package scenraview

import (
	"slices"
	"strconv"
	"strings"

	"github.com/nsgo-dev/nsgo/base/camera"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	uiflow "github.com/nsgo-dev/nsgo/ui/flow"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// nodeKey is the diagram's ID of the node with id of the flow at index
// flow of a file, and splitNode its parts.
func nodeKey(flow int, id string) string { return strconv.Itoa(flow) + "/" + id }

func splitNode(key string) (flow int, id string, ok bool) {
	head, id, ok := strings.Cut(key, "/")
	n, err := strconv.Atoi(head)
	if !ok || err != nil || n < 0 {
		return 0, "", false
	}
	return n, id, true
}

// nodeOf is the node of f's flows with the diagram's ID key.
func nodeOf(f flowfile.File, key string) (flowfile.Node, bool) {
	flow, id, ok := splitNode(key)
	if !ok || flow >= len(f.Flows) {
		return flowfile.Node{}, false
	}
	return f.Flows[flow].Node(id)
}

// flowData maps the flow at index i of the file shown onto a diagram's
// data. Each node's face reads its node from the document, as the view
// keeps a face while its ID stays.
func flowData(e *Editor, i int) func() uiflow.Data {
	return func() uiflow.Data {
		var data uiflow.Data
		f := e.cur().doc.Get()
		if i >= len(f.Flows) {
			return data
		}
		for _, n := range f.Flows[i].Nodes {
			data.Nodes = append(data.Nodes, uiflow.Node{ID: n.ID, Label: n.Label, Muted: n.Status == "skipped",
				Content: nodeCard{e: e, key: nodeKey(i, n.ID)}})
		}
		for _, l := range f.Flows[i].Edges {
			data.Edges = append(data.Edges, uiflow.Edge{Source: l.From, Target: l.To, Label: l.Label, Dashed: l.Dashed})
		}
		return data
	}
}

// flowDirection is the way the flow at index i of the file shown points.
func flowDirection(e *Editor, i int) func() uiflow.Direction {
	return func() uiflow.Direction {
		if f := e.cur().doc.Get(); i < len(f.Flows) && f.Flows[i].Direction == flowfile.Right {
			return uiflow.LeftToRight
		}
		return uiflow.TopToBottom
	}
}

// chart is the state of the diagram of the flow at index i, kept by the
// editor so a diagram at an index stays as files change.
func (e *Editor) chart(i int) *uiflow.State {
	for len(e.charts) <= i {
		e.charts = append(e.charts, &uiflow.State{})
	}
	return e.charts[i]
}

// revealNode has the camera center the node with the key.
func (e *Editor) revealNode(key string) {
	if flow, id, ok := splitNode(key); ok {
		e.chart(flow).Reveal(id)
	}
}

// flowLayout is where each flow of f is in the plane, packed f.Spacing()
// apart at the sizes their diagrams laid out at, under their captions when
// there are several, and the size of the whole.
func flowLayout(f flowfile.File, size func(i int) core.Size) ([]core.Point, core.Size) {
	room := core.Points(0)
	if f.Several() {
		room = captionRoom
	}
	sizes := make([]core.Size, len(f.Flows))
	for i := range f.Flows {
		s := size(i)
		sizes[i] = core.Size{Width: s.Width, Height: s.Height + room}
	}
	at, total := pack(sizes, core.Points(f.Spacing()))
	for i := range at {
		at[i].Y += room
	}
	return at, total
}

// nodeCard is a node's face: its kind's icon tile, label and note, and
// its status above them.
type nodeCard struct {
	e   *Editor
	key string
}

// Render implements core.Node.
func (c nodeCard) Render(cx *core.Scope) core.Node {
	node := core.NewMemo(cx, func() flowfile.Node {
		n, _ := nodeOf(c.e.cur().doc.Get(), c.key)
		return n
	})
	return core.Dynamic(func(cx *core.Scope) core.Node { return card(cx, node.Get()) })
}

func card(cx *core.Scope, n flowfile.Node) core.Node {
	colors := ui.ColorsOf(cx)
	tile, icon := colors.Accent, lucide.ListChecks
	switch n.Kind {
	case "start":
		tile, icon = colors.Up, lucide.Play
	case "end":
		tile, icon = colors.Danger, lucide.Flag
	case "decision":
		tile, icon = colors.Warn, lucide.Split
	case "data":
		tile, icon = colors.Info, lucide.Database
	case "note":
		tile, icon = colors.Muted, lucide.StickyNote
	}
	card := uiflow.Card{Title: n.Label, Description: n.Note,
		Leading: core.View{Style: core.Style{Width: core.Pt(30), Height: core.Pt(30), CornerRadius: 8, FlexShrink: 0,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
			StyleFunc: func() core.Style { return core.Style{Background: tile().Faded(.16)} },
			Children:  []core.Node{ui.Icon(icon, 16, tile)}}}
	if n.Note != "" {
		// A note may be long: let it wrap to a few lines.
		card.Body = core.Group{
			core.Text{Content: n.Label, MaxLines: 2, Style: ui.TypeBody.Merge(core.Style{FontWeight: ui.Semibold}), StyleFunc: ui.Colored(colors.Text)},
			core.Text{Content: n.Note, MaxLines: 3, Style: core.Style{FontSize: 11, LineHeight: 15}, StyleFunc: ui.Colored(colors.Muted)},
		}
	}
	if n.Status != "" {
		tr := locales.For(cx)
		color, icon, label := colors.Muted, lucide.CircleDashed, tr.StatusTodo()
		switch n.Status {
		case "doing":
			color, icon, label = colors.Accent, lucide.LoaderCircle, tr.StatusDoing()
			card.Variant = uiflow.CardAccent
		case "done":
			color, icon, label = colors.Up, lucide.CircleCheck, tr.StatusDone()
		case "blocked":
			color, icon, label = colors.Danger, lucide.CircleAlert, tr.StatusBlocked()
		case "skipped":
			icon, label = lucide.CircleSlash, tr.StatusSkipped()
		}
		card.Header = core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4}, Children: []core.Node{
			ui.Icon(icon, 12, color), ui.Text(label, 11, ui.Medium, color)}}
	}
	return card
}

// flowPane is the flows shown, each a diagram of its own packed on one
// camera under its name when there are several, with the pane's tools
// over them. The camera hands the pointer to the diagram under it. The
// diagrams at each index follow the files shown instead of remounting
// per file, as remounting a chart is costly.
func flowPane(cx *core.Scope, e *Editor) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	fin := ui.FinOf(cx)
	full := func() bool {
		return slices.ContainsFunc(e.cur().doc.Get().Flows, func(d flowfile.Flow) bool { return len(d.Nodes) > 0 })
	}
	tool := func(icon func() *core.SVG, label string, enabled func() bool, press func(), testID string) core.Node {
		b := ui.IconButton{Icon: icon, Label: label, OnPress: press, TestID: testID}
		if enabled != nil {
			b.Disabled = func() bool { return !enabled() }
		}
		return b
	}
	toolbar := core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(8), Right: core.Pt(8), Direction: core.Row, Gap: 2,
		Padding: core.All(4), CornerRadius: ui.RadiusControl + 4, BorderWidth: 1},
		StyleFunc: func() core.Style {
			return fin.Raised().Over(core.Style{Background: colors.Surface(), BorderColor: colors.Border()})
		},
		Children: []core.Node{
			tool(lucide.ArrowRightLeft, tr.LeftToRight(), full, e.ToggleDirection, "direction"),
			tool(lucide.ZoomOut, tr.ZoomOut(), full, e.camera.ZoomOut, "zoom-out"),
			tool(lucide.ZoomIn, tr.ZoomIn(), full, e.camera.ZoomIn, "zoom-in"),
			tool(lucide.Scan, tr.Fit(), full, e.camera.Fit, "fit"),
		}}
	count := core.NewMemo(cx, func() int { return len(e.cur().doc.Get().Flows) })
	layout := func() ([]core.Point, core.Size) {
		return flowLayout(e.cur().doc.Get(), func(i int) core.Size { return e.chart(i).Size() })
	}
	origin := func(i int) core.Point {
		if at, _ := layout(); i < len(at) {
			return at[i]
		}
		return core.Point{}
	}
	// flowAt is the flow whose diagram is at a point of the plane, and the
	// point in the diagram's coordinates.
	flowAt := func(p core.Point) (int, core.Point, bool) {
		at, _ := layout()
		for i, o := range at {
			size := e.chart(i).Size()
			local := core.Point{X: p.X - o.X, Y: p.Y - o.Y}
			if local.X >= 0 && local.Y >= 0 && local.X < size.Width && local.Y < size.Height {
				return i, local, true
			}
		}
		return 0, core.Point{}, false
	}
	selectedIn := func(i int) func() string {
		return func() string {
			if flow, id, ok := splitNode(e.cur().selected.Get()); ok && flow == i {
				return id
			}
			return ""
		}
	}
	item := func(cx *core.Scope, i int) core.Node {
		return core.Group{
			core.Show{When: func() bool { return e.cur().doc.Get().Several() }, Then: func(cx *core.Scope) core.Node {
				return core.Dynamic(func(cx *core.Scope) core.Node {
					f := e.cur().doc.Get()
					if i >= len(f.Flows) {
						return core.Group{}
					}
					name := f.Flows[i].Name
					if name == "" {
						name = f.Flows[i].ID
					}
					o := origin(i)
					return caption(cx, name, o, int(e.chart(i).Size().Width), e.camera.Zoom)
				})
			}},
			core.View{TestID: "flow-" + strconv.Itoa(i), StyleFunc: func() core.Style {
				o := origin(i)
				return core.Style{Position: core.Absolute, Left: core.Pt(o.X), Top: core.Pt(o.Y)}
			}, Children: []core.Node{uiflow.Diagram{State: e.chart(i), Camera: e.camera, Origin: func() core.Point { return origin(i) },
				Data: flowData(e, i), Direction: flowDirection(e, i), Selected: selectedIn(i),
				OnSelect: func(id string) {
					if id == "" {
						e.shown().selected.Set("")
					} else {
						e.shown().selected.Set(nodeKey(i, id))
					}
				}}}},
		}
	}
	indices := func() []int {
		out := make([]int, count.Get())
		for i := range out {
			out[i] = i
		}
		return out
	}
	board := core.View{TestID: "flows", StyleFunc: func() core.Style {
		_, total := layout()
		return core.Style{Width: core.Pt(total.Width), Height: core.Pt(total.Height)}
	}, Children: []core.Node{core.Each(indices, func(i int) int { return i }, func(cx *core.Scope, i func() int, _ func() int) core.Node {
		return item(cx, core.Untracked(cx, i))
	})}}
	press := func(p core.Point, ev core.PressEvent) bool {
		if i, local, ok := flowAt(p); ok {
			return e.chart(i).Press(local, ev)
		}
		if e.shown().selected.GetUntracked() != "" {
			e.shown().selected.Set("")
		}
		return false
	}
	pointAt := func(p core.Point, over bool) {
		i, local, ok := flowAt(p)
		for j := range count.GetUntracked() {
			e.chart(j).Hover(local, over && ok && i == j)
		}
	}
	canvas := camera.View{TestID: "flow", State: e.camera, Content: board, OnPress: press, OnHover: pointAt,
		LabelFunc: func() string {
			nodes, edges := 0, 0
			for _, d := range e.cur().doc.Get().Flows {
				nodes, edges = nodes+len(d.Nodes), edges+len(d.Edges)
			}
			return tr.DiagramLabel(nodes, edges)
		},
		Cursor: func() core.Cursor {
			for j := range count.Get() {
				if e.chart(j).Hovered() != "" {
					// Nodes are pressables, which keep the platform's arrow.
					return core.CursorDefault
				}
			}
			return core.CursorAuto
		},
		Children: []core.Node{toolbar}}
	empty := core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0),
		AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(24)}, PointerEvents: core.PointerEventsNone,
		Children: []core.Node{core.Text{Content: tr.FlowEmpty(), Style: ui.TypeBody, StyleFunc: ui.Colored(colors.Muted)}}}
	return core.View{Style: ui.FillRest, StyleFunc: ui.Filled(colors.Background), Children: []core.Node{canvas,
		core.Show{When: func() bool { return !full() }, Then: func(*core.Scope) core.Node { return empty }}}}
}

// toggled is the flows ToggleDirection turns: that of the selected node,
// or else all.
func toggled(f flowfile.File, selected string) []int {
	if flow, _, ok := splitNode(selected); ok && flow < len(f.Flows) {
		return []int{flow}
	}
	var all []int
	for i := range f.Flows {
		all = append(all, i)
	}
	return all
}

// ToggleDirection turns the flow of the selected node, or else every
// flow shown, between down and right as the first of them points,
// unless the agent works.
func (e *Editor) ToggleDirection() {
	d := e.shown()
	f := d.doc.GetUntracked().Clone()
	which := toggled(f, d.selected.GetUntracked())
	if e.locked() || f.Kind != flowfile.Chart || len(which) == 0 {
		return
	}
	to := flowfile.Right
	if f.Flows[which[0]].Direction == flowfile.Right {
		to = flowfile.Down
	}
	for _, i := range which {
		f.Flows[i].Direction = to
	}
	d.change(f, "")
	e.cx.Post(e.camera.Fit)
}
