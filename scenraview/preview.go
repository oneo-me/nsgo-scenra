package scenraview

import (
	"os"

	"github.com/nsgo-dev/nsgo/base/camera"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/flowfile"
	"github.com/oneo-me/nsgo-scenra/scenraview/locales"
)

// Preview shows a Scenra file read-only on a pan-and-zoom canvas, as a
// QuickLook preview or viewer pane: when Path is set and Source is empty it
// reads the file and resolves Project from ProjectOf(Path), falling back
// gracefully when the surrounding project cannot be read (such as inside a
// sandboxed QuickLook extension). Broken or unreadable files show the error
// state instead of a blank canvas.
type Preview struct {
	Path    string
	Source  string
	Project string
	TestID  string
	Style   core.Style
}

// Render implements core.Node.
func (p Preview) Render(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	tr := locales.For(cx)

	source := p.Source
	project := p.Project
	if p.Path != "" {
		if source == "" {
			data, err := os.ReadFile(p.Path)
			if err != nil {
				return previewBroken(cx, p.TestID, p.Style, err.Error())
			}
			source = string(data)
		}
		if project == "" {
			project = ProjectOf(p.Path)
		}
	}

	f, err := flowfile.Parse(source)
	if err != nil {
		return previewBroken(cx, p.TestID, p.Style, err.Error())
	}

	bg := colors.Inset
	if f.Kind == flowfile.Chart {
		bg = colors.Background
		if !hasFlowNodes(f) {
			return previewEmpty(cx, p.TestID, p.Style, bg, tr.FlowEmpty())
		}
	} else if len(f.Screens) == 0 {
		return previewEmpty(cx, p.TestID, p.Style, bg, tr.DesignEmpty())
	}

	c := &camera.State{}
	fin := ui.FinOf(cx)
	tool := func(icon func() *core.SVG, label string, press func(), testID string) core.Node {
		return ui.IconButton{Icon: icon, Label: label, OnPress: press, TestID: testID}
	}
	toolbar := core.View{
		Style: core.Style{
			Position: core.Absolute, Top: core.Pt(12), Right: core.Pt(12),
			Direction: core.Row, Gap: 2, Padding: core.All(4),
			CornerRadius: ui.RadiusControl + 4, BorderWidth: 1,
		},
		StyleFunc: func() core.Style {
			return fin.Raised().Over(core.Style{Background: colors.Surface(), BorderColor: colors.Border()})
		},
		Children: []core.Node{
			tool(lucide.ZoomOut, tr.ZoomOut(), c.ZoomOut, "preview-zoom-out"),
			tool(lucide.ZoomIn, tr.ZoomIn(), c.ZoomIn, "preview-zoom-in"),
			tool(lucide.Scan, tr.Fit(), c.Fit, "preview-fit"),
		},
	}

	canvas := camera.View{
		TestID:    "preview-camera",
		State:     c,
		FitMargin: designFitMargin,
		Content:   Still{Source: source, Project: project, TestID: "preview-still"},
		Children:  []core.Node{toolbar},
	}
	return core.View{
		TestID:    p.TestID,
		Style:     ui.FillRest.Merge(core.Style{Overflow: core.Hidden}).Merge(p.Style),
		StyleFunc: ui.Filled(bg),
		NoFocus:   true,
		Children:  []core.Node{canvas},
	}
}

func hasFlowNodes(f flowfile.File) bool {
	for _, flow := range f.Flows {
		if len(flow.Nodes) > 0 {
			return true
		}
	}
	return false
}

func previewBroken(cx *core.Scope, testID string, style core.Style, problem string) core.Node {
	colors := ui.ColorsOf(cx)
	tr := locales.For(cx)
	return core.View{
		TestID:    testID,
		Style:     ui.FillRest.Merge(core.Style{AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(32), Gap: 12}).Merge(style),
		StyleFunc: ui.Filled(colors.Background),
		Children: []core.Node{
			core.View{
				TestID: "file-broken",
				Style:  core.Style{AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 12},
				Children: []core.Node{
					ui.Icon(lucide.FileExclamationPoint, 32, colors.Danger),
					ui.Text(tr.FileBroken(), 15, ui.Semibold, colors.Text),
					core.Text{
						Content:   problem,
						TestID:    "file-problem",
						Style:     ui.TypeCode.Merge(core.Style{TextAlign: core.TextAlignCenter, MaxWidth: core.Pt(480)}),
						StyleFunc: ui.Colored(colors.Muted),
					},
				},
			},
		},
	}
}

func previewEmpty(cx *core.Scope, testID string, style core.Style, bg func() core.Color, message string) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{
		TestID:    testID,
		Style:     ui.FillRest.Merge(core.Style{AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Padding: core.All(24)}).Merge(style),
		StyleFunc: ui.Filled(bg),
		Children: []core.Node{
			core.Text{Content: message, Style: ui.TypeBody, StyleFunc: ui.Colored(colors.Muted)},
		},
	}
}
