// Package tile provides tile-based reuse planning for SVG output.
//
// A grid is partitioned into square tiles of a configurable size and each
// unique non-empty tile is described exactly once as an SVG symbol; every
// other occurrence only references that symbol through a use element.
package tile

import (
	"errors"
	"fmt"

	"github.com/tinychameleon/blocky/grid"
	"github.com/tinychameleon/blocky/shape"
)

// ErrBadTileSize is returned when the configured tile size is not positive.
var ErrBadTileSize = errors.New("tile size must be greater than zero")

// ErrBadAtlasLimit is returned when the atlas limit is negative.
var ErrBadAtlasLimit = errors.New("atlas limit must not be negative")

// ErrAtlasLimit is returned when a tiled plan would contain more unique
// symbols than the configured atlas limit.
var ErrAtlasLimit = errors.New("unique tile count exceeds atlas limit")

// Symbol describes a single unique tile, defined once in the SVG defs.
// Shapes use tile-local coordinates and empty (transparent) cells are
// omitted so transparency is preserved inside reused tiles.
type Symbol struct {
	ID     string
	Width  int
	Height int
	Shapes []shape.Interface
}

// Placement is a single use of a Symbol positioned at image coordinates.
// Width and Height are the actual rendered tile size, which can be smaller
// than TileSize for the partial tiles on the right and bottom edges.
type Placement struct {
	ID     string
	X      int
	Y      int
	Width  int
	Height int
}

// Plan is the result of tiling a grid.
type Plan struct {
	TileSize   int
	Width      int
	Height     int
	Symbols    []Symbol
	Placements []Placement
}

// Build partitions g into tileSize-sized squares, recording each unique
// non-empty tile exactly once. Tiles are visited in row-major order and
// symbol IDs are assigned on first use, so the plan is stable across runs
// of the same input.
//
// Tiles smaller than tileSize are emitted for image dimensions that are
// not divisible by tileSize, and fully transparent tiles produce no
// symbol or placement. Exact RGBA values participate in tile identity,
// so tiles sharing RGB values but differing in alpha never collide.
//
// An atlasLimit greater than zero caps the number of unique symbols;
// once the cap is exceeded Build returns ErrAtlasLimit. A limit of zero
// means no cap.
func Build(g grid.Grid, tileSize, atlasLimit int) (Plan, error) {
	if tileSize <= 0 {
		return Plan{}, ErrBadTileSize
	}
	if atlasLimit < 0 {
		return Plan{}, ErrBadAtlasLimit
	}

	width, height := g.Bounds()
	cols := (width + tileSize - 1) / tileSize
	rows := (height + tileSize - 1) / tileSize

	ids := make(map[string]string)
	symbols := make(map[string]Symbol)
	order := make([]string, 0)
	plan := Plan{TileSize: tileSize, Width: width, Height: height}

	for ty := 0; ty < rows; ty++ {
		for tx := 0; tx < cols; tx++ {
			x0 := tx * tileSize
			y0 := ty * tileSize
			tw := minInt(tileSize, width-x0)
			th := minInt(tileSize, height-y0)

			// The key starts with the tile dimensions so two tiles whose
			// flattened cells coincide but whose shapes differ (for example
			// a 2x8 and an 8x2 edge tile) never share a symbol. A presence
			// byte plus four RGBA bytes per cell keeps the key exact:
			// erased (nil) cells never match stored colours, including a
			// stored fully transparent black.
			key := make([]byte, 0, 4+tw*th*5)
			key = append(key, byte(tw>>8), byte(tw), byte(th>>8), byte(th))
			cells := make([]shape.Interface, 0, tw*th)
			empty := true
			for dy := 0; dy < th; dy++ {
				for dx := 0; dx < tw; dx++ {
					c := g.At(x0+dx, y0+dy)
					if c == nil {
						key = append(key, 0, 0, 0, 0, 0)
						continue
					}
					empty = false
					key = append(key, 1, c.R, c.G, c.B, c.A)
					cells = append(cells, shape.Pixel(dx, dy, *c))
				}
			}
			if empty {
				continue
			}

			k := string(key)
			id, ok := ids[k]
			if !ok {
				id = fmt.Sprintf("tile-%d", len(order))
				ids[k] = id
				order = append(order, k)
				if atlasLimit > 0 && len(order) > atlasLimit {
					return Plan{}, ErrAtlasLimit
				}
				symbols[k] = Symbol{ID: id, Width: tw, Height: th, Shapes: cells}
			}
			plan.Placements = append(plan.Placements, Placement{
				ID:     id,
				X:      x0,
				Y:      y0,
				Width:  tw,
				Height: th,
			})
		}
	}

	plan.Symbols = make([]Symbol, 0, len(order))
	for _, k := range order {
		plan.Symbols = append(plan.Symbols, symbols[k])
	}
	return plan, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
