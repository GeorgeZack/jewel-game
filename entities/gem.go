package entities

import (
	"image/color"
	"jewel-game/board"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type GemType struct {
	Color			color.RGBA
	Name			string
	Rotation, Scale	float32
	Texture			rl.Texture2D
	TexturePath		string
	Role			string	// battle role: attack, mana, heal, or empty
}

var lockColor = rl.Color{ R: 140, G: 210, B: 255, A: 255 }

// GemLook is everything needed to draw one gem at one moment.
type GemLook struct {
	Type	*GemType	// nil for prisms
	Prism	rl.Texture2D
	Gem		board.Gem
	Center	rl.Vector2
	Size	float32		// cell size in pixels
	Scale	float32
	Alpha	float32
	Flash	float32		// 0..1 white-out while being cleared
	Time	float64
}

func (l GemLook) tint() rl.Color {
	if l.Gem.Special == board.Prism || l.Type == nil {
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
	if l.Gem.Special == board.Prism {
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
	}
	if tex.ID != 0 {
		d := s * 0.74
		src := rl.Rectangle{ Width: float32(tex.Width), Height: float32(tex.Height) }
		dst := rl.Rectangle{ X: c.X, Y: c.Y, Width: d, Height: d }
		rl.DrawTexturePro(tex, src, dst, rl.Vector2{ X: d / 2, Y: d / 2 }, rot, rl.Fade(lerpColor(col, rl.White, l.Flash), l.Alpha))
	} else {
		rl.DrawPoly(c, 6, s*0.34, rot, rl.Fade(col, l.Alpha))
	}

	glow := rl.Fade(rl.White, (0.55+0.45*pulse)*l.Alpha)
	switch l.Gem.Special {
	case board.BoltH, board.BoltV:
		drawBolt(c, s, l.Gem.Special == board.BoltH, glow)
	case board.Bomb:
		rl.DrawRing(c, s*0.40, s*0.40+3+2*pulse, 0, 360, 36, glow)
		spark := rl.Vector2{ X: c.X + s*0.29, Y: c.Y - s*0.29 }
		rl.DrawCircleV(spark, 3+3*pulse, rl.Fade(rl.Orange, l.Alpha))
	}

	if l.Gem.Locked {
		drawLock(c, l.Size, l.Alpha)
	}
}

func drawBolt(c rl.Vector2, s float32, horizontal bool, col rl.Color) {
	along, across := rl.Vector2{ X: 1 }, rl.Vector2{ Y: 1 }
	if !horizontal {
		along, across = across, along
	}
	at := func(a, b float32) rl.Vector2 {
		return rl.Vector2{ X: c.X + along.X*a + across.X*b, Y: c.Y + along.Y*a + across.Y*b }
	}
	half, off := s*0.40, s*0.10
	for _, side := range []float32{ -off, off } {
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
	cell := rl.Rectangle{ X: c.X - size/2 + 3, Y: c.Y - size/2 + 3, Width: size - 6, Height: size - 6 }
	rl.DrawRectangleRounded(cell, 0.18, 6, rl.Fade(lockColor, 0.18*alpha))
	rl.DrawRectangleRoundedLines(cell, 0.18, 6, 3, rl.Fade(lockColor, 0.9*alpha))

	// small padlock in the bottom-right corner
	w, h := size*0.22, size*0.16
	body := rl.Rectangle{ X: cell.X + cell.Width - w - 4, Y: cell.Y + cell.Height - h - 4, Width: w, Height: h }
	shackle := rl.Vector2{ X: body.X + w/2, Y: body.Y }
	rl.DrawRing(shackle, w*0.24, w*0.36, 180, 360, 12, rl.Fade(lockColor, alpha))
	rl.DrawRectangleRounded(body, 0.25, 4, rl.Fade(lockColor, alpha))
	rl.DrawCircleV(rl.Vector2{ X: body.X + w/2, Y: body.Y + h/2 }, h*0.16, rl.Fade(rl.DarkBlue, alpha))
}

func lerpColor(a, b rl.Color, t float32) rl.Color {
	if t <= 0 {
		return a
	}
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t) }
	return rl.Color{ R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: l(a.A, b.A) }
}
