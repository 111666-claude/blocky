package svg

import (
	"image"
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/tinychameleon/blocky/grid"
	"github.com/tinychameleon/blocky/strategy"
)

// testGrid builds an NRGBA-backed grid, erasing zero-alpha cells the same
// way svg.String hides transparent pixels by default.
func testGrid(width, height int, cell func(x, y int) (color.RGBA, bool)) grid.Grid {
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

var (
	brickRed  = color.RGBA{0xc0, 0x39, 0x2b, 0xff}
	mortar    = color.RGBA{0x7f, 0x7f, 0x7f, 0xff}
	skyBlue   = color.RGBA{0x3b, 0x9e, 0xe1, 0xff}
	leafGreen = color.RGBA{0x2e, 0x7d, 0x32, 0xff}
)

// classicSvg renders the grid through the normal pixel strategy.
func classicSvg(g grid.Grid, extra ...Option) string {
	s := svg{g: g, strategy: strategy.Pixels, atlasLimit: defaultAtlasLimit}
	for _, o := range extra {
		o(&s)
	}
	return s.String()
}

// tiledSvg renders the grid through the symbol/use path.
func tiledSvg(g grid.Grid, tileSize int, extra ...Option) string {
	s := svg{
		g:          g,
		strategy:   strategy.Pixels,
		tileSize:   tileSize,
		atlasLimit: defaultAtlasLimit,
	}
	for _, o := range extra {
		o(&s)
	}
	return s.String()
}

func assertSamePixels(t *testing.T, want, got *image.NRGBA) {
	t.Helper()
	if want.Bounds() != got.Bounds() {
		t.Fatalf("Bounds differ: %v vs %v", want.Bounds(), got.Bounds())
	}
	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			w := want.NRGBAAt(x, y)
			g := got.NRGBAAt(x, y)
			// Alpha conversion can round by one; compare within tolerance.
			if absInt(int(w.R)-int(g.R)) > 1 ||
				absInt(int(w.G)-int(g.G)) > 1 ||
				absInt(int(w.B)-int(g.B)) > 1 ||
				absInt(int(w.A)-int(g.A)) > 1 {
				t.Errorf("Pixel (%d,%d): got %v, want %v", x, y, g, w)
			}
		}
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestTiledBrickWallUseCount(t *testing.T) {
	// 16x16 wall of 8x8 bricks: two distinct bricks, each used twice.
	g := testGrid(16, 16, func(x, y int) (color.RGBA, bool) {
		if (x/8+y/8)%2 == 0 {
			return brickRed, true
		}
		return mortar, true
	})

	out := tiledSvg(g, 8)
	doc, err := parseSVG(out)
	if err != nil {
		t.Fatal(err)
	}
	if doc.ViewBox != "0 0 16 16" {
		t.Errorf("Got viewBox %q, want 0 0 16 16", doc.ViewBox)
	}
	if len(doc.Symbols) != 2 {
		t.Errorf("Got %d symbols, want 2", len(doc.Symbols))
	}
	if len(doc.Uses) != 4 {
		t.Errorf("Got %d use elements, want 4", len(doc.Uses))
	}
	if len(doc.Rects) != 0 {
		t.Errorf("Got %d top-level rects, want 0 (all inside symbols)", len(doc.Rects))
	}
	if doc.Symbols[0].ID != "tile-0" || doc.Symbols[1].ID != "tile-1" {
		t.Errorf("Got symbol IDs %q, %q; want tile-0, tile-1",
			doc.Symbols[0].ID, doc.Symbols[1].ID)
	}
	// Each symbol body is defined once with tile-local coordinates.
	for i, sym := range doc.Symbols {
		if len(sym.Rects) != 64 {
			t.Errorf("Symbol %d: got %d rects, want 64", i, len(sym.Rects))
		}
	}
	wantUses := []struct{ href, x, y string }{
		{"#tile-0", "0", "0"},
		{"#tile-1", "8", "0"},
		{"#tile-1", "0", "8"},
		{"#tile-0", "8", "8"},
	}
	for i, u := range doc.Uses {
		if u.Href != wantUses[i].href || u.X != wantUses[i].x || u.Y != wantUses[i].y {
			t.Errorf("Use %d: got (%s at %s,%s), want (%s at %s,%s)",
				i, u.Href, u.X, u.Y,
				wantUses[i].href, wantUses[i].x, wantUses[i].y)
		}
	}
}

func TestTiledBrickWallRenderedPixels(t *testing.T) {
	g := testGrid(16, 16, func(x, y int) (color.RGBA, bool) {
		if (x/8+y/8)%2 == 0 {
			return brickRed, true
		}
		return mortar, true
	})

	want, err := paint(classicSvg(g))
	if err != nil {
		t.Fatal(err)
	}
	got, err := paint(tiledSvg(g, 8))
	if err != nil {
		t.Fatal(err)
	}
	assertSamePixels(t, want, got)
}

func TestTiledNonDivisibleEdge(t *testing.T) {
	// 10x6 image with 8x8 tiles: one full tile and a 2x6 edge tile,
	// the original viewBox must be preserved exactly.
	g := testGrid(10, 6, func(x, y int) (color.RGBA, bool) {
		if x < 8 {
			return skyBlue, true
		}
		return leafGreen, true
	})

	out := tiledSvg(g, 8)
	doc, err := parseSVG(out)
	if err != nil {
		t.Fatal(err)
	}
	if doc.ViewBox != "0 0 10 6" {
		t.Errorf("Got viewBox %q, want 0 0 10 6", doc.ViewBox)
	}
	if len(doc.Uses) != 2 {
		t.Errorf("Got %d uses, want 2", len(doc.Uses))
	}
	if w := doc.Symbols[1].Width; w != "2" {
		t.Errorf("Edge symbol width = %q, want 2", w)
	}

	want, err := paint(classicSvg(g))
	if err != nil {
		t.Fatal(err)
	}
	got, err := paint(out)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePixels(t, want, got)
}

func TestTiledTransparentPixels(t *testing.T) {
	// Scattered erased cells, including a fully transparent tile.
	g := testGrid(16, 8, func(x, y int) (color.RGBA, bool) {
		if x < 8 {
			return color.RGBA{}, false
		}
		if (x+y)%3 == 0 {
			return color.RGBA{}, false
		}
		return brickRed, true
	})

	out := tiledSvg(g, 8)
	doc, err := parseSVG(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Uses) != 1 {
		t.Errorf("Got %d uses, want 1 (empty tile omitted)", len(doc.Uses))
	}

	want, err := paint(classicSvg(g))
	if err != nil {
		t.Fatal(err)
	}
	got, err := paint(out)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePixels(t, want, got)

	// A pixel inside the transparent region stays transparent.
	if px := got.NRGBAAt(2, 2); px.A != 0 {
		t.Errorf("Transparent region pixel = %v, want alpha 0", px)
	}
}

func TestTiledKeepInvisibleAlpha(t *testing.T) {
	// With keepInvisible, zero-alpha colours remain in the grid and must
	// survive the tiled path with a fill-opacity of 0.
	half := color.RGBA{0xff, 0x10, 0x20, 0x80}
	g := testGrid(8, 4, func(x, y int) (color.RGBA, bool) {
		if x < 4 {
			return color.RGBA{0, 0, 0, 0}, true
		}
		return half, true
	})

	out := tiledSvg(g, 4, KeepInvisible)
	doc, err := parseSVG(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Symbols) != 2 {
		t.Errorf("Got %d symbols, want 2", len(doc.Symbols))
	}
	var foundZeroOpacity bool
	for _, sym := range doc.Symbols {
		for _, r := range sym.Rects {
			if r.Opacity == "0" {
				foundZeroOpacity = true
			}
		}
	}
	if !foundZeroOpacity {
		t.Errorf("No fill-opacity=\"0\" rect found in symbols")
	}

	want, err := paint(classicSvg(g, KeepInvisible))
	if err != nil {
		t.Fatal(err)
	}
	got, err := paint(out)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePixels(t, want, got)
}

func TestTiledPaletteConflict(t *testing.T) {
	// Same RGB channels, different alpha: distinct symbols required.
	opaque := color.RGBA{0xff, 0, 0, 0xff}
	half := color.RGBA{0xff, 0, 0, 0x80}
	g := testGrid(8, 4, func(x, y int) (color.RGBA, bool) {
		if x < 4 {
			return opaque, true
		}
		return half, true
	})

	doc, err := parseSVG(tiledSvg(g, 4))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Symbols) != 2 {
		t.Errorf("Got %d symbols, want 2 (alpha conflict)", len(doc.Symbols))
	}
	if len(doc.Uses) != 2 {
		t.Errorf("Got %d uses, want 2", len(doc.Uses))
	}

	want, err := paint(classicSvg(g))
	if err != nil {
		t.Fatal(err)
	}
	got, err := paint(tiledSvg(g, 4))
	if err != nil {
		t.Fatal(err)
	}
	assertSamePixels(t, want, got)
}

func TestTiledTinySizeRendersIdentically(t *testing.T) {
	g := testGrid(5, 3, func(x, y int) (color.RGBA, bool) {
		if x == y {
			return brickRed, true
		}
		return mortar, true
	})

	want, err := paint(classicSvg(g))
	if err != nil {
		t.Fatal(err)
	}
	got, err := paint(tiledSvg(g, 1))
	if err != nil {
		t.Fatal(err)
	}
	assertSamePixels(t, want, got)
}

func TestTiledExcludeColour(t *testing.T) {
	g := testGrid(8, 8, func(x, y int) (color.RGBA, bool) {
		if x < 4 {
			return brickRed, true
		}
		return mortar, true
	})

	opt := Exclude(brickRed)
	want, err := paint(classicSvg(g, opt))
	if err != nil {
		t.Fatal(err)
	}
	got, err := paint(tiledSvg(g, 4, opt))
	if err != nil {
		t.Fatal(err)
	}
	assertSamePixels(t, want, got)
}

func TestTiledDisabledMatchesClassicOutput(t *testing.T) {
	g := testGrid(9, 9, func(x, y int) (color.RGBA, bool) {
		if x == y {
			return color.RGBA{}, false
		}
		return skyBlue, true
	})

	// Default construction leaves tiled output off, so the string is
	// byte-for-byte the classic output.
	classic := classicSvg(g)
	s := svg{g: g, strategy: strategy.Pixels, atlasLimit: defaultAtlasLimit}
	if got := s.String(); got != classic {
		t.Errorf("Default output differs from classic output")
	}

	// An explicit zero tile size also disables tiling.
	zero := tiledSvg(g, 0)
	if zero != classic {
		t.Errorf("tileSize 0 output differs from classic output")
	}
	if strings.Contains(zero, "<symbol") {
		t.Errorf("Disabled tiling still emits symbols")
	}
}

func TestTiledAtlasLimitFallsBack(t *testing.T) {
	// 6x6 image with 2x2 tiles gives 9 unique symbols.
	g := testGrid(6, 6, func(x, y int) (color.RGBA, bool) {
		return color.RGBA{uint8(x * 17), uint8(y * 17), 0x33, 0xff}, true
	})

	fallback := tiledSvg(g, 2, TileAtlasLimit(8))
	if strings.Contains(fallback, "<symbol") {
		t.Errorf("Exceeded atlas limit still emits symbols")
	}
	want, err := paint(classicSvg(g))
	if err != nil {
		t.Fatal(err)
	}
	got, err := paint(fallback)
	if err != nil {
		t.Fatal(err)
	}
	assertSamePixels(t, want, got)

	// A cap that fits keeps the tiled path.
	tiled := tiledSvg(g, 2, TileAtlasLimit(9))
	doc, err := parseSVG(tiled)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Symbols) != 9 {
		t.Errorf("Got %d symbols, want 9", len(doc.Symbols))
	}
}

func TestTiledUnlimitedAtlas(t *testing.T) {
	g := testGrid(6, 6, func(x, y int) (color.RGBA, bool) {
		return color.RGBA{uint8(x * 17), uint8(y * 17), 0x33, 0xff}, true
	})
	doc, err := parseSVG(tiledSvg(g, 2, TileAtlasLimit(-1)))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Symbols) != 9 {
		t.Errorf("Got %d symbols, want 9 with disabled cap", len(doc.Symbols))
	}
}

func TestNewRejectsTilingOptions(t *testing.T) {
	path := writeTempPNG(t)
	defer os.Remove(path)

	if _, err := New(path, TileReuse(-1)); err == nil {
		t.Errorf("TileReuse(-1): expected error")
	}
	if _, err := New(path, TileReuse(4), TileAtlasLimit(-2)); err == nil {
		t.Errorf("TileAtlasLimit(-2): expected error")
	}
	if _, err := New(path, TileReuse(4), TileAtlasLimit(0)); err != nil {
		t.Errorf("TileAtlasLimit(0): unexpected error %v", err)
	}
}

func TestTiledEmptyImageEmitsNoSymbols(t *testing.T) {
	g := testGrid(8, 8, func(x, y int) (color.RGBA, bool) {
		return color.RGBA{}, false
	})
	out := tiledSvg(g, 4)
	if strings.Contains(out, "<symbol") || strings.Contains(out, "<use") {
		t.Errorf("Fully transparent image emitted symbols: %s", out)
	}
	if !strings.Contains(out, `viewBox="0 0 8 8"`) {
		t.Errorf("Empty image lost its viewBox: %s", out)
	}
}
