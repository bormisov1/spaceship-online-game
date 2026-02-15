package main

const (
	SpatialCellSize = 80.0 // ~2x largest entity radius (AsteroidRadius=40)
)

// EntityRef identifies an entity in the grid
type EntityRef struct {
	Kind byte // 'p'=player, 'r'=projectile, 'm'=mob, 'a'=asteroid, 'k'=pickup
	Idx  int  // index into the corresponding flat list
}

// SpatialGrid is a dynamically-sized grid for broad-phase collision queries
type SpatialGrid struct {
	cols  int
	rows  int
	cells [][]EntityRef
}

// NewSpatialGrid creates a grid sized for the given world dimensions
func NewSpatialGrid(worldW, worldH float64) SpatialGrid {
	cols := int(worldW/SpatialCellSize) + 1
	rows := int(worldH/SpatialCellSize) + 1
	cells := make([][]EntityRef, cols*rows)
	return SpatialGrid{cols: cols, rows: rows, cells: cells}
}

// Clear resets all cells (keeps allocated capacity)
func (g *SpatialGrid) Clear() {
	for i := range g.cells {
		g.cells[i] = g.cells[i][:0]
	}
}

func (g *SpatialGrid) cellIdx(x, y float64) int {
	cx := int(x / SpatialCellSize)
	cy := int(y / SpatialCellSize)
	if cx < 0 {
		cx = 0
	} else if cx >= g.cols {
		cx = g.cols - 1
	}
	if cy < 0 {
		cy = 0
	} else if cy >= g.rows {
		cy = g.rows - 1
	}
	return cy*g.cols + cx
}

// Insert adds an entity reference at the given position
func (g *SpatialGrid) Insert(x, y float64, ref EntityRef) {
	idx := g.cellIdx(x, y)
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
	if maxCX >= g.cols {
		maxCX = g.cols - 1
	}
	if minCY < 0 {
		minCY = 0
	}
	if maxCY >= g.rows {
		maxCY = g.rows - 1
	}
	for cy := minCY; cy <= maxCY; cy++ {
		for cx := minCX; cx <= maxCX; cx++ {
			idx := cy*g.cols + cx
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
	if maxCX >= g.cols {
		maxCX = g.cols - 1
	}
	if minCY < 0 {
		minCY = 0
	}
	if maxCY >= g.rows {
		maxCY = g.rows - 1
	}
	var result []EntityRef
	for cy := minCY; cy <= maxCY; cy++ {
		for cx := minCX; cx <= maxCX; cx++ {
			idx := cy*g.cols + cx
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
	if maxCX >= g.cols {
		maxCX = g.cols - 1
	}
	if minCY < 0 {
		minCY = 0
	}
	if maxCY >= g.rows {
		maxCY = g.rows - 1
	}
	for cy := minCY; cy <= maxCY; cy++ {
		for cx := minCX; cx <= maxCX; cx++ {
			idx := cy*g.cols + cx
			buf = append(buf, g.cells[idx]...)
		}
	}
	return buf
}
