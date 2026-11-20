// Package svg defines high-level operations for converting image files
// into SVG files.
package svg

import (
	"fmt"
	"github.com/tinychameleon/blocky/grid"
	"github.com/tinychameleon/blocky/strategy"
	"github.com/tinychameleon/blocky/tile"
	"image"
	"image/color"
	"io"
	"os"
	"strings"

	_ "image/png"
)

// defaultAtlasLimit bounds the number of unique tile symbols in tiled
// output so that pathological images cannot generate an unbounded defs
// section; exceeding it triggers classic per-shape output instead.
const defaultAtlasLimit = 10000

// New creates an instance of svg.Interface to convert an image
// into an SVG file.
func New(srcFilePath string, options ...Option) (Interface, error) {
	file, err := os.Open(srcFilePath)
	if err != nil {
		return nil, svgError(err)
	}
	defer file.Close()

	m, err := loadImage(file)
	if err != nil {
		return nil, svgError(err)
	}

	s := svg{g: grid.New(m), atlasLimit: defaultAtlasLimit}
	for _, opt := range options {
		opt(&s)
	}
	if s.tileSize < 0 {
		return nil, svgError(tile.ErrBadTileSize)
	}
	if s.atlasLimit < -1 {
		return nil, svgError(tile.ErrBadAtlasLimit)
	}
	return s, nil
}

func loadImage(r io.Reader) (image.Image, error) {
	m, _, err := image.Decode(r)
	return m, err
}

func svgError(e error) error {
	return fmt.Errorf("svg error: %w", e)
}

// Interface exists to explicitly define what can be done to SVG
// instance objects.
type Interface interface {
	fmt.Stringer
}

type svg struct {
	g             grid.Grid
	debug         bool
	keepInvisible bool
	exclude       *color.RGBA
	strategy      strategy.Func
	tileSize      int
	atlasLimit    int
}

func (s svg) String() string {
	if !s.keepInvisible {
		s.g.EraseMatching(func(c color.RGBA) bool {
			return c.A == 0x00
		})
	}

	if s.exclude != nil {
		s.g.EraseMatching(func(c color.RGBA) bool {
			return *s.exclude == c
		})
	}

	var b strings.Builder
	width, height := s.g.Bounds()
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d">`,
		width, height)
	fmt.Fprintf(&b, "\n")

	if s.debug {
		fmt.Fprintln(&b,
			"<style>rect { stroke-width: 0.01; stroke: #000 }</style>")
	}

	plan, tiled := s.tilePlan()
	if tiled {
		writeTiles(&b, plan)
	} else {
		for _, sh := range s.strategy(s.g) {
			fmt.Fprintln(&b, sh)
		}
	}

	fmt.Fprintf(&b, "</svg>")
	return b.String()
}

// tilePlan builds a reuse plan when tiled output is enabled. A plan that
// would exceed the atlas limit returns tiled=false so callers transparently
// fall back to the classic per-shape output.
func (s svg) tilePlan() (tile.Plan, bool) {
	if s.tileSize <= 0 {
		return tile.Plan{}, false
	}
	limit := s.atlasLimit
	if limit == 0 {
		limit = defaultAtlasLimit
	}
	if limit < 0 {
		limit = 0
	}
	plan, err := tile.Build(s.g, s.tileSize, limit)
	if err != nil {
		return tile.Plan{}, false
	}
	return plan, true
}

func writeTiles(b *strings.Builder, p tile.Plan) {
	if len(p.Symbols) == 0 {
		return
	}

	fmt.Fprintln(b, "<defs>")
	for _, sym := range p.Symbols {
		fmt.Fprintf(b, `<symbol id="%s" width="%d" height="%d">`+"\n",
			sym.ID, sym.Width, sym.Height)
		for _, sh := range sym.Shapes {
			fmt.Fprintln(b, sh)
		}
		fmt.Fprintln(b, "</symbol>")
	}
	fmt.Fprintln(b, "</defs>")

	for _, u := range p.Placements {
		fmt.Fprintf(b, `<use href="#%s" width="%d" height="%d" x="%d" y="%d"/>`+"\n",
			u.ID, u.Width, u.Height, u.X, u.Y)
	}
}
