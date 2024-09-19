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