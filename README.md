# Blocky
Converts pixel art image files into SVGs for the web.

## Usage
Using `blocky` is pretty simple: you give it a PNG file and it writes the SVG
data to standard output.
```
$ blocky [-debug] [-keepInvisible] [-exclude=#RRGGBB[AA]] [-optimize=pixels|rects] [-tiles=N] [-tileAtlasLimit=N] FILE
```
The flags `blocky` accepts mean the following:

| Flag                   | Default | Explanation                                  |
| ---------------------- | ------- | -------------------------------------------- |
| `-debug`               | Off     | Enables debug mode output                    |
| `-keepInvisible`       | Off     | Write elements that have `0x00` alpha values |
| `-exclude=#RRGGBB[AA]` | Off     | Excludes the given colour from output. Alpha is `0xFF` when unspecified. |
| `-optimize`            | Off     | Optimize rectangles in the SVG               |
| `-tiles=N`             | Off (`0`) | Reuse `N`x`N` pixel tiles through SVG `symbol`/`use` elements |
| `-tileAtlasLimit=N`    | `10000` | Unique symbols allowed before falling back to normal output; `-1` disables the cap |

### Optimize Flag Values
The optimize flag supports the following modes:

| Mode     | Description                                 |
| -------- | ------------------------------------------- |
| Off      | No optimization. Defaults to `pixels`       |
| `pixels` | Emits individual pixels                     |
| `rects`  | Emits rectangles for contiguous colours     |

### Tile Reuse
Maps with repeating textures (brick walls, terrain, UI tiles) otherwise emit
the same paths thousands of times. With `-tiles=N`, blocky partitions the
image into `N` by `N` squares, emits each unique square once inside
`<defs><symbol>…</symbol></defs>` and places every occurrence with a single
`<use>` element:

```
$ blocky -tiles=8 artwork.png > artwork.svg
```

Tile identity covers the exact RGBA contents, so transparent cells are
preserved inside symbols and two tiles that share RGB values but differ in
alpha never collide. Fully transparent tiles emit nothing.

Edge handling and safeguards:

- Images whose dimensions are not divisible by `N` get smaller edge tiles on
  the right and bottom; the original image size and `viewBox` are unchanged.
- Symbol IDs are assigned in row-major first-seen order (`tile-0`, `tile-1`,
  …), so output is stable between runs of the same image.
- Tiles of different dimensions never share a symbol, even when their pixel
  contents look identical after flattening.
- `-tileAtlasLimit` bounds the number of unique symbols. When an image would
  exceed it — for example a noisy texture producing one unique tile per cell —
  blocky silently falls back to the normal per-shape output, so the SVG can
  never grow an unbounded `defs` section. `-tileAtlasLimit=-1` removes the
  cap.
- With tiling disabled (the default) the generated SVG is byte-for-byte the
  same as before.

On a repeating 128×128 brick image with `-tiles=8` the output drops from
~955 KB to ~22 KB raw and ~42 KB to ~1.3 KB gzipped while rendering the same
pixels (256 `use` elements backed by 2 symbols).

### Example
To run in debug mode, while converting `artwork.png` into `logo.svg`:
```
$ blocky -debug artwork.png > logo.svg
```

## Development
Requires Go 1.13 to build and has no external dependencies.

### Testing
The default action for testing is to run all tests:
```
$ make test
```
Set the `SPEC` variable to target a specific package during testing:
```
$ make test SPEC=./svg
```

### Debug Mode
When the `-debug` flag is passed, the resulting SVG will have stroke styling
embedded into it so that the rectangle outputs can be distinguished:
```
$ make build
$ ./blocky -debug image.png > image.svg
```

## SVG Resouce Sizes
This tool was created to turn pixel art PNG files into SVGs for website use, so
it's important to keep track of how the generated SVG elements will affect page
sizes.

The test case is a `455B`, `25x20` PNG file; these numbers will most likely differ
based on PNG file, but it should still be a decent ballpark.

| Version | Description              | Size (B) |  ΔSize | GZ (B) |  ΔGZ | ~Ratio (%) |
| ------- | ------------------------ | -------: | -----: | -----: | ---: | ---------: |
| v0.1    | Pixel-for-pixel output   |   33,411 |      – |  1,649 |    – | 20:1 (95%) |
| v0.2    | Alpha & colour exclusion |   12,486 |   ↓63% |    835 | ↓50% | 15:1 (93%) |
| v0.3    | Rectangle optimization   |    6,610 |   ↓47% |    649 | ↓22% | 10:1 (90%) |

Currently, the generated SVG is `1.4x` larger than the source image. There are
other opportunities for further size optimizations:

- Organizing output to take better advantage of compression
- Looking for the largest NxM rectangle instead of Nx1 and 1xN
- Finding colour islands and pushing them into a higher z-index

However, the size increase over the source image is acceptable to me for now,
since the source image isn't actually delivered to the clients. If we look at
linear scaling of the source image to some realistic resolutions, the current
SVG size is adequate:

| WxH     |  Size (B) |
| ------- |  -------: |
| SVG     |       649 |
| 25x20   |       455 |
| 75x60   |     1,365 |
| 150x120 |     2,730 |
