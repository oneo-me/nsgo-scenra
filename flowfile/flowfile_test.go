package flowfile

import (
	"strings"
	"testing"
)

const sample = `# Orders
flow direction=right

node start "Order placed" kind=start
node check "Check stock" note="Reserve every item" status=done
start -> check
check -> ship "in stock"
check ..> start "out \"of\" stock"   # retry
ship -> done -> archive
`

func TestParseFlow(t *testing.T) {
	f, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	d := f.Flow
	if f.Kind != Chart || d.Direction != Right {
		t.Fatalf("header: %q %q", f.Kind, d.Direction)
	}
	var ids []string
	for _, n := range d.Nodes {
		ids = append(ids, n.ID)
	}
	if got := strings.Join(ids, " "); got != "start check ship done archive" {
		t.Fatalf("nodes: %s", got)
	}
	if n, _ := d.Node("check"); n.Note != "Reserve every item" || n.Status != "done" || n.Label != "Check stock" {
		t.Fatalf("check: %+v", n)
	}
	if n, _ := d.Node("ship"); n.Label != "ship" {
		t.Fatalf("an implicit node takes its ID as its label: %+v", n)
	}
	if len(d.Edges) != 5 || d.Edges[2] != (Edge{From: "check", To: "start", Label: `out "of" stock`, Dashed: true}) {
		t.Fatalf("edges: %+v", d.Edges)
	}
}

func TestFormatRoundTrips(t *testing.T) {
	d, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(d.Format())
	if err != nil {
		t.Fatalf("%v\n%s", err, d.Format())
	}
	if again.Format() != d.Format() {
		t.Fatalf("not stable:\n%s\n---\n%s", d.Format(), again.Format())
	}
	if !strings.Contains(d.Format(), `check ..> start "out \"of\" stock"`) {
		t.Fatalf("format:\n%s", d.Format())
	}
}

func TestParseErrors(t *testing.T) {
	for src, want := range map[string]string{
		"flow direction=up":             "line 1: unknown direction",
		"flow\n\nnode a kind=box":       "line 3: unknown kind",
		"flow\nnode a \"x":              "unclosed quote",
		"flow\na -> ":                   "a line of a flow is",
		"flow\na => b":                  "expected -> or ..>",
		"flow\nnode":                    "node needs an ID",
		"flow\nnode a \"A\" colour=red": "unknown attribute",
	} {
		if _, err := Parse(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", src, err, want)
		}
	}
}

func TestEdits(t *testing.T) {
	var d Flow
	for _, n := range []Node{{ID: "a"}, {ID: "b", Label: "B"}} {
		if err := d.AddNode(n); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.AddEdge(Edge{From: "a", To: "c"}); err == nil {
		t.Fatal("an edge to a missing node was added")
	}
	if err := d.AddEdge(Edge{From: "a", To: "b"}); err != nil {
		t.Fatal(err)
	}
	label := "Begin"
	if err := d.UpdateNode("a", NodePatch{Label: &label, Rename: "begin"}); err != nil {
		t.Fatal(err)
	}
	if d.Edges[0].From != "begin" || d.Nodes[0].Label != "Begin" {
		t.Fatalf("rename: %+v", d)
	}
	if err := d.RemoveNode("b"); err != nil || len(d.Edges) != 0 {
		t.Fatalf("remove: %v %+v", err, d)
	}
	bad := "huge"
	if err := d.UpdateNode("begin", NodePatch{Kind: &bad}); err == nil || d.Nodes[0].Kind != "" {
		t.Fatalf("a bad kind was set: %v", err)
	}
}
