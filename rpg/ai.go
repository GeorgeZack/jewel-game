package rpg

import "jewel-game/board"

// ChooseSpell picks a spell for the side to move, or nil to make a swap instead.
func (d *Duel) ChooseSpell() *Spell {
	me := d.Sides[d.Turn]
	c := Ctx{ D: d, Me: me, Foe: d.Sides[1-d.Turn], Board: d.Board }
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

// ChooseMove picks the best looking swap for the side to move. Higher level foes make fewer
// mistakes: noise shrinks as they level, so early monsters miss good moves.
func (d *Duel) ChooseMove() (a, c board.Point, ok bool) {
	me := d.Sides[d.Turn]
	noise := max(2, 14-me.Level)
	best := -1
	b := d.Board
	consider := func(p, q board.Point) {
		if !b.In(q) || !b.IsValidSwap(p, q) {
			return
		}
		score := d.rng.Intn(noise)
		for _, m := range b.PreviewSwap(p, q) {
			score += d.scoreMatch(me, m)
		}
		if score > best {
			best, a, c, ok = score, p, q, true
		}
	}
	for y := range b.H() {
		for x := range b.W() {
			p := board.Point{ X: x, Y: y }
			consider(p, p.Add(board.Point{ X: 1 }))
			consider(p, p.Add(board.Point{ Y: 1 }))
		}
	}
	return a, c, ok
}

func (d *Duel) scoreMatch(me *Combatant, m board.Match) int {
	score := m.Size
	if m.Size >= 4 {
		score += 20	// another turn beats nearly everything
	}
	switch {
	case m.Color == Skull:
		score += 6 + 8*m.Bombs
	case m.Color < NUM_MANA:
		// colors its spells want, and not those it is already full of
		if me.Mana[m.Color] < me.MaxMana[m.Color] {
			score += 2 * (m.Mult - 1)
			for _, s := range me.Spells {
				if me.Mana[m.Color] < s.Cost[m.Color] {
					score += 3
				}
			}
		} else {
			score -= 2
		}
	}
	return score
}
