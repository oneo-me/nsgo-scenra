package scenraview

import (
	"math"
	"slices"

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

// packTight places boxes of sizes gap apart, in any order, where the
// whole wastes the least room in a landscape pane: it packs them by the
// skyline, each box where its top lands highest and then leftmost, at
// container widths from the widest box to all boxes in a row and in
// several orders, and keeps the packing whose smallest packAspect
// rectangle around it is the smallest, the first of equals. It returns
// each box's top left corner, by the index of sizes, and the whole's
// size.
func packTight(sizes []core.Size, gap core.Points) ([]core.Point, core.Size) {
	if len(sizes) == 0 {
		return nil, core.Size{}
	}
	var widest, sum, area core.Points
	for _, s := range sizes {
		widest = max(widest, s.Width)
		sum += s.Width + gap
		area += (s.Width + gap) * (s.Height + gap)
	}
	widths := []core.Points{widest, max(widest, core.Points(math.Sqrt(float64(area)*packAspect)))}
	const steps = 48
	for i := 0; i <= steps; i++ {
		widths = append(widths, widest+(sum-widest)*core.Points(i)/steps)
	}
	orders := [][]int{
		sortedBy(sizes, func(a, b core.Size) bool { return a.Height > b.Height || a.Height == b.Height && a.Width > b.Width }),
		sortedBy(sizes, func(a, b core.Size) bool { return a.Width*a.Height > b.Width*b.Height }),
		sortedBy(sizes, func(a, b core.Size) bool { return a.Width > b.Width || a.Width == b.Width && a.Height > b.Height }),
	}
	var best []core.Point
	var bestTotal core.Size
	bestScore := math.Inf(1)
	for _, order := range orders {
		for _, limit := range widths {
			at, total := skyline(sizes, order, gap, limit)
			if score := max(float64(total.Width)/packAspect, float64(total.Height)); score < bestScore-1e-9 {
				best, bestTotal, bestScore = at, total, score
			}
		}
	}
	return best, bestTotal
}

// sortedBy is the indexes of sizes ordered by before, stable.
func sortedBy(sizes []core.Size, before func(a, b core.Size) bool) []int {
	out := make([]int, len(sizes))
	for i := range out {
		out[i] = i
	}
	slices.SortStableFunc(out, func(a, b int) int {
		switch {
		case before(sizes[a], sizes[b]):
			return -1
		case before(sizes[b], sizes[a]):
			return 1
		}
		return 0
	})
	return out
}

// segment is a stretch of a skyline: from x, w wide, at height y.
type segment struct{ x, w, y core.Points }

// skyline places the boxes of sizes in order in a container limit wide,
// unless a box is wider, gap apart: each at the x where it rests lowest
// on those placed before, then the leftmost.
func skyline(sizes []core.Size, order []int, gap, limit core.Points) ([]core.Point, core.Size) {
	at := make([]core.Point, len(sizes))
	var total core.Size
	limit = max(limit, slices.MaxFunc(sizes, func(a, b core.Size) int { return cmpPoints(a.Width, b.Width) }).Width)
	line := []segment{{0, limit + gap, 0}}
	for _, i := range order {
		w, h := sizes[i].Width+gap, sizes[i].Height+gap
		bestY, bestX, found := core.Points(math.Inf(1)), core.Points(0), false
		for s := range line {
			x := line[s].x
			if x+w > limit+gap+1e-6 {
				break
			}
			// The box rests on the highest segment under it.
			y, reach := core.Points(0), x+w
			for k := s; k < len(line) && line[k].x < reach-1e-6; k++ {
				y = max(y, line[k].y)
			}
			if y < bestY-1e-6 {
				bestY, bestX, found = y, x, true
			}
		}
		if !found {
			bestX, bestY = 0, 0
			for _, seg := range line {
				bestY = max(bestY, seg.y)
			}
		}
		at[i] = core.Point{X: bestX, Y: bestY}
		total.Width = max(total.Width, bestX+sizes[i].Width)
		total.Height = max(total.Height, bestY+sizes[i].Height)
		line = raise(line, bestX, w, bestY+h)
	}
	return at, total
}

// raise sets the skyline to y from x for w, splitting the segments it
// cuts.
func raise(line []segment, x, w, y core.Points) []segment {
	var out []segment
	end := x + w
	added := false
	for _, s := range line {
		sEnd := s.x + s.w
		if sEnd <= x+1e-6 || s.x >= end-1e-6 {
			if s.x >= end-1e-6 && !added {
				out = append(out, segment{x, w, y})
				added = true
			}
			out = append(out, s)
			continue
		}
		if s.x < x {
			out = append(out, segment{s.x, x - s.x, s.y})
		}
		if !added {
			out = append(out, segment{x, w, y})
			added = true
		}
		if sEnd > end {
			out = append(out, segment{end, sEnd - end, s.y})
		}
	}
	if !added {
		out = append(out, segment{x, w, y})
	}
	return out
}

func cmpPoints(a, b core.Points) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
