package main

import "math"
import "unsafe"

type Triangle struct {
	x1, y1, x2, y2, x3, y3 int32
}

func renderTriangle(ptr unsafe.Pointer, w, h int32, ox, oy int32, t *Triangle, rgba uint32) {
	data := unsafe.Slice((*uint32)(ptr), w * h)
	color := (rgba >> 24) | ((rgba >> 8) & 0xff00) | ((rgba << 8) & 0xff0000) | (rgba << 24)

	x1 := ox + t.x1
	y1 := oy + t.y1
	x2 := ox + t.x2
	y2 := oy + t.y2
	x3 := ox + t.x3
	y3 := oy + t.y3

	xmin := min(max(min(min(x1, x2), x3), 0), w - 1)
	ymin := min(max(min(min(y1, y2), y3), 0), w - 1)
	xmax := min(max(max(max(x1, x2), x3), 0), w - 1)
	ymax := min(max(max(max(y1, y2), y3), 0), w - 1)

	for y := ymin; y <= ymax; y++ {
		for x := xmin; x <= xmax; x++ {
			if ((x2 - x1) * (y - y1) > (x - x1) * (y2 - y1) &&
				(x3 - x2) * (y - y2) > (x - x2) * (y3 - y2) &&
				(x1 - x3) * (y - y3) > (x - x3) * (y1 - y3)) {
					data[x + w * y] = color
			}
		}
	}
}

func roundTileEdges(ptr unsafe.Pointer, columns, rows, tileW, tileH, radius int) {
    data := unsafe.Slice((*uint32)(ptr), columns * tileW * rows * tileH)
    stride := columns * tileW

    radius = min(min(radius, tileW / 2), tileH / 2)
    for y := 0; y < radius; y++ {
        for x := 0; x < radius; x++ {
            distSq := (radius - x) * (radius - x) + (radius - y) * (radius - y)
            value := ((radius+1)*(radius+1) - distSq) * 16
            mask := (uint32(min(max(value, 0), 255)) << 24) | uint32(0xffffff)
            data[x+stride*y] &= mask
            data[(tileW-x-1)+stride*y] &= mask
            data[x+stride*(tileH-y-1)] &= mask
            data[(tileW-x-1)+stride*(tileH-y-1)] &= mask
        }
    }
    for r := 0; r < rows * tileH; r += tileH {
        for c := 0; c < columns * tileW; c += tileW {
            if r == 0 && c == 0 {
                continue
            }
            for y := 0; y < tileH; y++ {
                for x := 0; x < tileW; x++ {
                    idx := c+x + stride*(r+y)
                    data[idx] = (data[x + stride*y] & 0xff000000) | (data[idx] & 0xffffff)
                }
            }
        }
    }
}

func makeTileHighlight(ptr unsafe.Pointer, w, h, innerW, innerH int32, rgba uint32) {
    data := unsafe.Slice((*uint32)(ptr), w * h)
    color := (rgba >> 24) | ((rgba >> 8) & 0xff00) | ((rgba << 8) & 0xff0000) // no alpha

    xStart := (w - innerW) / 2
    yStart := (h - innerH) / 2
    xEnd := w - xStart
    yEnd := h - yStart

    for y := int32(0); y < h; y++ {
        for x := int32(0); x < w; x++ {
            alpha := uint32(0xa0)
            if x < xStart || x >= xEnd || y < yStart || y >= yEnd {
                alpha = uint32(min((255 / xStart) * (1 + min(min(x, w - x), min(y, h - y))), 255))
            }
            data[x + w * y] = (alpha << 24) | color
        }
    }
}

func makeTileCursor(ptr unsafe.Pointer, w, h int32, rgba uint32) {
    data := unsafe.Slice((*uint32)(ptr), w * h)
    color := (rgba >> 24) | ((rgba >> 8) & 0xff00) | ((rgba << 8) & 0xff0000) | (rgba << 24)

    borderWidth := w / 8
    innerRadiusF := float32(w) * 0.2
    innerRadius := int32(innerRadiusF)
    irsq := innerRadiusF * innerRadiusF
    midX := float32(w) * 0.5
    midY := float32(h) * 0.5

    for y := int32(0); y < h; y++ {
        for x := int32(0); x < w; x++ {
            alpha := 0.0
            distSq := (float32(x) - midX) * (float32(x) - midX) + (float32(y) - midY) * (float32(y) - midY)
            if distSq <= irsq {
                alpha = 1.0
            } else {
                xx := min(x, w - x - 1)
                yy := min(y, h - y - 1)
                if xx < innerRadius && yy < innerRadius {
                    dx := float32(xx - innerRadius)
                    dy := float32(yy - innerRadius)
                    borderDist := float32(math.Sqrt(float64(dx * dx + dy * dy)))
                    if borderDist >= innerRadiusF - float32(borderWidth) && borderDist <= innerRadiusF {
                        alpha = 1.0
                    }
                } else if xx <= borderWidth || yy <= borderWidth {
                    alpha = 1.0
                }
            }
            data[x + w * y] = (uint32(float64(color >> 24) * alpha) << 24) | (color & 0xffffff)
        }
    }
}

func setAlphaToBrightness(ptr unsafe.Pointer, w, h int32) {
    data := unsafe.Slice((*uint32)(ptr), w * h)
    for y := int32(0); y < h; y++ {
        for x := int32(0); x < w; x++ {
            color := data[x + w * y]
            lum := ((color & 0xff0000) >> 16) + ((color & 0xff00) >> 8) + (color & 0xff)
            data[x + w * y] = (((lum / 3) & 0xff) << 24) | (color & 0xffffff)
        }
    }
}

// murmur128
func generateNext128(startupTimestamp int64, frameCounter int64, prevHash64 uint64) (uint64, uint64) {
	const c1 = uint64(0x87c37b91114253d5)
	const c2 = uint64(0x4cf5ad432745937f)

	a := uint64(startupTimestamp)
	b := uint64(prevHash64)
	h1 := uint64(frameCounter)
	h2 := uint64(frameCounter)

	a *= c1
	a = (a << 31) | (a >> (64 - 31)) //ROTL64(a, 31);
	a *= c2
	h1 ^= a

	h1 = (h1 << 27) | (h1 >> (64 - 27)) //ROTL64(h1, 27);
	h1 += h2
	h1 = h1*5 + 0x52dce729

	b *= c2
	b = (b << 33) | (b >> (64 - 33)) //ROTL64(b, 33);
	b *= c1
	h2 ^= b

	h2 = (h2 << 31) | (h2 >> (64 - 31)) //ROTL64(h2, 31);
	h2 += h1
	h2 = h2*5 + 0x38495ab5

	a *= c1
	a = (a << 31) | (a >> (64 - 31)) //ROTL64(a, 31);
	a *= c2
	h1 ^= a

	const size = uint64(16)
	h1 ^= size
	h2 ^= size

	h1 += h2
	h2 += h1

	const d1 = uint64(0xff51afd7ed558ccd)
	const d2 = uint64(0xc4ceb9fe1a85ec53)

	h1 ^= h1 >> 33
	h1 *= d1
	h1 ^= h1 >> 33
	h1 *= d2
	h1 ^= h1 >> 33

	h2 ^= h2 >> 33
	h2 *= d1
	h2 ^= h2 >> 33
	h2 *= d2
	h2 ^= h2 >> 33

	h1 += h2
	h2 += h1

	return h1, h2
}
