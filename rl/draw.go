package rl

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// atlas holds a white pixel for solid shapes plus every glyph drawn recently, so shapes and text
// share one source image and a whole frame of them goes to the GPU as one batch.
const atlasSize = 2048

var atlas = func() *ebiten.Image {
	img := ebiten.NewImage(atlasSize, atlasSize)
	img.SubImage(image.Rect(0, 0, 3, 3)).(*ebiten.Image).Fill(color.White)
	return img
}()

type slot struct{ x, y, w, h int }

var (
	slots				= map[*ebiten.Image]slot{}
	packX, packY, rowH	= 4, 0, 4
)

// glyphSlot copies a glyph image into the atlas the first time it's seen.
func glyphSlot(img *ebiten.Image) (slot, bool) {
	if sl, ok := slots[img]; ok {
		return sl, true
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 || w > 256 || h > 256 {
		return slot{}, false
	}
	if packX+w+1 > atlasSize {
		packX, packY, rowH = 0, packY+rowH+1, 0
	}
	if packY+h+1 > atlasSize {
		// full: start over, everything gets copied in again as it's used
		flush()
		atlas.Clear()
		atlas.SubImage(image.Rect(0, 0, 3, 3)).(*ebiten.Image).Fill(color.White)
		clear(slots)
		packX, packY, rowH = 4, 0, 4
	}
	sl := slot{ packX, packY, w, h }
	op := &ebiten.DrawImageOptions{ Blend: ebiten.BlendCopy }
	op.GeoM.Translate(float64(packX-b.Min.X), float64(packY-b.Min.Y))
	atlas.DrawImage(img, op)
	slots[img] = sl
	packX += w + 1
	rowH = max(rowH, h)
	return sl, true
}

// Shapes are collected into one batch and drawn with a single call, flushed only when something
// else (an image, text, a clip change, the end of the frame) has to be drawn in order. Edges get a
// one-pixel fringe fading to transparent instead of Ebitengine's anti-aliasing, which costs extra
// render passes per call and made the game crawl in browsers.
var (
	verts	[]ebiten.Vertex
	idx		[]uint32
)

func premul(c Color) (r, g, b, a float32) {
	a = float32(c.A) / 255
	return float32(c.R) / 255 * a, float32(c.G) / 255 * a, float32(c.B) / 255 * a, a
}

// pv is a point already in physical pixels.
type pv struct{ x, y float32 }

// Scratch buffers reused by every shape, so drawing doesn't allocate (garbage collection is slow in WebAssembly).
var (
	physA, physB				[]pv
	ptsA, ptsB					[]Vector2
	vA, vB, vC, vD				[]uint32
)

func phys(buf *[]pv, pts []Vector2) []pv {
	out := (*buf)[:0]
	for _, p := range pts {
		x, y := transform(p.X, p.Y)
		out = append(out, pv{ x, y })
	}
	*buf = out
	return out
}

func indices(buf *[]uint32, n int) []uint32 {
	if cap(*buf) < n {
		*buf = make([]uint32, n)
	}
	*buf = (*buf)[:n]
	return *buf
}

func addVert(p pv, c Color, alpha float32) uint32 {
	r, g, b, a := premul(c)
	verts = append(verts, ebiten.Vertex{ DstX: p.x, DstY: p.y, SrcX: 1.5, SrcY: 1.5, ColorR: r * alpha, ColorG: g * alpha, ColorB: b * alpha, ColorA: a * alpha })
	return uint32(len(verts) - 1)
}

func flush() {
	if len(idx) > 0 && screen != nil {
		screen.DrawTriangles32(verts, idx, atlas, nil)
	}
	verts, idx = verts[:0], idx[:0]
}

// glyphQuad adds a textured rectangle from the atlas, tinted by c.
func glyphQuad(sl slot, x, y float32, c Color) {
	r, g, b, a := premul(c)
	v := func(dx, dy float32) uint32 {
		verts = append(verts, ebiten.Vertex{ DstX: x + dx, DstY: y + dy, SrcX: float32(sl.x) + dx, SrcY: float32(sl.y) + dy, ColorR: r, ColorG: g, ColorB: b, ColorA: a })
		return uint32(len(verts) - 1)
	}
	w, h := float32(sl.w), float32(sl.h)
	quad(v(0, 0), v(w, 0), v(0, h), v(w, h))
	maybeFlush()
}

func quad(a, b, c, d uint32) {
	idx = append(idx, a, b, c, b, d, c)
}

func norm(x, y float32) (float32, float32) {
	l := float32(math.Hypot(float64(x), float64(y)))
	if l == 0 {
		return 0, 0
	}
	return x / l, y / l
}

const feather = 0.5	// half the soft edge width, in physical pixels

// fan fills a convex polygon with a soft edge.
func fan(pts []Vector2, c Color) {
	if c.A == 0 {
		return
	}
	p := phys(&physA, pts)
	n := len(p)
	if n < 3 {
		return
	}
	var area float32
	for i := range n {
		j := (i + 1) % n
		area += p[i].x*p[j].y - p[j].x*p[i].y
	}
	sign := float32(1)
	if area < 0 {
		sign = -1
	}
	// outward normal of edge i (from point i to i+1)
	edge := func(i int) (float32, float32) {
		j := (i + 1) % n
		x, y := norm(p[j].x-p[i].x, p[j].y-p[i].y)
		return y * sign, -x * sign
	}
	inner := indices(&vA, n)
	outer := indices(&vB, n)
	for i := range n {
		ax, ay := edge((i + n - 1) % n)
		bx, by := edge(i)
		mx, my := norm(ax+bx, ay+by)
		d := mx*bx + my*by
		if d < 0.3 {
			d = 0.3
		}
		ox, oy := mx/d*feather, my/d*feather
		inner[i] = addVert(pv{ p[i].x - ox, p[i].y - oy }, c, 1)
		outer[i] = addVert(pv{ p[i].x + ox, p[i].y + oy }, c, 0)
	}
	for i := 1; i+1 < n; i++ {
		idx = append(idx, inner[0], inner[i], inner[i+1])
	}
	for i := range n {
		j := (i + 1) % n
		quad(inner[i], outer[i], inner[j], outer[j])
	}
	maybeFlush()
}

// strip fills the band between two point lists of equal length, closing it if closed, with soft edges.
func strip(outerPts, innerPts []Vector2, closed bool, c Color) {
	n := len(outerPts)
	if n < 2 || len(innerPts) != n || c.A == 0 {
		return
	}
	o, in := phys(&physA, outerPts), phys(&physB, innerPts)
	oF, oS, iS, iF := indices(&vA, n), indices(&vB, n), indices(&vC, n), indices(&vD, n)
	for k := range n {
		dx, dy := o[k].x-in[k].x, o[k].y-in[k].y
		w := float32(math.Hypot(float64(dx), float64(dy)))
		ux, uy := norm(dx, dy)
		f := min(feather, w/2)
		oF[k] = addVert(pv{ o[k].x + ux*feather, o[k].y + uy*feather }, c, 0)
		oS[k] = addVert(pv{ o[k].x - ux*f, o[k].y - uy*f }, c, 1)
		iS[k] = addVert(pv{ in[k].x + ux*f, in[k].y + uy*f }, c, 1)
		iF[k] = addVert(pv{ in[k].x - ux*feather, in[k].y - uy*feather }, c, 0)
	}
	segs := n - 1
	if closed {
		segs = n
	}
	for k := range segs {
		j := (k + 1) % n
		quad(oF[k], oS[k], oF[j], oS[j])
		quad(oS[k], iS[k], oS[j], iS[j])
		quad(iS[k], iF[k], iS[j], iF[j])
	}
	maybeFlush()
}

func maybeFlush() {
	if len(verts) > 1<<15 {
		flush()
	}
}

// arc appends points along a circle to pts.
func arc(pts []Vector2, c Vector2, r, startDeg, endDeg float32, segments int) []Vector2 {
	for i := 0; i <= segments; i++ {
		a := float64(startDeg+(endDeg-startDeg)*float32(i)/float32(segments)) * math.Pi / 180
		pts = append(pts, Vector2{ X: c.X + r*float32(math.Cos(a)), Y: c.Y + r*float32(math.Sin(a)) })
	}
	return pts
}

// circleSegments picks enough segments for a smooth edge at the current zoom.
func circleSegments(r float32) int {
	return max(12, min(96, int(r*scale()*0.8)))
}

func DrawCircleV(c Vector2, radius float32, col Color) {
	ptsA = arc(ptsA[:0], c, radius, 0, 360, circleSegments(radius))
	fan(ptsA[:len(ptsA)-1], col)
}

func DrawRing(c Vector2, inner, outer, startAngle, endAngle float32, segments int32, col Color) {
	n := max(int(segments), circleSegments(outer)*int(math.Abs(float64(endAngle-startAngle)))/360, 2)
	ptsA = arc(ptsA[:0], c, outer, startAngle, endAngle, n)
	ptsB = arc(ptsB[:0], c, inner, startAngle, endAngle, n)
	strip(ptsA, ptsB, false, col)
}

// DrawPoly draws a regular polygon with its first corner at rotation degrees.
func DrawPoly(c Vector2, sides int32, radius, rotation float32, col Color) {
	ptsA = arc(ptsA[:0], c, radius, rotation, rotation+360, int(sides))
	fan(ptsA[:len(ptsA)-1], col)
}

func DrawTriangle(a, b, c Vector2, col Color) {
	ptsA = append(ptsA[:0], a, b, c)
	fan(ptsA, col)
}

func DrawLineEx(a, b Vector2, thick float32, col Color) {
	dx, dy := b.X-a.X, b.Y-a.Y
	l := float32(math.Hypot(float64(dx), float64(dy)))
	if l == 0 {
		return
	}
	nx, ny := -dy/l*thick/2, dx/l*thick/2
	ptsA = append(ptsA[:0], Vector2{ a.X + nx, a.Y + ny }, Vector2{ b.X + nx, b.Y + ny }, Vector2{ b.X - nx, b.Y - ny }, Vector2{ a.X - nx, a.Y - ny })
	fan(ptsA, col)
}

func rectPoints(pts []Vector2, r Rectangle) []Vector2 {
	return append(pts[:0], Vector2{ r.X, r.Y }, Vector2{ r.X + r.Width, r.Y }, Vector2{ r.X + r.Width, r.Y + r.Height }, Vector2{ r.X, r.Y + r.Height })
}

func DrawRectangleRec(r Rectangle, col Color) {
	ptsA = rectPoints(ptsA, r)
	fan(ptsA, col)
}

func DrawRectangle(x, y, w, h int32, col Color) {
	DrawRectangleRec(Rectangle{ X: float32(x), Y: float32(y), Width: float32(w), Height: float32(h) }, col)
}

func DrawRectangleGradientV(x, y, w, h int32, top, bottom Color) {
	p := phys(&physA, append(ptsA[:0], Vector2{ float32(x), float32(y) }, Vector2{ float32(x + w), float32(y) }, Vector2{ float32(x), float32(y + h) }, Vector2{ float32(x + w), float32(y + h) }))
	quad(addVert(p[0], top, 1), addVert(p[1], top, 1), addVert(p[2], bottom, 1), addVert(p[3], bottom, 1))
}

// DrawRectangleLinesEx draws an outline inside the rectangle.
func DrawRectangleLinesEx(r Rectangle, thick float32, col Color) {
	in := Rectangle{ X: r.X + thick, Y: r.Y + thick, Width: r.Width - 2*thick, Height: r.Height - 2*thick }
	ptsA, ptsB = rectPoints(ptsA, r), rectPoints(ptsB, in)
	strip(ptsA, ptsB, true, col)
}

// roundedPoints outlines a rounded rectangle clockwise from the top-left corner, with corner
// radius radius and segments steps per corner.
func roundedPoints(pts []Vector2, r Rectangle, radius float32, segments int) []Vector2 {
	radius = max(0, min(radius, r.Width/2, r.Height/2))
	corners := []struct{ c Vector2; start float32 }{
		{ Vector2{ r.X + radius, r.Y + radius }, 180 },
		{ Vector2{ r.X + r.Width - radius, r.Y + radius }, 270 },
		{ Vector2{ r.X + r.Width - radius, r.Y + r.Height - radius }, 0 },
		{ Vector2{ r.X + radius, r.Y + r.Height - radius }, 90 },
	}
	pts = pts[:0]
	for _, k := range corners {
		pts = arc(pts, k.c, radius, k.start, k.start+90, segments)
	}
	return pts
}

func cornerSegments(segments int32, radius float32) int {
	return max(int(segments), min(16, int(radius*scale()/3)), 1)
}

// DrawRectangleRounded uses raylib's roundness: the corner radius is roundness times half the shorter side.
func DrawRectangleRounded(r Rectangle, roundness float32, segments int32, col Color) {
	radius := roundness * min(r.Width, r.Height) / 2
	ptsA = roundedPoints(ptsA, r, radius, cornerSegments(segments, radius))
	fan(ptsA, col)
}

// DrawRectangleRoundedLines draws the outline just outside the rectangle, as raylib does.
func DrawRectangleRoundedLines(r Rectangle, roundness float32, segments int32, thick float32, col Color) {
	radius := roundness * min(r.Width, r.Height) / 2
	n := cornerSegments(segments, radius+thick)
	out := Rectangle{ X: r.X - thick, Y: r.Y - thick, Width: r.Width + 2*thick, Height: r.Height + 2*thick }
	ptsA, ptsB = roundedPoints(ptsA, out, radius+thick, n), roundedPoints(ptsB, r, radius, n)
	strip(ptsA, ptsB, true, col)
}
