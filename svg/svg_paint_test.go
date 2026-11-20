package svg

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
)

// Test-only SVG parser and painter. blocky generates a tiny, fixed SVG
// dialect, so the painter only needs to understand its symbol/use/rect
// elements to verify that tiled output renders the same pixels as the
// classic per-shape output without pulling in an external rasterizer.

type svgDocument struct {
	XMLName xml.Name    `xml:"svg"`
	ViewBox string      `xml:"viewBox,attr"`
	Symbols []svgSymbol `xml:"defs>symbol"`
	Rects   []svgRect   `xml:"rect"`
	Uses    []svgUse    `xml:"use"`
}

type svgSymbol struct {
	ID     string    `xml:"id,attr"`
	Width  string    `xml:"width,attr"`
	Height string    `xml:"height,attr"`
	Rects  []svgRect `xml:"rect"`
}

type svgRect struct {
	X       string `xml:"x,attr"`
	Y       string `xml:"y,attr"`
	Width   string `xml:"width,attr"`
	Height  string `xml:"height,attr"`
	Fill    string `xml:"fill,attr"`
	Opacity string `xml:"fill-opacity,attr"`
}

type svgUse struct {
	Href   string `xml:"href,attr"`
	X      string `xml:"x,attr"`
	Y      string `xml:"y,attr"`
	Width  string `xml:"width,attr"`
	Height string `xml:"height,attr"`
}

func parseSVG(data string) (svgDocument, error) {
	var doc svgDocument
	if err := xml.Unmarshal([]byte(strings.TrimSpace(data)), &doc); err != nil {
		return doc, fmt.Errorf("parse svg: %w", err)
	}
	return doc, nil
}

// paintSVG renders a parsed blocky SVG into an NRGBA image; untouched
// pixels remain fully transparent.
func paintSVG(doc svgDocument) (*image.NRGBA, error) {
	fields := strings.Fields(doc.ViewBox)
	if len(fields) != 4 {
		return nil, fmt.Errorf("bad viewBox %q", doc.ViewBox)
	}
	width, err := strconv.Atoi(fields[2])
	if err != nil {
		return nil, err
	}
	height, err := strconv.Atoi(fields[3])
	if err != nil {
		return nil, err
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for _, r := range doc.Rects {
		if err := paintRect(img, r, 0, 0, image.Rectangle{}); err != nil {
			return nil, err
		}
	}

	symbols := make(map[string]svgSymbol, len(doc.Symbols))
	for _, sym := range doc.Symbols {
		symbols[sym.ID] = sym
	}
	for _, u := range doc.Uses {
		id := strings.TrimPrefix(u.Href, "#")
		sym, ok := symbols[id]
		if !ok {
			return nil, fmt.Errorf("use references unknown symbol %q", id)
		}
		ux, err := strconv.Atoi(u.X)
		if err != nil {
			return nil, err
		}
		uy, err := strconv.Atoi(u.Y)
		if err != nil {
			return nil, err
		}
		uw, err := strconv.Atoi(u.Width)
		if err != nil {
			return nil, err
		}
		uh, err := strconv.Atoi(u.Height)
		if err != nil {
			return nil, err
		}
		// A use of a symbol establishes a viewport: content outside it
		// is clipped, matching how browsers render symbol references.
		clip := image.Rect(ux, uy, ux+uw, uy+uh)
		for _, r := range sym.Rects {
			if err := paintRect(img, r, ux, uy, clip); err != nil {
				return nil, err
			}
		}
	}
	return img, nil
}

func paintRect(img *image.NRGBA, r svgRect, offsetX, offsetY int, clip image.Rectangle) error {
	x, err := strconv.Atoi(r.X)
	if err != nil {
		return err
	}
	y, err := strconv.Atoi(r.Y)
	if err != nil {
		return err
	}
	w, err := strconv.Atoi(r.Width)
	if err != nil {
		return err
	}
	h, err := strconv.Atoi(r.Height)
	if err != nil {
		return err
	}
	clr, err := parseFill(r.Fill)
	if err != nil {
		return err
	}
	alpha := uint8(255)
	if r.Opacity != "" {
		o, err := strconv.ParseFloat(r.Opacity, 64)
		if err != nil {
			return err
		}
		alpha = uint8(o*255 + 0.5)
	}
	clr.A = alpha

	for py := y + offsetY; py < y+offsetY+h; py++ {
		for px := x + offsetX; px < x+offsetX+w; px++ {
			if !clip.Empty() && !image.Pt(px, py).In(clip) {
				continue
			}
			img.SetNRGBA(px, py, clr)
		}
	}
	return nil
}

func parseFill(s string) (color.NRGBA, error) {
	if !strings.HasPrefix(s, "#") || len(s) != 7 {
		return color.NRGBA{}, fmt.Errorf("bad fill %q", s)
	}
	r, err1 := strconv.ParseUint(s[1:3], 16, 8)
	g, err2 := strconv.ParseUint(s[3:5], 16, 8)
	b, err3 := strconv.ParseUint(s[5:7], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return color.NRGBA{}, fmt.Errorf("bad fill %q", s)
	}
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b)}, nil
}

// paint renders a raw blocky SVG string in one step.
func paint(data string) (*image.NRGBA, error) {
	doc, err := parseSVG(data)
	if err != nil {
		return nil, err
	}
	return paintSVG(doc)
}
