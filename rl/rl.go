// Package rl is a small raylib-style drawing and input API on top of Ebitengine. The game was
// written against raylib-go, which can't target the browser; keeping the same names let the
// game code move over unchanged. Only what the game uses is implemented.
package rl

import (
	"image/color"
	"math"
)

type Vector2 struct {
	X, Y float32
}

type Rectangle struct {
	X, Y, Width, Height float32
}

// Color is straight (not premultiplied) alpha, as in raylib.
type Color = color.RGBA

type Camera2D struct {
	Offset, Target	Vector2
	Rotation		float32	// ignored
	Zoom			float32
}

var (
	White		= Color{ R: 255, G: 255, B: 255, A: 255 }
	Black		= Color{ R: 0, G: 0, B: 0, A: 255 }
	RayWhite	= Color{ R: 245, G: 245, B: 245, A: 255 }
	Yellow		= Color{ R: 253, G: 249, B: 0, A: 255 }
	Orange		= Color{ R: 255, G: 161, B: 0, A: 255 }
	Lime		= Color{ R: 0, G: 158, B: 47, A: 255 }
	SkyBlue		= Color{ R: 102, G: 191, B: 255, A: 255 }
	DarkBlue	= Color{ R: 0, G: 82, B: 172, A: 255 }
	Beige		= Color{ R: 211, G: 176, B: 131, A: 255 }
)

func Fade(c Color, alpha float32) Color {
	c.A = uint8(255 * max(0, min(1, alpha)))
	return c
}

// ColorBrightness lightens toward white for factor > 0 and darkens toward black for factor < 0.
func ColorBrightness(c Color, factor float32) Color {
	factor = max(-1, min(1, factor))
	f := func(v uint8) uint8 {
		x := float32(v)
		if factor < 0 {
			x *= 1 + factor
		} else {
			x += (255 - x) * factor
		}
		return uint8(x)
	}
	return Color{ R: f(c.R), G: f(c.G), B: f(c.B), A: c.A }
}

// ColorFromHSV takes hue in degrees and saturation and value in 0..1.
func ColorFromHSV(hue, saturation, value float32) Color {
	k := func(n float32) uint8 {
		k := float32(math.Mod(float64(n+hue/60), 6))
		t := max(0, min(k, 4-k, 1))
		return uint8(255 * (value - value*saturation*t))
	}
	return Color{ R: k(5), G: k(3), B: k(1), A: 255 }
}

func CheckCollisionPointRec(p Vector2, r Rectangle) bool {
	return p.X >= r.X && p.X < r.X+r.Width && p.Y >= r.Y && p.Y < r.Y+r.Height
}
