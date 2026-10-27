package core

import (
	"jewel-game/board"
	"jewel-game/ui"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type particle struct {
	pos, vel	rl.Vector2
	life, max	float32
	size		float32
	col			rl.Color
}

type popup struct {
	text		string
	pos			rl.Vector2
	life, max	float32
	size		float32
	col			rl.Color
}

// flash is the bright streak, ring, or wash left by a detonating special.
type flash struct {
	kind		board.Special
	rect		rl.Rectangle	// bolts: the swept row/column
	center		rl.Vector2		// bombs, prisms
	radius		float32
	life, max	float32
}

type Effects struct {
	particles	[]particle
	popups		[]popup
	flashes		[]flash
	shake		float32
}

func (e *Effects) Burst(pos rl.Vector2, col rl.Color, n int, speed float32) {
	for i := 0; i < n; i++ {
		a := rand.Float64() * 2 * math.Pi
		v := speed * (0.4 + rand.Float32()*0.8)
		life := 0.35 + rand.Float32()*0.4
		e.particles = append(e.particles, particle{
			pos:	pos,
			vel:	rl.Vector2{ X: float32(math.Cos(a)) * v, Y: float32(math.Sin(a)) * v },
			life:	life,
			max:	life,
			size:	2 + rand.Float32()*3,
			col:	col,
		})
	}
}

func (e *Effects) Popup(text string, pos rl.Vector2, size float32, col rl.Color) {
	e.popups = append(e.popups, popup{ text: text, pos: pos, life: 1.1, max: 1.1, size: size, col: col })
}

func (e *Effects) Flash(f flash) {
	f.max = f.life
	e.flashes = append(e.flashes, f)
	e.shake = min(0.6, e.shake+0.12)
}

func (e *Effects) Update(dt float32) {
	alive := e.particles[:0]
	for _, p := range e.particles {
		p.life -= dt
		if p.life > 0 {
			p.pos.X += p.vel.X * dt
			p.pos.Y += p.vel.Y * dt
			p.vel.Y += 600 * dt
			alive = append(alive, p)
		}
	}
	e.particles = alive

	pops := e.popups[:0]
	for _, p := range e.popups {
		p.life -= dt
		p.pos.Y -= 40 * dt
		if p.life > 0 {
			pops = append(pops, p)
		}
	}
	e.popups = pops

	flashes := e.flashes[:0]
	for _, f := range e.flashes {
		f.life -= dt
		if f.life > 0 {
			flashes = append(flashes, f)
		}
	}
	e.flashes = flashes
	e.shake = max(0, e.shake-dt)
}

// Offset is the current screen-shake displacement.
func (e *Effects) Offset() rl.Vector2 {
	if e.shake <= 0 {
		return rl.Vector2{}
	}
	m := e.shake * 14
	return rl.Vector2{ X: (rand.Float32()*2 - 1) * m, Y: (rand.Float32()*2 - 1) * m }
}

func (e *Effects) DrawFlashes() {
	for _, f := range e.flashes {
		t := f.life / f.max
		switch {
		case f.kind.IsBolt():
			grow := rl.Rectangle{ X: f.rect.X, Y: f.rect.Y, Width: f.rect.Width, Height: f.rect.Height }
			if f.kind == board.BoltH {
				grow.Y += f.rect.Height * (1 - t) / 2
				grow.Height *= t
			} else {
				grow.X += f.rect.Width * (1 - t) / 2
				grow.Width *= t
			}
			rl.DrawRectangleRec(grow, rl.Fade(rl.White, 0.85*t))
		case f.kind == board.Bomb:
			r := f.radius * (1.2 - 0.6*t)
			rl.DrawCircleV(f.center, r, rl.Fade(rl.Orange, 0.18*t))
			rl.DrawRing(f.center, r*0.9, r, 0, 360, 48, rl.Fade(rl.Yellow, 0.7*t))
		case f.kind == board.Prism:
			r := f.radius * (1.4 - t)
			rl.DrawRing(f.center, r*0.9, r, 0, 360, 64, rl.Fade(rl.ColorFromHSV(360*t, 0.6, 1), t))
		}
	}
}

func (e *Effects) Draw() {
	for _, p := range e.particles {
		rl.DrawCircleV(p.pos, p.size*(p.life/p.max+0.3), rl.Fade(p.col, p.life/p.max))
	}
	for _, p := range e.popups {
		a := min(1, p.life/p.max*2)
		w := ui.TextWidth(p.text, p.size)
		x := p.pos.X - w/2
		ui.Text(p.text, x+2, p.pos.Y+2, p.size, rl.Fade(rl.Black, 0.6*a))
		ui.Text(p.text, x, p.pos.Y, p.size, rl.Fade(p.col, a))
	}
}
