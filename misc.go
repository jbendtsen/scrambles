package main

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
