package ui

import (
	"fmt"
	"strings"

	"jewel-game/rl"
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

// Blocked makes widgets ignore the mouse, for everything underneath a modal dialog.
var Blocked bool

var clickTaken bool

// NewFrame re-arms clicks; one click only ever activates one widget, even if it changes the screen.
func NewFrame() {
	clickTaken = false
}

func Text(s string, x, y, size float32, col rl.Color) {
	rl.DrawTextEx(Font, s, rl.Vector2{ X: x, Y: y }, size, TEXT_SPACING, col)
}

func TextWidth(s string, size float32) float32 {
	return rl.MeasureTextEx(Font, s, size, TEXT_SPACING).X
}

func TextCentered(s string, cx, y, size float32, col rl.Color) {
	Text(s, cx-TextWidth(s, size)/2, y, size, col)
}

// TextOutlined draws a 1px black rim so light letters stay readable on the hex map.
func TextOutlined(s string, x, y, size float32, col rl.Color) {
	ink := rl.Color{ R: 0, G: 0, B: 0, A: 230 }
	for _, d := range [][2]float32{
		{-2, 0}, {2, 0}, {0, -2}, {0, 2},
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
	} {
		Text(s, x+d[0], y+d[1], size, ink)
	}
	Text(s, x, y, size, col)
}

func TextCenteredOutlined(s string, cx, y, size float32, col rl.Color) {
	TextOutlined(s, cx-TextWidth(s, size)/2, y, size, col)
}

func Panel(r rl.Rectangle) {
	if stone(r, true) {
		return
	}
	rl.DrawRectangleRounded(r, 0.04, 8, PanelColor)
	rl.DrawRectangleRoundedLines(r, 0.04, 8, 2, PanelEdge)
}

// Hover reports whether the mouse is over r and free to interact with it.
func Hover(r rl.Rectangle) bool {
	return !Blocked && rl.CheckCollisionPointRec(rl.GetMousePosition(), r)
}

// Clicked reports a click on r, for custom widgets.
func Clicked(r rl.Rectangle) bool { return clicked(r) }

func clicked(r rl.Rectangle) bool {
	if Blocked || clickTaken || !rl.IsMouseButtonPressed(rl.MouseLeftButton) || !rl.CheckCollisionPointRec(rl.GetMousePosition(), r) {
		return false
	}
	clickTaken = true
	return true
}

func box(r rl.Rectangle, on bool) {
	ensureTheme()
	tex := themeButton
	if on {
		tex = themeButtonOn
	}
	if nine(tex, r, 10, rl.White) {
		if !on && Hover(r) {
			nine(themeButtonOn, r, 10, rl.Fade(rl.White, 0.45))
		}
		return
	}
	col := buttonIdle
	if on {
		col = buttonOn
	} else if Hover(r) {
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
	col := TextColor
	if on {
		col = rl.Color{ R: 48, G: 28, B: 12, A: 255 }
	}
	TextCentered(label, r.X+r.Width/2, r.Y+(r.Height-size)/2, size, col)
	return clicked(r)
}

// ButtonIf is a Button that greys out and ignores clicks when disabled.
func ButtonIf(r rl.Rectangle, label string, enabled bool) bool {
	if enabled {
		return Button(r, label, false)
	}
	ensureTheme()
	if !nine(themeButton, r, 10, rl.Fade(rl.White, 0.45)) {
		rl.DrawRectangleRounded(r, 0.3, 6, rl.Fade(buttonIdle, 0.5))
	}
	size := min(r.Height*0.55, 22)
	for size > 10 && TextWidth(label, size) > r.Width-8 {
		size--
	}
	TextCentered(label, r.X+r.Width/2, r.Y+(r.Height-size)/2, size, DimText)
	return false
}

// Bar draws a meter like an HP bar, with "label  value/max" on it.
func Bar(r rl.Rectangle, label string, value, maxValue int, col rl.Color) {
	ensureTheme()
	framed := nine(themeBar, r, 5, rl.White)
	if !framed {
		rl.DrawRectangleRounded(r, 0.4, 6, rl.Color{ R: 0, G: 0, B: 0, A: 140 })
		rl.DrawRectangleRoundedLines(r, 0.4, 6, 1.5, PanelEdge)
	}
	if maxValue > 0 && value > 0 {
		fill := r
		fill.Width = r.Width * float32(min(value, maxValue)) / float32(maxValue)
		if framed && themeBarFill.ID != 0 {
			inset := min(5, min(r.Height, r.Width)*0.22)
			inner := rl.Rectangle{ X: r.X + inset, Y: r.Y + inset, Width: fill.Width - inset, Height: r.Height - inset*2 }
			if maxW := r.X + r.Width - inset - inner.X; inner.Width > maxW {
				inner.Width = maxW
			}
			if inner.Width > 1 && inner.Height > 1 {
				rl.DrawTexturePro(themeBarFill,
					rl.Rectangle{ Width: float32(themeBarFill.Width), Height: float32(themeBarFill.Height) },
					inner, rl.Vector2{}, 0, col)
			}
		} else {
			rl.DrawRectangleRounded(fill, 0.4, 6, col)
		}
	}
	size := min(r.Height*0.62, 18)
	y := r.Y + (r.Height-size)/2
	Text(label, r.X+10, y, size, TextColor)
	v := fmt.Sprintf("%d / %d", value, maxValue)
	Text(v, r.X+r.Width-10-TextWidth(v, size), y, size, TextColor)
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

// MinText is the smallest size Column text is drawn at, so long lines stay readable.
const MinText = 17

func (c *Column) Line(s string, size float32, col rl.Color) {
	size = max(size, MinText)
	r := c.Next(size + 2)
	Text(s, r.X, r.Y, size, col)
}

// Paragraph word-wraps s to the column width.
func (c *Column) Paragraph(s string, size float32, col rl.Color) {
	size = max(size, MinText)
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

// MeasureParagraph returns the height Paragraph would take at this width, assuming a gap of 4.
func MeasureParagraph(s string, size, width float32) float32 {
	size = max(size, MinText)
	lines, line := 0, ""
	for _, word := range strings.Fields(s) {
		next := word
		if line != "" {
			next = line + " " + word
		}
		if line != "" && TextWidth(next, size) > width {
			lines++
			next = word
		}
		line = next
	}
	if line != "" {
		lines++
	}
	return float32(lines) * (size + 6)
}
