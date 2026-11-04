package entities

import (
	"fmt"
	"image/color"
	"jewel-game/board"
	"math"

	"jewel-game/rl"
)

const (
	ShapeTexture = iota
	ShapeSkull
	ShapeCoin
	ShapeStar
)

type GemType struct {
	Color           color.RGBA
	Name            string
	Rotation, Scale float32
	Texture         rl.Texture2D
	TexturePath     string
	Bomb            rl.Texture2D // drawn instead of Texture for a bomb, the +5 skull
	Shape           int          // drawn procedurally instead of from Texture
	FullColor       bool         // Texture is already colored, so it is not multiplied by Color
}

var lockColor = rl.Color{R: 140, G: 210, B: 255, A: 255}

// GemLook is everything needed to draw one gem at one moment.
type GemLook struct {
	Type   *GemType // nil for prisms
	Prism  rl.Texture2D
	Gem    board.Gem
	Center rl.Vector2
	Size   float32 // cell size in pixels
	Scale  float32
	Alpha  float32
	Flash  float32 // 0..1 white-out while being cleared
	Time   float64
}

func (l GemLook) tint() rl.Color {
	if l.Gem.Special == board.Prism || l.Gem.Special == board.Wild || l.Type == nil {
		hue := float32(math.Mod(l.Time*120+float64(l.Center.X+l.Center.Y)*0.3, 360))
		return rl.ColorFromHSV(hue, 0.45, 1)
	}
	return l.Type.Color
}

func DrawGem(l GemLook) {
	col := l.tint()
	pulse := float32(0.5 + 0.5*math.Sin(l.Time*5))
	s := l.Size * l.Scale
	c := l.Center

	rl.DrawCircleV(c, s*0.36, rl.Fade(col, 0.16*l.Alpha))

	tex, rot := l.Prism, float32(0)
	if l.Gem.Special == board.Prism || l.Gem.Special == board.Wild {
		rot = float32(math.Mod(l.Time*40, 360))
		rl.DrawCircleV(c, s*0.42, rl.Fade(rl.White, (0.15+0.15*pulse)*l.Alpha))
		// spinning rainbow ring so prisms never read as a plain gem
		for i := 0; i < 6; i++ {
			a := float32(i)*60 + rot*2
			hue := float32(math.Mod(float64(a), 360))
			rl.DrawRing(c, s*0.40, s*0.45, a, a+50, 8, rl.Fade(rl.ColorFromHSV(hue, 0.7, 1), l.Alpha))
		}
	} else if l.Type != nil {
		tex = l.Type.Texture
		if l.Gem.Special == board.Bomb && l.Type.Bomb.ID != 0 {
			tex = l.Type.Bomb
		}
	}
	sprite := col
	if l.Type != nil && l.Type.FullColor && l.Gem.Special != board.Prism && l.Gem.Special != board.Wild {
		sprite = rl.White
	}
	if l.Type != nil && l.Type.Shape != ShapeTexture && l.Gem.Special != board.Wild {
		drawShape(l.Type.Shape, c, s, rl.Fade(lerpColor(col, rl.White, l.Flash), l.Alpha), l.Alpha)
	} else if tex.ID != 0 {
		d := s * 0.74
		if l.Type != nil && l.Type.FullColor && l.Gem.Special != board.Prism && l.Gem.Special != board.Wild {
			d = s * 0.90 // the opaque gem fills most of a trimmed full-color texture
		}
		src := rl.Rectangle{Width: float32(tex.Width), Height: float32(tex.Height)}
		dst := rl.Rectangle{X: c.X, Y: c.Y, Width: d, Height: d}
		rl.DrawTexturePro(tex, src, dst, rl.Vector2{X: d / 2, Y: d / 2}, rot, rl.Fade(lerpColor(sprite, rl.White, l.Flash), l.Alpha))
	} else {
		rl.DrawPoly(c, 6, s*0.34, rot, rl.Fade(col, l.Alpha))
	}

	glow := rl.Fade(rl.White, (0.55+0.45*pulse)*l.Alpha)
	switch l.Gem.Special {
	case board.BoltH, board.BoltV:
		drawBolt(c, s, l.Gem.Special == board.BoltH, glow)
	case board.Bomb:
		rl.DrawRing(c, s*0.40, s*0.40+3+2*pulse, 0, 360, 36, glow)
		spark := rl.Vector2{X: c.X + s*0.29, Y: c.Y - s*0.29}
		rl.DrawCircleV(spark, 3+3*pulse, rl.Fade(rl.Orange, l.Alpha))
		if l.Type != nil && (l.Type.Shape == ShapeSkull || l.Type.Bomb.ID != 0) {
			drawBadge("+5", c, s, rl.Fade(rl.Orange, l.Alpha))
		}
	case board.Wild:
		drawBadge(fmt.Sprintf("x%d", max(l.Gem.Mult, 1)), c, s, rl.Fade(rl.White, l.Alpha))
	}

	if l.Gem.Locked {
		drawLock(c, l.Size, l.Alpha)
	}
}

// drawBadge prints a small label over a gem's lower right, with a dark backing to stay readable.
func drawBadge(text string, c rl.Vector2, s float32, col rl.Color) {
	size := int32(s * 0.3)
	x, y := int32(c.X+s*0.02), int32(c.Y+s*0.1)
	w := rl.MeasureText(text, size)
	rl.DrawRectangle(x-3, y-1, w+6, size+2, rl.Fade(rl.Black, 0.65*float32(col.A)/255))
	rl.DrawText(text, x, y, size, col)
}

func drawShape(shape int, c rl.Vector2, s float32, col rl.Color, alpha float32) {
	dark := rl.Fade(rl.Color{R: 25, G: 20, B: 30, A: 255}, alpha)
	switch shape {
	case ShapeSkull:
		rl.DrawCircleV(rl.Vector2{X: c.X, Y: c.Y - s*0.07}, s*0.27, col)
		jaw := rl.Rectangle{X: c.X - s*0.15, Y: c.Y + s*0.1, Width: s * 0.3, Height: s * 0.19}
		rl.DrawRectangleRounded(jaw, 0.3, 4, col)
		for _, side := range []float32{-1, 1} {
			rl.DrawCircleV(rl.Vector2{X: c.X + side*s*0.11, Y: c.Y - s*0.07}, s*0.075, dark)
			rl.DrawLineEx(rl.Vector2{X: c.X + side*s*0.05, Y: c.Y + s*0.17}, rl.Vector2{X: c.X + side*s*0.05, Y: c.Y + s*0.29}, 2, dark)
		}
		rl.DrawLineEx(rl.Vector2{X: c.X, Y: c.Y + s*0.17}, rl.Vector2{X: c.X, Y: c.Y + s*0.29}, 2, dark)
		rl.DrawPoly(rl.Vector2{X: c.X, Y: c.Y + s*0.05}, 3, s*0.045, 30, dark)
	case ShapeCoin:
		rl.DrawCircleV(c, s*0.3, col)
		rl.DrawRing(c, s*0.22, s*0.25, 0, 360, 28, rl.Fade(rl.Color{R: 170, G: 110, B: 10, A: 255}, alpha))
		rl.DrawPoly(c, 4, s*0.12, 0, rl.Fade(rl.Color{R: 170, G: 110, B: 10, A: 255}, alpha))
	case ShapeStar:
		rl.DrawPoly(c, 3, s*0.34, 30, col)
		rl.DrawPoly(c, 3, s*0.34, 90, col)
		rl.DrawCircleV(c, s*0.1, rl.Fade(rl.White, 0.6*alpha))
	}
}

func drawBolt(c rl.Vector2, s float32, horizontal bool, col rl.Color) {
	along, across := rl.Vector2{X: 1}, rl.Vector2{Y: 1}
	if !horizontal {
		along, across = across, along
	}
	at := func(a, b float32) rl.Vector2 {
		return rl.Vector2{X: c.X + along.X*a + across.X*b, Y: c.Y + along.Y*a + across.Y*b}
	}
	half, off := s*0.40, s*0.10
	for _, side := range []float32{-off, off} {
		rl.DrawLineEx(at(-half, side), at(half, side), 3, col)
	}
	// arrowheads at both ends
	tip, back, wide := s*0.48, s*0.36, s*0.17
	rl.DrawTriangle(at(tip, 0), at(back, -wide), at(back, wide), col)
	rl.DrawTriangle(at(-tip, 0), at(-back, wide), at(-back, -wide), col)
	rl.DrawTriangle(at(tip, 0), at(back, wide), at(back, -wide), col)
	rl.DrawTriangle(at(-tip, 0), at(-back, -wide), at(-back, wide), col)
}

func drawLock(c rl.Vector2, size, alpha float32) {
	cell := rl.Rectangle{X: c.X - size/2 + 3, Y: c.Y - size/2 + 3, Width: size - 6, Height: size - 6}
	rl.DrawRectangleRounded(cell, 0.18, 6, rl.Fade(lockColor, 0.18*alpha))
	rl.DrawRectangleRoundedLines(cell, 0.18, 6, 3, rl.Fade(lockColor, 0.9*alpha))

	// small padlock in the bottom-right corner
	w, h := size*0.22, size*0.16
	body := rl.Rectangle{X: cell.X + cell.Width - w - 4, Y: cell.Y + cell.Height - h - 4, Width: w, Height: h}
	shackle := rl.Vector2{X: body.X + w/2, Y: body.Y}
	rl.DrawRing(shackle, w*0.24, w*0.36, 180, 360, 12, rl.Fade(lockColor, alpha))
	rl.DrawRectangleRounded(body, 0.25, 4, rl.Fade(lockColor, alpha))
	rl.DrawCircleV(rl.Vector2{X: body.X + w/2, Y: body.Y + h/2}, h*0.16, rl.Fade(rl.DarkBlue, alpha))
}

func lerpColor(a, b rl.Color, t float32) rl.Color {
	if t <= 0 {
		return a
	}
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t) }
	return rl.Color{R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: l(a.A, b.A)}
}
