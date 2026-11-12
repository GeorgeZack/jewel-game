package core

import (
	"jewel-game/board"
	"jewel-game/entities"
	"math"

	"jewel-game/rl"
)

const (
	FALL_ACCEL  = 70 // cells per second squared
	START_SPEED = 4  // cells per second
	POP_SPEED   = 5  // scale per second for gems appearing in place
	DIE_SPEED   = 4  // scale per second for gems being cleared
)

// Tile is the on-screen sprite for one gem. It's keyed by gem ID, so it glides to wherever
// the board puts that gem: swaps, falls, shuffles and undo all animate for free.
type Tile struct {
	ID     int
	Pos    rl.Vector2 // in board cells, fractional mid-move, off-board while spawning
	Target rl.Vector2
	Vel    float32
	Scale  float32
	Alpha  float32
	Flash  float32
	Gem    board.Gem // last seen state, so cleared gems can still be drawn while they vanish
	Dying  bool
}

func NewTile(g *board.Gem, pos rl.Vector2, scale float32) *Tile {
	return &Tile{ID: g.ID, Pos: pos, Target: pos, Scale: scale, Alpha: 1, Gem: *g}
}

// Update moves the tile toward its target and reports whether it has come to rest.
func (t *Tile) Update(dt float32) bool {
	if t.Dying {
		t.Flash = min(1, t.Flash+dt*8)
		t.Scale -= dt * DIE_SPEED * 0.5
		t.Alpha = max(0, t.Scale)
		return t.Scale <= 0
	}
	if t.Scale < 1 {
		t.Scale = min(1, t.Scale+dt*POP_SPEED)
	}
	dx, dy := t.Target.X-t.Pos.X, t.Target.Y-t.Pos.Y
	dist := float32(math.Hypot(float64(dx), float64(dy)))
	if dist < 0.001 {
		t.Pos, t.Vel = t.Target, 0
		return t.Scale >= 1
	}
	if t.Vel == 0 {
		t.Vel = START_SPEED
	}
	t.Vel += FALL_ACCEL * dt
	step := t.Vel * dt
	if step >= dist {
		t.Pos, t.Vel = t.Target, 0
		return t.Scale >= 1
	}
	t.Pos.X += dx / dist * step
	t.Pos.Y += dy / dist * step
	return false
}

func (t *Tile) Draw(tm *TileManager, offset rl.Vector2) {
	types := tm.gemTypes()
	var gt *entities.GemType
	if t.Gem.Color >= 0 && len(types) > 0 {
		gt = types[t.Gem.Color%len(types)]
	}
	c := tm.CellCenter(t.Pos)
	m := rl.GetMousePosition()
	look := entities.GemLook{
		Type:    gt,
		Prism:   tm.DataManager.PrismTexture(),
		Gem:     t.Gem,
		Center:  rl.Vector2{X: c.X + offset.X, Y: c.Y + offset.Y},
		Size:    tm.CellSize,
		Scale:   t.Scale,
		Alpha:   t.Alpha,
		Flash:   t.Flash,
		Time:    tm.time,
		Light:   rl.Vector2{X: m.X - offset.X, Y: m.Y - offset.Y},
		LightOn: true,
	}
	if tm.idleID == t.Gem.ID {
		look.PlayIdle = true
		look.IdleAt = tm.time - tm.idleStart
	}
	entities.DrawGem(look)
}
