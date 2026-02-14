package main

const (
	SpatialCellSize = 80.0 // ~2x largest entity radius (AsteroidRadius=40)
	SpatialCols     = 51   // ceil(4000/80) + 1
	SpatialRows     = 51
)

// EntityRef identifies an entity in the grid
type EntityRef struct {
	Kind byte // 'p'=player, 'r'=projectile, 'm'=mob, 'a'=asteroid, 'k'=pickup
	Idx  int  // index into the corresponding flat list
}

// SpatialGrid is a fixed-size grid for broad-phase collision queries
type SpatialGrid struct {
	cells [SpatialCols * SpatialRows][]EntityRef
}

// Clear resets all cells (keeps allocated capacity)
func (g *SpatialGrid) Clear() {
	for i := range g.cells {
		g.cells[i] = g.cells[i][:0]
	}
}

func cellIdx(x, y float64) int {
	cx := int(x / SpatialCellSize)
	cy := int(y / SpatialCellSize)
	if cx < 0 {
		cx = 0
	} else if cx >= SpatialCols {
		cx = SpatialCols - 1
	}
	if cy < 0 {
		cy = 0
	} else if cy >= SpatialRows {
		cy = SpatialRows - 1
	}
	return cy*SpatialCols + cx
}

// Insert adds an entity reference at the given position
func (g *SpatialGrid) Insert(x, y float64, ref EntityRef) {
	idx := cellIdx(x, y)
	g.cells[idx] = append(g.cells[idx], ref)
}

// InsertCircle adds an entity reference to all cells overlapping its bounding box
func (g *SpatialGrid) InsertCircle(x, y, radius float64, ref EntityRef) {
	minCX := int((x - radius) / SpatialCellSize)
	maxCX := int((x + radius) / SpatialCellSize)
	minCY := int((y - radius) / SpatialCellSize)
	maxCY := int((y + radius) / SpatialCellSize)
	if minCX < 0 {
		minCX = 0
	}
	if maxCX >= SpatialCols {
		maxCX = SpatialCols - 1
	}
	if minCY < 0 {
		minCY = 0
	}
	if maxCY >= SpatialRows {
		maxCY = SpatialRows - 1
	}
	for cy := minCY; cy <= maxCY; cy++ {
		for cx := minCX; cx <= maxCX; cx++ {
			idx := cy*SpatialCols + cx
			g.cells[idx] = append(g.cells[idx], ref)
		}
	}
}

// Query returns all entity refs in cells that overlap the given bounding box
func (g *SpatialGrid) Query(x, y, radius float64) []EntityRef {
	minCX := int((x - radius) / SpatialCellSize)
	maxCX := int((x + radius) / SpatialCellSize)
	minCY := int((y - radius) / SpatialCellSize)
	maxCY := int((y + radius) / SpatialCellSize)
	if minCX < 0 {
		minCX = 0
	}
	if maxCX >= SpatialCols {
		maxCX = SpatialCols - 1
	}
	if minCY < 0 {
		minCY = 0
	}
	if maxCY >= SpatialRows {
		maxCY = SpatialRows - 1
	}
	var result []EntityRef
	for cy := minCY; cy <= maxCY; cy++ {
		for cx := minCX; cx <= maxCX; cx++ {
			idx := cy*SpatialCols + cx
			result = append(result, g.cells[idx]...)
		}
	}
	return result
}

// QueryBuf appends results to buf and returns the extended slice, avoiding per-call allocation
func (g *SpatialGrid) QueryBuf(x, y, radius float64, buf []EntityRef) []EntityRef {
	minCX := int((x - radius) / SpatialCellSize)
	maxCX := int((x + radius) / SpatialCellSize)
	minCY := int((y - radius) / SpatialCellSize)
	maxCY := int((y + radius) / SpatialCellSize)
	if minCX < 0 {
		minCX = 0
	}
	if maxCX >= SpatialCols {
		maxCX = SpatialCols - 1
	}
	if minCY < 0 {
		minCY = 0
	}
	if maxCY >= SpatialRows {
		maxCY = SpatialRows - 1
	}
	for cy := minCY; cy <= maxCY; cy++ {
		for cx := minCX; cx <= maxCX; cx++ {
			idx := cy*SpatialCols + cx
			buf = append(buf, g.cells[idx]...)
		}
	}
	return buf
}
