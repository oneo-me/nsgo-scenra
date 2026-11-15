package scenraview

import (
	"math"

	"github.com/nsgo-dev/nsgo/core"
)

// packAspect is the shape a file's designs or flows pack toward: a
// landscape pane's, which shows them whole at the largest scale.
const packAspect = 16.0 / 10

// pack places boxes of sizes in rows, in their order and gap apart,
// wrapping the rows where the whole comes closest to packAspect, so a fit
// leaves the least margin: it tries every row width the order allows and
// keeps the one that fits such a pane at the largest scale, the first of
// equals. It returns each box's top left corner and the whole's size.
func pack(sizes []core.Size, gap core.Points) ([]core.Point, core.Size) {
	if len(sizes) == 0 {
		return nil, core.Size{}
	}
	var best []core.Point
	var bestTotal core.Size
	bestScore := math.Inf(1)
	limit := core.Points(0)
	for i, s := range sizes {
		if i > 0 {
			limit += gap
		}
		limit += s.Width
		at, total := shelves(sizes, gap, limit)
		if score := max(float64(total.Width)/packAspect, float64(total.Height)); score < bestScore-1e-9 {
			best, bestTotal, bestScore = at, total, score
		}
	}
	return best, bestTotal
}

// shelves places boxes in rows no wider than limit, unless one box is,
// each row below the tallest box of the one before.
func shelves(sizes []core.Size, gap, limit core.Points) ([]core.Point, core.Size) {
	at := make([]core.Point, len(sizes))
	var total core.Size
	var x, y, row core.Points
	for i, s := range sizes {
		if x > 0 && x+s.Width > limit+1e-6 {
			y += row + gap
			x, row = 0, 0
		}
		at[i] = core.Point{X: x, Y: y}
		total.Width = max(total.Width, x+s.Width)
		row = max(row, s.Height)
		total.Height = max(total.Height, y+row)
		x += s.Width + gap
	}
	return at, total
}
