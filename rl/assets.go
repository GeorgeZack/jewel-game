package rl

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var assets fs.FS

// SetAssets gives the files bundled into the binary, used when a file isn't on disk (always, in a browser).
func SetAssets(f fs.FS) {
	assets = f
}

// ReadFile prefers the file on disk, so edits can be hot reloaded, and falls back to the bundled copy.
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil || assets == nil {
		return data, err
	}
	if b, err2 := fs.ReadFile(assets, path); err2 == nil {
		return b, nil
	}
	return nil, err
}

// ReadEmbedded reads only the bundled copy, for when the file on disk is broken.
func ReadEmbedded(path string) ([]byte, error) {
	if assets == nil {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(assets, path)
}

// ReadDir lists the file names in a directory, on disk if it's there, else in the bundled copy.
func ReadDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil && assets != nil {
		entries, err = fs.ReadDir(assets, dir)
	}
	return names(entries), err
}

// ReadDirEmbedded lists a directory in the bundled copy only.
func ReadDirEmbedded(dir string) ([]string, error) {
	if assets == nil {
		return nil, fs.ErrNotExist
	}
	entries, err := fs.ReadDir(assets, dir)
	return names(entries), err
}

func names(entries []fs.DirEntry) []string {
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

var nextID uint32

type TextureFilter int

const (
	FilterPoint TextureFilter = iota
	FilterBilinear
)

type texData struct {
	img    *ebiten.Image
	filter ebiten.Filter
}

// Texture2D has ID 0 when it failed to load.
type Texture2D struct {
	ID            uint32
	Width, Height int32
	data          *texData
}

func LoadTexture(path string) Texture2D {
	raw, err := ReadFile(path)
	if err != nil {
		return Texture2D{}
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Texture2D{}
	}
	return NewTexture(img)
}

// NewTexture wraps an already-decoded image. The filter starts linear; pixel sprites set it to point.
func NewTexture(img image.Image) Texture2D {
	nextID++
	b := img.Bounds()
	return Texture2D{ID: nextID, Width: int32(b.Dx()), Height: int32(b.Dy()), data: &texData{img: ebiten.NewImageFromImage(img), filter: ebiten.FilterLinear}}
}

func UnloadTexture(t Texture2D) {
	if t.data != nil && t.data.img != nil {
		t.data.img.Deallocate()
	}
}

func SetTextureFilter(t Texture2D, f TextureFilter) {
	if t.data == nil {
		return
	}
	t.data.filter = ebiten.FilterLinear
	if f == FilterPoint {
		t.data.filter = ebiten.FilterNearest
	}
}

// DrawTexturePro draws the src part of t into dst, rotated by rotation degrees around origin
// (relative to dst's corner).
func DrawTexturePro(t Texture2D, src, dst Rectangle, origin Vector2, rotation float32, tint Color) {
	if t.data == nil || t.data.img == nil || screen == nil || src.Width == 0 || src.Height == 0 {
		return
	}
	flush()
	sub := t.data.img
	if src.X != 0 || src.Y != 0 || int32(src.Width) != t.Width || int32(src.Height) != t.Height {
		sub = t.data.img.SubImage(image.Rect(int(src.X), int(src.Y), int(src.X+src.Width), int(src.Y+src.Height))).(*ebiten.Image)
	}
	op := &ebiten.DrawImageOptions{Filter: t.data.filter}
	op.GeoM.Scale(float64(dst.Width/src.Width), float64(dst.Height/src.Height))
	op.GeoM.Translate(-float64(origin.X), -float64(origin.Y))
	op.GeoM.Rotate(float64(rotation) * math.Pi / 180)
	op.GeoM.Translate(float64(dst.X), float64(dst.Y))
	s := float64(scale())
	op.GeoM.Scale(s, s)
	ox, oy := transform(0, 0)
	op.GeoM.Translate(float64(ox), float64(oy))
	r, g, b, a := premul(tint)
	op.ColorScale.Scale(r, g, b, a)
	screen.DrawImage(sub, op)
}

// Font is a TTF face; sizes are raylib-style, the full line height in pixels.
type Font struct {
	ID        uint32
	Texture   Texture2D // only so raylib-style filter calls have something to take
	src       *text.GoTextFaceSource
	lineRatio float64 // line height per em
}

var defaultFont Font

func LoadFontEx(path string, size int32, codepoints []rune) Font {
	raw, err := ReadFile(path)
	if err != nil {
		return Font{}
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(raw))
	if err != nil {
		return Font{}
	}
	m := (&text.GoTextFace{Source: src, Size: 100}).Metrics()
	nextID++
	f := Font{ID: nextID, src: src, lineRatio: (m.HAscent + m.HDescent) / 100}
	if defaultFont.src == nil {
		defaultFont = f
	}
	return f
}

func UnloadFont(Font) {}

func (f Font) face(size float32) *text.GoTextFace {
	return &text.GoTextFace{Source: f.src, Size: float64(size) / f.lineRatio}
}

// Shaping text is expensive, especially in WebAssembly, and the UI redraws the same strings every
// frame, so laid-out glyphs and measured widths are cached and dropped after going unused a while.
type textKey struct {
	src  *text.GoTextFaceSource
	size float32
	s    string
}

type layout struct {
	glyphs []text.Glyph
	used   int
}

var (
	frameNo int
	layouts = map[textKey]*layout{}
	widths  = map[textKey]float32{}
)

// trimTextCaches runs once per frame.
func trimTextCaches() {
	frameNo++
	if frameNo%120 != 0 {
		return
	}
	for k, l := range layouts {
		if frameNo-l.used > 120 {
			delete(layouts, k)
		}
	}
	if len(widths) > 20000 {
		clear(widths)
	}
}

// DrawTextEx ignores spacing; MeasureTextEx does too, so layout stays consistent.
func DrawTextEx(f Font, s string, pos Vector2, size, spacing float32, col Color) {
	if f.src == nil || screen == nil || s == "" || col.A == 0 {
		return
	}
	// rendered at the physical size so text stays sharp when zoomed
	px := float32(math.Round(float64(size*scale())*2) / 2)
	if px < 1 {
		return
	}
	k := textKey{f.src, px, s}
	l := layouts[k]
	if l == nil {
		l = &layout{glyphs: text.AppendGlyphs(nil, s, f.face(px), nil)}
		layouts[k] = l
	}
	l.used = frameNo
	x, y := transform(pos.X, pos.Y)
	x, y = float32(math.Round(float64(x))), float32(math.Round(float64(y)))
	for _, gl := range l.glyphs {
		if gl.Image == nil {
			continue
		}
		if sl, ok := glyphSlot(gl.Image); ok {
			glyphQuad(sl, x+float32(math.Round(gl.X)), y+float32(math.Round(gl.Y)), col)
		}
	}
}

func MeasureTextEx(f Font, s string, size, spacing float32) Vector2 {
	if f.src == nil {
		return Vector2{}
	}
	k := textKey{f.src, size, s}
	w, ok := widths[k]
	if !ok {
		fw, _ := text.Measure(s, f.face(size), 0)
		w = float32(fw)
		widths[k] = w
	}
	return Vector2{X: w, Y: size}
}

func DrawText(s string, x, y, size int32, col Color) {
	DrawTextEx(defaultFont, s, Vector2{X: float32(x), Y: float32(y)}, float32(size), 1, col)
}

func MeasureText(s string, size int32) int32 {
	return int32(MeasureTextEx(defaultFont, s, float32(size), 1).X)
}
