package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tinychameleon/blocky/shape"
	"github.com/tinychameleon/blocky/strategy"
	"github.com/tinychameleon/blocky/svg"
)

type cmdFlags struct {
	debug      bool
	keep       bool
	exclude    string
	strategy   string
	tileSize   int
	atlasLimit int
}

func main() {
	var flags cmdFlags
	flag.BoolVar(&flags.debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&flags.keep, "keepInvisible", false,
		"Output elements that have 0x00 alpha values")
	flag.StringVar(&flags.exclude, "exclude", "",
		"Exclude #RRGGBB[AA] colour from output")
	flag.StringVar(&flags.strategy, "optimize", "pixels",
		"Optimize SVG rectangle output. Values: pixels")
	flag.IntVar(&flags.tileSize, "tiles", 0,
		"Reuse NxN pixel tiles through SVG symbol/use elements (0 disables)")
	flag.IntVar(&flags.atlasLimit, "tileAtlasLimit", 0,
		"Maximum unique tile symbols before falling back to normal output (0 uses the built-in default, -1 disables the cap)")
	flag.Usage = usage
	flag.Parse()

	options, err := fromFlags(flags)
	if err != nil {
		exit("Bad option:", err)
	}

	s, err := svg.New(flag.Arg(0), options...)
	if err != nil {
		exit("Bad input file:", err)
	}
	fmt.Println(s)
}

func fromFlags(flags cmdFlags) ([]svg.Option, error) {
	var o []svg.Option

	if flags.debug {
		o = append(o, svg.DebugMode)
	}

	if flags.keep {
		o = append(o, svg.KeepInvisible)
	}

	if flags.exclude != "" {
		c, err := shape.ParseHex(flags.exclude)
		if err != nil {
			return nil, err
		}
		o = append(o, svg.Exclude(c))
	}

	f := strategy.Pixels
	if flags.strategy == "rects" {
		f = strategy.Rects
	}
	o = append(o, svg.Strategy(f))

	if flags.tileSize < 0 {
		return nil, fmt.Errorf("-tiles must be greater than zero")
	}
	if flags.atlasLimit < -1 {
		return nil, fmt.Errorf("-tileAtlasLimit must be -1, 0, or a positive count")
	}
	if flags.tileSize > 0 {
		o = append(o, svg.TileReuse(flags.tileSize))
		if flags.atlasLimit != 0 {
			o = append(o, svg.TileAtlasLimit(flags.atlasLimit))
		}
	}

	return o, nil
}

func exit(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func usage() {
	fmt.Printf("Usage: %s [-debug] [-keepInvisible] [-exclude=#RRGGBB[AA]] [-optimize=pixels|rects] [-tiles=N] [-tileAtlasLimit=N] FILE\n",
		os.Args[0])
	flag.PrintDefaults()
}
