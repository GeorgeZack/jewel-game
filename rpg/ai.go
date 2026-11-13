package rpg

import "jewel-game/board"

// ChooseSpell picks a spell for the side to move, or nil to make a swap instead.
func (d *Duel) ChooseSpell() *Spell {
	me := d.Sides[d.Turn]
	c := Ctx{D: d, Me: me, Foe: d.Sides[1-d.Turn], Board: d.Board}
	var options []*Spell
	for _, s := range me.Spells {
		if me.CanCast(s) && (s.Worth == nil || s.Worth(c)) {
			options = append(options, s)
		}
	}
	if len(options) == 0 || d.rng.Intn(100) < 25 {
		return nil
	}
	return options[d.rng.Intn(len(options))]
}

// ChooseMove picks a swap for the side to move. It does not solve the board.
// A combo (a run of 4 or more, a wild, or a +5 skull) is only taken when the foe notices it.
// That gets more reliable with level, so an early monster often leaves one sitting there.
// Other turns prefer skulls and mana it still has room for, over gold, stars, and mana it is full of.
func (d *Duel) ChooseMove() (a, c board.Point, ok bool) {
	me := d.Sides[d.Turn]
	var combo, useful, plain [][2]board.Point
	b := d.Board
	for y := range b.H() {
		for x := range b.W() {
			p := board.Point{X: x, Y: y}
			for _, q := range []board.Point{p.Add(board.Point{X: 1}), p.Add(board.Point{Y: 1})} {
				if !b.In(q) || !b.IsValidSwap(p, q) {
					continue
				}
				tier := 0
				for _, m := range b.PreviewSwap(p, q) {
					switch {
					case m.Size >= 4 || m.Wilds > 0 || (m.Color == Skull && m.Bombs > 0):
						tier = 2
					case tier < 2 && usefulMatch(me, m):
						tier = 1
					}
				}
				move := [2]board.Point{p, q}
				switch tier {
				case 2:
					combo = append(combo, move)
				case 1:
					useful = append(useful, move)
				default:
					plain = append(plain, move)
				}
			}
		}
	}
	pick := func(pool [][2]board.Point) (board.Point, board.Point, bool) {
		if len(pool) == 0 {
			return board.Point{}, board.Point{}, false
		}
		m := pool[d.rng.Intn(len(pool))]
		return m[0], m[1], true
	}
	// Level 1 notices about a quarter of combos. Level 16 notices about two thirds.
	if len(combo) > 0 && d.rng.Intn(100) < min(85, 20+me.Level*3) {
		return pick(combo)
	}
	// Prefer a skull or mana it can still store, when it has one.
	if len(useful) > 0 && (len(plain) == 0 || d.rng.Intn(100) < min(95, 70+me.Level*2)) {
		return pick(useful)
	}
	if a, c, ok = pick(plain); ok {
		return a, c, ok
	}
	if a, c, ok = pick(useful); ok {
		return a, c, ok
	}
	return pick(combo)
}

func usefulMatch(me *Combatant, m board.Match) bool {
	if m.Color == Skull {
		return true
	}
	return IsMana(m.Color) && me.Mana[m.Color] < me.MaxMana[m.Color]
}
