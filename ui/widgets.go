package ui

import (
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Font is used by every widget; the GameController sets it once fonts are loaded.
var Font rl.Font

var (
	PanelColor	= rl.Color{ R: 16, G: 14, B: 30, A: 215 }
	PanelEdge	= rl.Color{ R: 90, G: 80, B: 140, A: 160 }
	Accent		= rl.Color{ R: 255, G: 196, B: 64, A: 255 }
	TextColor	= rl.Color{ R: 236, G: 232, B: 248, A: 255 }
	DimText		= rl.Color{ R: 160, G: 152, B: 190, A: 255 }
	buttonIdle	= rl.Color{ R: 48, G: 42, B: 80, A: 255 }
	buttonHover	= rl.Color{ R: 72, G: 64, B: 118, A: 255 }
	buttonOn	= rl.Color{ R: 196, G: 140, B: 40, A: 255 }
)

const TEXT_SPACING = 1

func Text(s string, x, y, size float32, col rl.Color) {
	rl.DrawTextEx(Font, s, rl.Vector2{ X: x, Y: y }, size, TEXT_SPACING, col)
}

func TextWidth(s string, size float32) float32 {
	return rl.MeasureTextEx(Font, s, size, TEXT_SPACING).X
}

func TextCentered(s string, cx, y, size float32, col rl.Color) {
	Text(s, cx-TextWidth(s, size)/2, y, size, col)
}

func Panel(r rl.Rectangle) {
	rl.DrawRectangleRounded(r, 0.04, 8, PanelColor)
	rl.DrawRectangleRoundedLines(r, 0.04, 8, 2, PanelEdge)
}

func clicked(r rl.Rectangle) bool {
	return rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(rl.GetMousePosition(), r)
}

func box(r rl.Rectangle, on bool) {
	col := buttonIdle
	if on {
		col = buttonOn
	} else if rl.CheckCollisionPointRec(rl.GetMousePosition(), r) {
		col = buttonHover
	}
	rl.DrawRectangleRounded(r, 0.3, 6, col)
}

// Button draws a button and reports whether it was clicked this frame.
func Button(r rl.Rectangle, label string, on bool) bool {
	box(r, on)
	size := min(r.Height*0.55, 22)
	for size > 10 && TextWidth(label, size) > r.Width-8 {
		size--
	}
	TextCentered(label, r.X+r.Width/2, r.Y+(r.Height-size)/2, size, TextColor)
	return clicked(r)
}

// Toggle draws a labelled switch and returns its new value.
func Toggle(r rl.Rectangle, label string, on bool) bool {
	size := min(r.Height*0.6, 20)
	Text(label, r.X, r.Y+(r.Height-size)/2, size, TextColor)
	sw := rl.Rectangle{ X: r.X + r.Width - 54, Y: r.Y + (r.Height-24)/2, Width: 54, Height: 24 }
	box(sw, on)
	knobX := sw.X + 12
	if on {
		knobX = sw.X + sw.Width - 12
	}
	rl.DrawCircleV(rl.Vector2{ X: knobX, Y: sw.Y + 12 }, 9, TextColor)
	if clicked(r) {
		return !on
	}
	return on
}

// Stepper draws "label  < value >" and returns -1, 0 or +1 depending on which arrow was clicked.
func Stepper(r rl.Rectangle, label, value string) int {
	size := min(r.Height*0.6, 20)
	Text(label, r.X, r.Y+(r.Height-size)/2, size, TextColor)
	const arrowW, valueW = 28, 112
	right := rl.Rectangle{ X: r.X + r.Width - arrowW, Y: r.Y + 2, Width: arrowW, Height: r.Height - 4 }
	left := rl.Rectangle{ X: right.X - valueW - arrowW, Y: right.Y, Width: arrowW, Height: right.Height }
	delta := 0
	if Button(left, "<", false) {
		delta = -1
	}
	if Button(right, ">", false) {
		delta = 1
	}
	TextCentered(value, left.X+arrowW+valueW/2, r.Y+(r.Height-size)/2, size, Accent)
	return delta
}

// Column hands out rectangles top to bottom, for simple panel layouts.
type Column struct {
	X, Y, W	float32
	Gap		float32
}

func (c *Column) Next(h float32) rl.Rectangle {
	r := rl.Rectangle{ X: c.X, Y: c.Y, Width: c.W, Height: h }
	c.Y += h + c.Gap
	return r
}

func (c *Column) Heading(s string) {
	c.Y += 6
	r := c.Next(22)
	Text(s, r.X, r.Y, 18, Accent)
	rl.DrawLineEx(rl.Vector2{ X: r.X, Y: r.Y + 21 }, rl.Vector2{ X: r.X + r.Width, Y: r.Y + 21 }, 1, PanelEdge)
}

func (c *Column) Line(s string, size float32, col rl.Color) {
	r := c.Next(size + 2)
	Text(s, r.X, r.Y, size, col)
}

// Paragraph word-wraps s to the column width.
func (c *Column) Paragraph(s string, size float32, col rl.Color) {
	line := ""
	for _, word := range strings.Fields(s) {
		next := word
		if line != "" {
			next = line + " " + word
		}
		if line != "" && TextWidth(next, size) > c.W {
			c.Line(line, size, col)
			next = word
		}
		line = next
	}
	if line != "" {
		c.Line(line, size, col)
	}
}

// Row splits a rectangle into n equal cells with a gap between them.
func Row(r rl.Rectangle, n int, gap float32) []rl.Rectangle {
	w := (r.Width - gap*float32(n-1)) / float32(n)
	out := make([]rl.Rectangle, n)
	for i := range out {
		out[i] = rl.Rectangle{ X: r.X + float32(i)*(w+gap), Y: r.Y, Width: w, Height: r.Height }
	}
	return out
}
