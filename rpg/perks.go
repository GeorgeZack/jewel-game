package rpg

import (
	"fmt"
	"slices"

	"jewel-game/board"
)

// Perks are the rules large Orrery nodes carry: orrery.yaml. A perk only ever works on the hero's
// side of a duel, and always in its node's region color. Its numbers are in the YAML.
const (
	PerkStrike		= "strike"		// a match of Size or more also deals Amount damage (Trade: instead of the extra turn)
	PerkMend		= "mend"		// a match of Size or more also heals Amount (Trade: instead of the extra turn)
	PerkWildFour	= "wild_four"	// a match of exactly four leaves an xAmount Wild instead of the extra turn
	PerkSkullPaint	= "skull_paint"	// a skull match turns Amount gems beside it into the color
	PerkOverflow	= "overflow"	// the color's max is Amount lower, and mana past it spills into To
	PerkForsake		= "forsake"		// skull matches deal Amount more, and the color's matches pay no mana
	PerkSeed		= "seed"		// the duel starts with Amount Wilds or skull Bombs (Special) on the board
)

var PerkKinds = []string{ PerkStrike, PerkMend, PerkWildFour, PerkSkullPaint, PerkOverflow, PerkForsake, PerkSeed }

// SEED_WILD_MULT is the multiplier of a Wild a seed perk plants.
const SEED_WILD_MULT = 2

// Perk is a rule a node gives. Color is its node's region.
type Perk struct {
	Kind	string	`yaml:"kind"`
	Size	int		`yaml:"size"`	// strike, mend: the shortest run that counts
	Amount	int		`yaml:"amount"`
	To		string	`yaml:"to"`		// overflow: the color that takes the spill
	Special	string	`yaml:"special"`	// seed: wild or bomb
	Trade	bool	`yaml:"trade"`	// strike, mend: works only on a run of four or more, which then gives no extra turn

	Node	string	`yaml:"-"`	// the node's ID, for its name
	Color	int		`yaml:"-"`
	ToMana	int		`yaml:"-"`
}

// checkPerk reports what is wrong with a perk's kind and parameters.
func checkPerk(p *Perk) (bad []string) {
	wrong := func(format string, a ...any) { bad = append(bad, fmt.Sprintf(format, a...)) }
	switch p.Kind {
	case PerkStrike, PerkMend:
		if p.Size < 3 || p.Size > 5 {
			wrong("%s size %d should be 3 to 5", p.Kind, p.Size)
		}
		if p.Amount <= 0 {
			wrong("%s needs an amount", p.Kind)
		}
		if p.Trade && p.Size < 4 {
			wrong("%s trade needs size 4 or more, as only those give an extra turn", p.Kind)
		}
	case PerkWildFour:
		if p.Amount < 2 || p.Amount > 3 {
			wrong("wild_four amount %d should be 2 or 3", p.Amount)
		}
	case PerkSkullPaint:
		if p.Amount < 1 || p.Amount > 4 {
			wrong("skull_paint amount %d should be 1 to 4", p.Amount)
		}
	case PerkOverflow:
		if p.ToMana < 0 || p.ToMana == p.Color {
			wrong("overflow to %q should be another color", p.To)
		}
		if p.Amount < 0 || p.Amount > BASE_MANA/2 {
			wrong("overflow amount %d should be 0 to %d", p.Amount, BASE_MANA/2)
		}
	case PerkForsake:
		if p.Amount <= 0 {
			wrong("forsake needs an amount")
		}
	case PerkSeed:
		if p.Special != "wild" && p.Special != "bomb" {
			wrong("seed special %q should be wild or bomb", p.Special)
		}
		if p.Amount < 1 || p.Amount > 4 {
			wrong("seed amount %d should be 1 to 4", p.Amount)
		}
	default:
		wrong("unknown perk kind %q", p.Kind)
	}
	return bad
}

// heroPerks is every perk on the nodes a hero owns.
func heroPerks(h *Hero) []*Perk {
	var out []*Perk
	for _, id := range h.Nodes {
		if n := NodeByID(id); n != nil && n.Perk != nil {
			out = append(out, n.Perk)
		}
	}
	return out
}

func (c *Combatant) perksOf(kind string) []*Perk {
	var out []*Perk
	for _, p := range c.Perks {
		if p.Kind == kind {
			out = append(out, p)
		}
	}
	return out
}

// HasPerk reports whether a side has a perk of this kind in this color.
func (c *Combatant) HasPerk(kind string, color int) bool {
	return slices.ContainsFunc(c.Perks, func(p *Perk) bool { return p.Kind == kind && p.Color == color })
}

// perkFired logs a perk working, once a cascade step.
func (d *Duel) perkFired(fired map[*Perk]bool, p *Perk) {
	if fired[p] {
		return
	}
	fired[p] = true
	if d.OnPerk != nil {
		d.OnPerk(p)
	}
	d.logf("log.perk", "%s: %s!", d.Sides[d.Turn].Name, PerkLabel(p))
}

// SetBoard gives the duel its board, and plants what the hero's seed perks start it with. A
// planted gem never makes a match by itself, so the board starts quiet.
func (d *Duel) SetBoard(b *board.Board) {
	d.Board = b
	me := d.Sides[Player]
	for _, p := range me.perksOf(PerkSeed) {
		planted := 0
		for _, at := range b.RandomCells(b.W() * b.H()) {
			if planted == p.Amount {
				break
			}
			g := b.At(at)
			if g == nil || g.Special != board.None || g.Locked {
				continue
			}
			was := *g
			if p.Special == "wild" {
				b.PlaceWild(at, SEED_WILD_MULT)
			} else {
				b.Paint(at, Skull, board.Bomb)
			}
			if b.HasMatches() {
				*g = was
				continue
			}
			planted++
		}
		if planted > 0 {
			d.logf("log.perk", "%s: %s!", me.Name, PerkLabel(p))
		}
	}
}

// paintBeside turns up to n gems next to the matched skulls of a step into color. It returns
// how many it turned.
func paintBeside(b *board.Board, res *board.StepResult, color, n int) int {
	done := 0
	for _, c := range res.Cleared {
		if !c.Matched || c.Gem.Color != Skull {
			continue
		}
		for _, dir := range []board.Point{ {X: 1}, {X: -1}, {Y: 1}, {Y: -1} } {
			if done == n {
				return done
			}
			g := b.At(c.Pos.Add(dir))
			if g == nil || g.Special != board.None || g.Locked || g.Color == color || g.Color == Skull {
				continue
			}
			g.Color = color
			done++
		}
	}
	return done
}
