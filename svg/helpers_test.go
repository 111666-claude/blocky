package svg

import (
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"testing"
)

// writeTempPNG writes a small valid PNG and returns its path.
func writeTempPNG(t *testing.T) string {
	t.Helper()
	f, err := ioutil.TempFile("", "blocky-*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			m.SetRGBA(x, y, color.RGBA{R: uint8(x * 60), G: uint8(y * 60), B: 10, A: 255})
		}
	}
	if err := png.Encode(f, m); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}
