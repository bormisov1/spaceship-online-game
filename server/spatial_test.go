package main

import "testing"

func TestSpatialGridInsertAndQuery(t *testing.T) {
	grid := NewSpatialGrid(WorldWidth, WorldHeight)
	grid.Clear()

	ref := EntityRef{Kind: 'p', Idx: 0}
	grid.Insert(100, 100, ref)

	// Query around (100,100) should find it
	results := grid.Query(100, 100, 50)
	found := false
	for _, r := range results {
		if r.Kind == 'p' && r.Idx == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find entity at (100,100)")
	}

	// Query far away should not find it
	results = grid.Query(3000, 3000, 50)
	for _, r := range results {
		if r.Kind == 'p' && r.Idx == 0 {
			t.Error("should not find entity at (3000,3000)")
		}
	}
}

func TestSpatialGridClear(t *testing.T) {
	grid := NewSpatialGrid(WorldWidth, WorldHeight)
	grid.Clear()

	grid.Insert(500, 500, EntityRef{Kind: 'm', Idx: 0})
	grid.Clear()

	results := grid.Query(500, 500, 100)
	if len(results) != 0 {
		t.Errorf("expected 0 results after clear, got %d", len(results))
	}
}

func TestSpatialGridInsertCircle(t *testing.T) {
	grid := NewSpatialGrid(WorldWidth, WorldHeight)
	grid.Clear()

	// Insert a large entity (asteroid radius 40)
	grid.InsertCircle(160, 160, 40, EntityRef{Kind: 'a', Idx: 0})

	// Query at edge of bounding box should find it
	results := grid.Query(120, 120, 5)
	found := false
	for _, r := range results {
		if r.Kind == 'a' && r.Idx == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find circle entity near its edge")
	}
}

func TestSpatialGridBoundaryClamp(t *testing.T) {
	grid := NewSpatialGrid(WorldWidth, WorldHeight)
	grid.Clear()

	// Negative coords should clamp to 0
	grid.Insert(-10, -10, EntityRef{Kind: 'p', Idx: 0})
	results := grid.Query(0, 0, 50)
	found := false
	for _, r := range results {
		if r.Kind == 'p' && r.Idx == 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected to find entity inserted at negative coords")
	}

	// Beyond world edge should clamp to max
	grid.Insert(5000, 5000, EntityRef{Kind: 'p', Idx: 1})
	results = grid.Query(WorldWidth, WorldHeight, 50)
	found = false
	for _, r := range results {
		if r.Kind == 'p' && r.Idx == 1 {
			found = true
		}
	}
	if !found {
		t.Error("expected to find entity inserted beyond world edge")
	}
}
