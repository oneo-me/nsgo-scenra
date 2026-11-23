package scenraview

import (
	"math/rand/v2"
	"testing"

	"github.com/nsgo-dev/nsgo/core"
)

func TestPackTightKeepsBoxesApartAndTogether(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for round := range 200 {
		n := 1 + r.IntN(14)
		sizes := make([]core.Size, n)
		for i := range sizes {
			sizes[i] = core.Size{Width: core.Points(40 + r.IntN(800)), Height: core.Points(30 + r.IntN(600))}
		}
		const gap = 24
		at, total := packTight(sizes, gap)
		for i, a := range sizes {
			if at[i].X < 0 || at[i].Y < 0 || at[i].X+a.Width > total.Width+1e-6 || at[i].Y+a.Height > total.Height+1e-6 {
				t.Fatalf("round %d: box %d at %v of %v leaves the whole %v", round, i, at[i], a, total)
			}
			for j := i + 1; j < n; j++ {
				b := sizes[j]
				if at[i].X < at[j].X+b.Width+gap-1e-6 && at[j].X < at[i].X+a.Width+gap-1e-6 &&
					at[i].Y < at[j].Y+b.Height+gap-1e-6 && at[j].Y < at[i].Y+a.Height+gap-1e-6 {
					t.Fatalf("round %d: boxes %d at %v and %d at %v are closer than the gap", round, i, at[i], j, at[j])
				}
			}
		}
		// It wastes no more room than rows in order do.
		_, rows := pack(sizes, gap)
		score := func(s core.Size) float64 { return max(float64(s.Width)/packAspect, float64(s.Height)) }
		if score(total) > score(rows)+1e-6 {
			t.Fatalf("round %d: the tight packing %v takes more room than rows %v", round, total, rows)
		}
	}
}

func TestPackTightFillsAroundATallBox(t *testing.T) {
	// A tall box and small ones: the small ones stack beside it rather
	// than in a row of their own.
	sizes := []core.Size{{Width: 390, Height: 800}, {Width: 200, Height: 100}, {Width: 200, Height: 100}, {Width: 200, Height: 100}, {Width: 200, Height: 100}}
	_, total := packTight(sizes, 20)
	if total.Height > 800 || total.Width > 390+20+2*(200+20) {
		t.Fatalf("the packing is %v", total)
	}
}

func TestCaptionsKeepToTheRoomAboveTheirDesigns(t *testing.T) {
	room := core.Points(captionRoom + 40)
	if k := captionScale(1, room); k != 1 {
		t.Fatalf("at full size a caption grows %v times", k)
	}
	// Zoomed far out, a caption shrinks with the designs rather than
	// covering the row above.
	for _, zoom := range []float64{0.5, 0.2, 0.05} {
		if k := captionScale(zoom, room); core.Points((captionGap+captionHeight)*k) > room+1e-6 || k > 1/zoom {
			t.Fatalf("at zoom %v a caption grows %v times, past its room", zoom, k)
		}
	}
}
