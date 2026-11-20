package svg

import (
	"github.com/tinychameleon/blocky/strategy"
	"image/color"
)

type Option func(*svg)

func DebugMode(s *svg) {
	s.debug = true
}

func KeepInvisible(s *svg) {
	s.keepInvisible = true
}

func Exclude(c color.RGBA) Option {
	return func(s *svg) {
		s.exclude = &c
	}
}

func Strategy(f strategy.Func) Option {
	return func(s *svg) {
		s.strategy = f
	}
}

// TileReuse enables tiled output: the image is partitioned into size by
// size squares, each unique tile is emitted once as an SVG symbol and
// remaining occurrences reference it through use elements.
func TileReuse(size int) Option {
	return func(s *svg) {
		s.tileSize = size
	}
}

// TileAtlasLimit caps the number of unique tile symbols emitted in tiled
// output. When the limit is exceeded the SVG falls back to the classic
// per-shape output. A value of zero keeps the built-in default cap and a
// negative value disables the cap entirely.
func TileAtlasLimit(n int) Option {
	return func(s *svg) {
		s.atlasLimit = n
	}
}
