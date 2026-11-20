package tile

import (
	"image"
	"image/color"
	"testing"

	"github.com/tinychameleon/blocky/grid"
)

// makeGrid builds a width by height grid; cell(x,y) returning false means
// the cell is erased, matching how svg.String hides transparent pixels.
func makeGrid(width, height int, cell func(x, y int) (color.RGBA, bool)) grid.Grid {
	m := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c, ok := cell(x, y)
			if ok {
				m.SetRGBA(x, y, c)
			}
		}
	}
	g := grid.New(m)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if _, ok := cell(x, y); !ok {
				g.EraseAt(x, y)
			}
		}
	}
	return g
}

func solid(c color.RGBA) func(x, y int) (color.RGBA, bool) {
	return func(x, y int) (color.RGBA, bool) {
		return c, true
	}
}

func TestBuildReusesIdenticalTiles(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	blue := color.RGBA{0, 0, 0xff, 0xff}
	// Repeating brick wall: 16x16 image, 8x8 tiles. The four quadrants
	// alternate between two patterns, so only two symbols are needed.
	g := makeGrid(16, 16, func(x, y int) (color.RGBA, bool) {
		if (x/8+y/8)%2 == 0 {
			return red, true
		}
		return blue, true
	})

	plan, err := Build(g, 8, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(plan.Symbols) != 2 {
		t.Errorf("Got %d symbols, want 2", len(plan.Symbols))
	}
	if len(plan.Placements) != 4 {
		t.Errorf("Got %d placements, want 4", len(plan.Placements))
	}
	if plan.Symbols[0].ID != "tile-0" || plan.Symbols[1].ID != "tile-1" {
		t.Errorf("Got IDs %q, %q; want tile-0, tile-1",
			plan.Symbols[0].ID, plan.Symbols[1].ID)
	}
	// Row-major first-seen ordering: top-left pattern wins tile-0.
	wantPositions := [][2]int{{0, 0}, {8, 0}, {0, 8}, {8, 8}}
	for i, p := range plan.Placements {
		if p.X != wantPositions[i][0] || p.Y != wantPositions[i][1] {
			t.Errorf("Placement %d: got (%d,%d), want (%d,%d)",
				i, p.X, p.Y, wantPositions[i][0], wantPositions[i][1])
		}
	}
	wantIDs := []string{"tile-0", "tile-1", "tile-1", "tile-0"}
	for i, p := range plan.Placements {
		if p.ID != wantIDs[i] {
			t.Errorf("Placement %d: got %s, want %s", i, p.ID, wantIDs[i])
		}
	}
}

func TestBuildIsStableAcrossRuns(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	blue := color.RGBA{0, 0, 0xff, 0xff}
	g := makeGrid(8, 8, func(x, y int) (color.RGBA, bool) {
		if x < 4 {
			return red, true
		}
		return blue, true
	})

	first, err := Build(g, 4, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	second, err := Build(g, 4, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(first.Symbols) != len(second.Symbols) {
		t.Fatalf("Symbol counts differ: %d vs %d",
			len(first.Symbols), len(second.Symbols))
	}
	for i := range first.Symbols {
		if first.Symbols[i].ID != second.Symbols[i].ID {
			t.Errorf("Symbol %d ID changed: %q vs %q",
				i, first.Symbols[i].ID, second.Symbols[i].ID)
		}
	}
	for i := range first.Placements {
		if first.Placements[i] != second.Placements[i] {
			t.Errorf("Placement %d changed: %v vs %v",
				i, first.Placements[i], second.Placements[i])
		}
	}
}

func TestBuildSkipsFullyTransparentTiles(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	// Left 8x8 tile is entirely erased; right one is solid red.
	g := makeGrid(16, 8, func(x, y int) (color.RGBA, bool) {
		if x < 8 {
			return color.RGBA{}, false
		}
		return red, true
	})

	plan, err := Build(g, 8, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(plan.Symbols) != 1 {
		t.Errorf("Got %d symbols, want 1", len(plan.Symbols))
	}
	if len(plan.Placements) != 1 {
		t.Errorf("Got %d placements, want 1", len(plan.Placements))
	}
	if p := plan.Placements[0]; p.X != 8 || p.Y != 0 {
		t.Errorf("Got placement (%d,%d), want (8,0)", p.X, p.Y)
	}
}

func TestBuildPreservesTransparencyInsideTiles(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	// Each tile has an erased pixel at the same local position; the
	// symbol must omit that pixel rather than paint over it.
	g := makeGrid(8, 8, func(x, y int) (color.RGBA, bool) {
		if x%4 == y%4 {
			return color.RGBA{}, false
		}
		return red, true
	})

	plan, err := Build(g, 4, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(plan.Symbols) != 1 {
		t.Fatalf("Got %d symbols, want 1", len(plan.Symbols))
	}
	if got := len(plan.Symbols[0].Shapes); got != 12 {
		t.Errorf("Got %d symbol shapes, want 12 (16 cells minus 4 erased)", got)
	}
}

func TestBuildHandlesNonDivisibleDimensions(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	g := makeGrid(10, 6, solid(red))

	plan, err := Build(g, 8, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if plan.Width != 10 || plan.Height != 6 {
		t.Errorf("Got plan bounds %dx%d, want 10x6", plan.Width, plan.Height)
	}
	if len(plan.Placements) != 2 {
		t.Errorf("Got %d placements, want 2", len(plan.Placements))
	}
	// Edge tiles carry their real, smaller dimensions.
	edge := plan.Symbols[0]
	if edge.Width != 8 || edge.Height != 6 {
		t.Errorf("Got origin tile %dx%d, want 8x6", edge.Width, edge.Height)
	}
	last := plan.Placements[1]
	if last.X != 8 || last.Y != 0 {
		t.Errorf("Got last placement (%d,%d), want (8,0)", last.X, last.Y)
	}
}

func TestBuildWithTinyTiles(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	blue := color.RGBA{0, 0, 0xff, 0xff}
	g := makeGrid(2, 2, func(x, y int) (color.RGBA, bool) {
		if x == y {
			return red, true
		}
		return blue, true
	})

	plan, err := Build(g, 1, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(plan.Symbols) != 2 {
		t.Errorf("Got %d symbols, want 2", len(plan.Symbols))
	}
	if len(plan.Placements) != 4 {
		t.Errorf("Got %d placements, want 4", len(plan.Placements))
	}
	for _, sym := range plan.Symbols {
		if sym.Width != 1 || sym.Height != 1 {
			t.Errorf("Got symbol size %dx%d, want 1x1", sym.Width, sym.Height)
		}
	}
}

func TestBuildTileLargerThanImage(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	g := makeGrid(3, 2, solid(red))

	plan, err := Build(g, 16, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(plan.Symbols) != 1 {
		t.Errorf("Got %d symbols, want 1", len(plan.Symbols))
	}
	sym := plan.Symbols[0]
	if sym.Width != 3 || sym.Height != 2 {
		t.Errorf("Got symbol size %dx%d, want 3x2", sym.Width, sym.Height)
	}
}

func TestBuildRejectsBadTileSize(t *testing.T) {
	g := makeGrid(4, 4, solid(color.RGBA{}))
	for _, size := range []int{0, -1, -10} {
		if _, err := Build(g, size, 0); err != ErrBadTileSize {
			t.Errorf("Size %d: got %v, want ErrBadTileSize", size, err)
		}
	}
}

func TestBuildRejectsNegativeAtlasLimit(t *testing.T) {
	g := makeGrid(4, 4, solid(color.RGBA{1, 2, 3, 255}))
	if _, err := Build(g, 2, -1); err != ErrBadAtlasLimit {
		t.Errorf("Got %v, want ErrBadAtlasLimit", err)
	}
}

func TestBuildEnforcesAtlasLimit(t *testing.T) {
	// Every 2x2 tile is unique, giving 9 symbols over a 6x6 image.
	g := makeGrid(6, 6, func(x, y int) (color.RGBA, bool) {
		return color.RGBA{uint8(x * 17), uint8(y * 17), 0, 0xff}, true
	})

	if _, err := Build(g, 2, 9); err != nil {
		t.Errorf("Limit 9: got %v, want no error", err)
	}
	if _, err := Build(g, 2, 8); err != ErrAtlasLimit {
		t.Errorf("Limit 8: got %v, want ErrAtlasLimit", err)
	}
	if _, err := Build(g, 2, 1); err != ErrAtlasLimit {
		t.Errorf("Limit 1: got %v, want ErrAtlasLimit", err)
	}
}

func TestBuildDistinguishesAlphaPalettes(t *testing.T) {
	// Two tiles with identical RGB but different alpha are palette
	// conflicts and must not share a symbol.
	opaque := color.RGBA{0xff, 0, 0, 0xff}
	half := color.RGBA{0xff, 0, 0, 0x80}
	g := makeGrid(8, 4, func(x, y int) (color.RGBA, bool) {
		if x < 4 {
			return opaque, true
		}
		return half, true
	})

	plan, err := Build(g, 4, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(plan.Symbols) != 2 {
		t.Errorf("Got %d symbols, want 2 (alpha differs)", len(plan.Symbols))
	}
	if len(plan.Placements) != 2 {
		t.Errorf("Got %d placements, want 2", len(plan.Placements))
	}
}

func TestBuildEdgeTilesOfDifferentShapeDoNotCollide(t *testing.T) {
	// 10x10 image, 8x8 tiles. The right edge tiles are 2x8 and the bottom
	// edge tiles are 8x2; a solid colour makes their flattened cells
	// identical, but their shapes differ so they must not share a symbol.
	red := color.RGBA{0xff, 0, 0, 0xff}
	g := makeGrid(10, 10, solid(red))

	plan, err := Build(g, 8, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	// Four tiles: 8x8, 2x8, 8x2, 2x2 — all distinct shapes.
	if len(plan.Symbols) != 4 {
		t.Errorf("Got %d symbols, want 4 (shapes must not collide)",
			len(plan.Symbols))
	}
	for _, p := range plan.Placements {
		var sym Symbol
		for _, s := range plan.Symbols {
			if s.ID == p.ID {
				sym = s
				break
			}
		}
		if sym.Width != p.Width || sym.Height != p.Height {
			t.Errorf("Placement (%d,%d) uses %dx%d symbol but is %dx%d",
				p.X, p.Y, sym.Width, sym.Height, p.Width, p.Height)
		}
	}
}

func TestBuildDistinguishesErasedFromTransparent(t *testing.T) { // An erased cell must not match a stored fully transparent colour;
	// otherwise a kept-invisible tile would silently alias an empty one.
	transparent := color.RGBA{0xff, 0, 0, 0x00}
	red := color.RGBA{0xff, 0, 0, 0xff}
	g := makeGrid(4, 4, func(x, y int) (color.RGBA, bool) {
		switch x {
		case 0:
			return color.RGBA{}, false
		case 1:
			return red, true
		case 2:
			return transparent, true
		default:
			return red, true
		}
	})

	plan, err := Build(g, 2, 0)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	// Left tiles pair an erased cell with red, right tiles pair a stored
	// transparent cell with red: same RGB layout, different identity.
	if len(plan.Symbols) != 2 {
		t.Errorf("Got %d symbols, want 2", len(plan.Symbols))
	}
	if len(plan.Placements) != 4 {
		t.Errorf("Got %d placements, want 4", len(plan.Placements))
	}
}
