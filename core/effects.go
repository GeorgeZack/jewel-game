package core

import (
	"jewel-game/board"
	"jewel-game/ui"
	"math"
	"math/rand"

	"jewel-game/rl"
)

type particle struct {
	pos, vel  rl.Vector2
	life, max float32
	size      float32
	col       rl.Color
}

type popup struct {
	text      string
	pos       rl.Vector2
	life, max float32
	size      float32
	col       rl.Color
}

// flash is the bright streak, ring, or wash left by a detonating special.
// A flash with no special and a color set is a plain colored ring (a heal pulse).
type flash struct {
	kind      board.Special
	rect      rl.Rectangle // bolts: the swept row/column
	center    rl.Vector2   // bombs, prisms, rings
	radius    float32
	life, max float32
	col       rl.Color
}

// slash is a cut across a rectangle, used when a duel hit lands.
type slash struct {
	rect      rl.Rectangle
	life, max float32
	power     float32
	flip      bool
}

type Effects struct {
	particles []particle
	popups    []popup
	flashes   []flash
	slashes   []slash
	shake     float32
}

func (e *Effects) Burst(pos rl.Vector2, col rl.Color, n int, speed float32) {
	for i := 0; i < n; i++ {
		a := rand.Float64() * 2 * math.Pi
		v := speed * (0.4 + rand.Float32()*0.8)
		life := 0.35 + rand.Float32()*0.4
		e.particles = append(e.particles, particle{
			pos:  pos,
			vel:  rl.Vector2{X: float32(math.Cos(a)) * v, Y: float32(math.Sin(a)) * v},
			life: life,
			max:  life,
			size: 2 + rand.Float32()*3,
			col:  col,
		})
	}
}

func (e *Effects) Popup(text string, pos rl.Vector2, size float32, col rl.Color) {
	e.popups = append(e.popups, popup{text: text, pos: pos, life: 1.1, max: 1.1, size: size, col: col})
}

func (e *Effects) Flash(f flash) {
	f.max = f.life
	e.flashes = append(e.flashes, f)
	e.shake = min(0.6, e.shake+0.12)
}

// Slash cuts across r. power is 0..1 and sets how thick and violent the cut is.
func (e *Effects) Slash(r rl.Rectangle, power float32) {
	power = min(1, max(0.08, power))
	life := 0.32 + 0.14*power
	e.slashes = append(e.slashes, slash{
		rect:  r,
		life:  life,
		max:   life,
		power: power,
		flip:  rand.Float32() < 0.5,
	})
	e.shake = min(0.75, e.shake+0.18+0.4*power)
	a, b := slashEnds(r, e.slashes[len(e.slashes)-1].flip)
	red := rl.Color{R: 220, G: 36, B: 42, A: 255}
	n := 8 + int(power*12)
	for i := 0; i < n; i++ {
		t := rand.Float32()
		at := rl.Vector2{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t}
		col := red
		if i%3 == 0 {
			col = rl.White
		}
		e.Burst(at, col, 1, 90+power*160)
	}
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
		p.pos.Y -= 90 * dt
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

	cuts := e.slashes[:0]
	for _, s := range e.slashes {
		s.life -= dt
		if s.life > 0 {
			cuts = append(cuts, s)
		}
	}
	e.slashes = cuts
	e.shake = max(0, e.shake-dt)
}

// Offset is the current screen-shake displacement.
func (e *Effects) Offset() rl.Vector2 {
	if e.shake <= 0 {
		return rl.Vector2{}
	}
	m := e.shake * 14
	return rl.Vector2{X: (rand.Float32()*2 - 1) * m, Y: (rand.Float32()*2 - 1) * m}
}

func (e *Effects) DrawFlashes(off rl.Vector2) {
	for _, f := range e.flashes {
		t := f.life / f.max
		center := rl.Vector2{X: f.center.X + off.X, Y: f.center.Y + off.Y}
		rect := f.rect
		rect.X += off.X
		rect.Y += off.Y
		switch {
		case f.kind.IsBolt():
			grow := rect
			if f.kind == board.BoltH {
				grow.Y += rect.Height * (1 - t) / 2
				grow.Height *= t
			} else {
				grow.X += rect.Width * (1 - t) / 2
				grow.Width *= t
			}
			rl.DrawRectangleRec(grow, rl.Fade(rl.White, 0.85*t))
		case f.kind == board.Bomb:
			r := f.radius * (1.2 - 0.6*t)
			rl.DrawCircleV(center, r, rl.Fade(rl.Orange, 0.18*t))
			rl.DrawRing(center, r*0.9, r, 0, 360, 48, rl.Fade(rl.Yellow, 0.7*t))
		case f.kind == board.Prism:
			r := f.radius * (1.4 - t)
			rl.DrawRing(center, r*0.9, r, 0, 360, 64, rl.Fade(rl.ColorFromHSV(360*t, 0.6, 1), t))
		default:
			if f.col.A == 0 {
				continue
			}
			r := f.radius * (1.2 - 0.45*t)
			rl.DrawCircleV(center, r*0.75, rl.Fade(f.col, 0.16*t))
			rl.DrawRing(center, r*0.78, r, 0, 360, 48, rl.Fade(f.col, 0.9*t))
		}
	}
	for _, s := range e.slashes {
		drawSlash(s, off)
	}
}

func (e *Effects) Draw() {
	for _, p := range e.particles {
		rl.DrawCircleV(p.pos, p.size*(p.life/p.max+0.3), rl.Fade(p.col, p.life/p.max))
	}
	for _, p := range e.popups {
		a := min(1, p.life/p.max*2)
		born := 1 - p.life/p.max
		punch := float32(1)
		if born < 0.12 {
			punch = 1.4 - born/0.12*0.4
		}
		size := p.size * punch
		w := ui.TextWidth(p.text, size)
		x := p.pos.X - w/2
		y := p.pos.Y - (size-p.size)*0.35
		ui.Text(p.text, x+2, y+2, size, rl.Fade(rl.Black, 0.6*a))
		ui.Text(p.text, x, y, size, rl.Fade(p.col, a))
	}
}

func slashEnds(r rl.Rectangle, flip bool) (rl.Vector2, rl.Vector2) {
	if flip {
		return rl.Vector2{X: r.X, Y: r.Y + r.Height}, rl.Vector2{X: r.X + r.Width, Y: r.Y}
	}
	return rl.Vector2{X: r.X, Y: r.Y}, rl.Vector2{X: r.X + r.Width, Y: r.Y + r.Height}
}

func drawSlash(s slash, off rl.Vector2) {
	t := s.life / s.max
	r := s.rect
	r.X += off.X
	r.Y += off.Y
	// The cut travels in the first slice of its life, then the mark fades.
	u := min(1, (1-t)/0.22)
	fade := float32(1)
	if t < 0.78 {
		fade = t / 0.78
	}
	rl.DrawRectangleRec(r, rl.Fade(rl.Color{R: 120, G: 8, B: 16, A: 255}, 0.22*s.power*fade))
	a, b := slashEnds(r, s.flip)
	tip := rl.Vector2{X: a.X + (b.X-a.X)*u, Y: a.Y + (b.Y-a.Y)*u}
	dx, dy := b.X-a.X, b.Y-a.Y
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length == 0 {
		return
	}
	nx, ny := -dy/length, dx/length
	gap := 10 + 14*s.power
	shift := func(p rl.Vector2, k float32) rl.Vector2 {
		return rl.Vector2{X: p.X + nx*gap*k, Y: p.Y + ny*gap*k}
	}
	red := rl.Color{R: 210, G: 24, B: 32, A: 255}
	thick := (5 + 16*s.power) * (0.55 + 0.45*fade)
	rl.DrawLineEx(shift(a, -0.35), shift(tip, -0.35), thick*0.45, rl.Fade(red, 0.55*fade))
	rl.DrawLineEx(a, tip, thick, rl.Fade(red, 0.9*fade))
	rl.DrawLineEx(a, tip, max(2, thick*0.28), rl.Fade(rl.White, fade))
	rl.DrawCircleV(tip, 4+7*s.power, rl.Fade(rl.White, fade*(1-u*0.35)))
}

// cellCluster is one orthogonally connected group of cleared gems of a color.
// Wilds (color < 0) that touch the group are included in cells, not in weight.
type cellCluster struct {
	cells  []board.Point
	weight int
}

// colorClusters groups cleared cells of color. Wilds bridge cells of that color
// when they sit between them, so one match stays one popup.
func colorClusters(cleared []board.Cleared, color int) []cellCluster {
	if color < 0 {
		return nil
	}
	colored := map[board.Point]bool{}
	open := map[board.Point]bool{}
	for _, c := range cleared {
		if c.Gem.Color == color {
			colored[c.Pos] = true
			open[c.Pos] = true
		} else if c.Gem.Color < 0 {
			open[c.Pos] = true
		}
	}
	seen := map[board.Point]bool{}
	dirs := [...]board.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	var out []cellCluster
	for _, c := range cleared {
		p := c.Pos
		if !colored[p] || seen[p] {
			continue
		}
		var cells []board.Point
		weight := 0
		q := []board.Point{p}
		seen[p] = true
		for len(q) > 0 {
			cur := q[0]
			q = q[1:]
			cells = append(cells, cur)
			if colored[cur] {
				weight++
			}
			for _, d := range dirs {
				n := cur.Add(d)
				if open[n] && !seen[n] {
					seen[n] = true
					q = append(q, n)
				}
			}
		}
		out = append(out, cellCluster{cells: cells, weight: weight})
	}
	return out
}

// splitAmounts shares total across weights, largest remainder first, so the parts add up.
func splitAmounts(total int, weights []int) []int {
	out := make([]int, len(weights))
	if total <= 0 || len(weights) == 0 {
		return out
	}
	sum := 0
	for _, w := range weights {
		if w > 0 {
			sum += w
		}
	}
	if sum == 0 {
		out[0] = total
		return out
	}
	left := total
	frac := make([]float64, len(weights))
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		share := float64(total) * float64(w) / float64(sum)
		out[i] = int(math.Floor(share))
		left -= out[i]
		frac[i] = share - float64(out[i])
	}
	for left > 0 {
		pick := 0
		for i := 1; i < len(weights); i++ {
			if frac[i] > frac[pick] {
				pick = i
			}
		}
		out[pick]++
		frac[pick] -= 1
		left--
	}
	return out
}
