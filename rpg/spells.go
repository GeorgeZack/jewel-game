package rpg

import "jewel-game/board"

// EffectKind is a status effect on a combatant. Durations count the owner's turns.
type EffectKind int

const (
	Stun	EffectKind = iota	// skips turns
	Poison						// loses Power HP each turn
	Regen						// gains Power HP each turn
	Shield						// takes Power less from each hit
	Might						// skull matches deal Power more
	Silence						// can't cast
	EFFECT_COUNT
)

var EffectNames = [EFFECT_COUNT]string{ "Stunned", "Poisoned", "Regen", "Shielded", "Might", "Silenced" }

type Effect struct {
	Kind	EffectKind
	Turns	int
	Power	int
}

// Ctx is what a spell works with when cast.
type Ctx struct {
	D			*Duel
	Me, Foe		*Combatant
	Board		*board.Board
}

func (c Ctx) mastery(i, per int) int { return c.Me.Stats[i] / per }

// Spell costs mana of up to NUM_MANA colors. Casting ends the turn unless Free is set.
type Spell struct {
	ID, Name	string
	Class		string		// empty: monsters only
	Level		int			// hero level needed
	Cost		[NUM_MANA]int
	Cooldown	int			// the caster's turns before it can be cast again
	Free		bool
	Desc		string
	Cast		func(c Ctx)
	Worth		func(c Ctx) bool	// optional AI check for whether casting is useful now
}

// cost is a spell's price in Fire, Water, Air, Earth and Aether order; colors left off cost nothing.
func cost(mana ...int) (c [NUM_MANA]int) {
	copy(c[:], mana)
	return c
}

// AETHER_RITES is the Class of the Aether rites: they belong to no class, so none knows them
// without the Orrery.
const AETHER_RITES = "Aether"

// OBLIVION_BACKLASH is the HP Oblivion costs its caster. A placeholder.
const OBLIVION_BACKLASH = 8

func hurt(c Ctx, n int)		{ c.D.Damage(c.Foe, n) }
func wounded(c Ctx) bool	{ return c.Me.HP*3 < c.Me.MaxHP*2 }
func foeHasMana(c Ctx) bool { return c.Foe.TotalMana() >= 8 }

// clearColor destroys one gem color, crediting the caster as usual for what breaks.
func clearColor(c Ctx, color int) { c.Board.Force(c.Board.CellsOfColor(color)) }

var Spells = []Spell{
	// Knight
	{ ID: "smite", Name: "Rebuke", Class: "Knight", Level: 1, Cost: cost(0, 0, 0, 6), Cooldown: 1,
		Desc: "Deal 6 + Force/2 damage.",
		Cast: func(c Ctx) { hurt(c, 6+c.mastery(Battle, 2)) } },
	{ ID: "bulwark", Name: "Salt Circle", Class: "Knight", Level: 2, Cost: cost(0, 6, 0, 2), Cooldown: 6, Free: true,
		Desc: "Free. Block 3 per hit for 4 turns.",
		Cast: func(c Ctx) { c.Me.AddEffect(Effect{ Shield, 4, 3 }) } },
	{ ID: "lay_hands", Name: "Absolution", Class: "Knight", Level: 4, Cost: cost(0, 10, 4, 0), Cooldown: 4,
		Desc: "Heal 15 + Nerve.", Worth: wounded,
		Cast: func(c Ctx) { c.D.Heal(c.Me, 15+c.Me.Stats[Morale]) } },
	{ ID: "shield_bash", Name: "Sanctus Bell", Class: "Knight", Level: 6, Cost: cost(4, 0, 0, 10), Cooldown: 7,
		Desc: "Deal 5 and stun the foe for 1 turn.",
		Cast: func(c Ctx) { hurt(c, 5); c.Foe.AddEffect(Effect{ Stun, 1, 0 }) } },
	{ ID: "muster", Name: "Inhumation", Class: "Knight", Level: 9, Cost: cost(8, 0, 8, 0), Cooldown: 5,
		Desc: "Destroy all skulls on the board.",
		Cast: func(c Ctx) { clearColor(c, Skull) } },
	{ ID: "judgement", Name: "Anathema", Class: "Knight", Level: 13, Cost: cost(10, 0, 10, 10), Cooldown: 6,
		Desc: "Deal 20 + Force damage.",
		Cast: func(c Ctx) { hurt(c, 20+c.Me.Stats[Battle]) } },

	// Warrior
	{ ID: "cleave", Name: "Hot Iron", Class: "Warrior", Level: 1, Cost: cost(6, 0, 0, 0), Cooldown: 1,
		Desc: "Deal 5 + Sulphur/3 damage.",
		Cast: func(c Ctx) { hurt(c, 5+c.mastery(Fire, 3)) } },
	{ ID: "rage", Name: "Blood Up", Class: "Warrior", Level: 2, Cost: cost(8, 0, 0, 0), Cooldown: 6, Free: true,
		Desc: "Free. Skull matches deal +4 for 4 turns.",
		Cast: func(c Ctx) { c.Me.AddEffect(Effect{ Might, 4, 4 }) } },
	{ ID: "war_cry", Name: "Garrotte", Class: "Warrior", Level: 4, Cost: cost(0, 0, 8, 4), Cooldown: 5,
		Desc: "Turn all Nitre into skulls.",
		Cast: func(c Ctx) { c.Board.Convert(Air, Skull) } },
	{ ID: "hamstring", Name: "Hamstring", Class: "Warrior", Level: 6, Cost: cost(10, 0, 4, 0), Cooldown: 7,
		Desc: "Deal 4 and stun the foe for 1 turn.",
		Cast: func(c Ctx) { hurt(c, 4); c.Foe.AddEffect(Effect{ Stun, 1, 0 }) } },
	{ ID: "whirlwind", Name: "Brawl", Class: "Warrior", Level: 9, Cost: cost(10, 0, 8, 0), Cooldown: 5,
		Desc: "Destroy a random row and column.",
		Cast: func(c Ctx) {
			p := c.Board.RandomCells(1)[0]
			c.Board.Force(c.Board.Rows(p.Y, p.Y))
			c.Board.Force(c.Board.Cols(p.X, p.X))
		} },
	{ ID: "execute", Name: "Last Rites", Class: "Warrior", Level: 13, Cost: cost(15, 0, 0, 10), Cooldown: 6,
		Desc: "Deal 10 + a quarter of what the foe has lost.",
		Cast: func(c Ctx) { hurt(c, 10+(c.Foe.MaxHP-c.Foe.HP)/4) } },

	// Druid
	{ ID: "thorns", Name: "Nightshade", Class: "Druid", Level: 1, Cost: cost(0, 0, 0, 6), Cooldown: 3,
		Desc: "Poison: 2 + Salt/6 per turn for 5 turns.",
		Cast: func(c Ctx) { c.Foe.AddEffect(Effect{ Poison, 5, 2 + c.mastery(Earth, 6) }) } },
	{ ID: "renewal", Name: "Poultice", Class: "Druid", Level: 2, Cost: cost(0, 6, 0, 4), Cooldown: 5, Free: true,
		Desc: "Free. Mend 4 per turn for 5 turns.", Worth: wounded,
		Cast: func(c Ctx) { c.Me.AddEffect(Effect{ Regen, 5, 4 }) } },
	{ ID: "overgrowth", Name: "Smothering", Class: "Druid", Level: 4, Cost: cost(0, 0, 4, 8), Cooldown: 4,
		Desc: "Turn all Sulphur into Salt.",
		Cast: func(c Ctx) { c.Board.Convert(Fire, Earth) } },
	{ ID: "entangle", Name: "Laudanum", Class: "Druid", Level: 6, Cost: cost(0, 4, 0, 12), Cooldown: 7,
		Desc: "Stun the foe for 2 turns.",
		Cast: func(c Ctx) { c.Foe.AddEffect(Effect{ Stun, 2, 0 }) } },
	{ ID: "wild_seed", Name: "Fairy Ring", Class: "Druid", Level: 9, Cost: cost(0, 8, 0, 10), Cooldown: 5,
		Desc: "Plant two x2 Catalysts on the board.",
		Cast: func(c Ctx) {
			for _, p := range c.Board.RandomCells(2) {
				c.Board.PlaceWild(p, 2)
			}
		} },
	{ ID: "earthquake", Name: "Subsidence", Class: "Druid", Level: 13, Cost: cost(0, 0, 8, 18), Cooldown: 6,
		Desc: "Destroy the bottom 3 rows.",
		Cast: func(c Ctx) { h := c.Board.H(); c.Board.Force(c.Board.Rows(h-3, h-1)) } },

	// Wizard
	{ ID: "firebolt", Name: "Phosphorus", Class: "Wizard", Level: 1, Cost: cost(7, 0, 0, 0), Cooldown: 1,
		Desc: "Deal 5 + Sulphur/2 damage.",
		Cast: func(c Ctx) { hurt(c, 5+c.mastery(Fire, 2)) } },
	{ ID: "mana_siphon", Name: "Siphon", Class: "Wizard", Level: 2, Cost: cost(0, 5, 0, 0), Cooldown: 4, Free: true,
		Desc: "Free. Steal up to 4 of each reagent.", Worth: foeHasMana,
		Cast: func(c Ctx) {
			for i := range c.Foe.Mana {
				n := min(4, c.Foe.Mana[i])
				c.Foe.Mana[i] -= n
				c.Me.Mana[i] = min(c.Me.MaxMana[i], c.Me.Mana[i]+n)
			}
		} },
	{ ID: "transmute", Name: "Transmute", Class: "Wizard", Level: 4, Cost: cost(0, 6, 6, 0), Cooldown: 4,
		Desc: "Turn all skulls into gold.",
		Cast: func(c Ctx) { c.Board.Convert(Skull, Gold) } },
	{ ID: "chain_lightning", Name: "Galvanic Arc", Class: "Wizard", Level: 6, Cost: cost(0, 0, 14, 0), Cooldown: 5,
		Desc: "Destroy all Nitre. Deal 2 per Nitre.",
		Cast: func(c Ctx) {
			hurt(c, 2*len(c.Board.CellsOfColor(Air)))
			clearColor(c, Air)
		} },
	{ ID: "hush", Name: "Seal of Silence", Class: "Wizard", Level: 9, Cost: cost(0, 12, 6, 0), Cooldown: 7,
		Desc: "Silence the foe for 3 turns.",
		Cast: func(c Ctx) { c.Foe.AddEffect(Effect{ Silence, 3, 0 }) } },
	{ ID: "meteor", Name: "Fulmination", Class: "Wizard", Level: 13, Cost: cost(22, 0, 8, 0), Cooldown: 6,
		Desc: "Deal 15 + Sulphur. Destroy a 5x5 area.",
		Cast: func(c Ctx) {
			hurt(c, 15+c.Me.Stats[Fire])
			c.Board.Force(c.Board.Square(c.Board.RandomCells(1)[0], 2))
		} },

	// Aether: on the Orrery's Aether region, for any class that reaches it
	{ ID: "void_bolt", Name: "Pierce the Veil", Class: AETHER_RITES, Level: 6, Cost: cost(0, 0, 0, 0, 8), Cooldown: 2,
		Desc: "Deal 8 + {aether}/2 damage.",
		Cast: func(c Ctx) { hurt(c, 8+c.mastery(Aether, 2)) } },
	{ ID: "unmake", Name: "Unmake", Class: AETHER_RITES, Level: 8, Cost: cost(0, 0, 0, 0, 12), Cooldown: 5,
		Desc: "Turn all gold and stars into skulls.",
		Cast: func(c Ctx) { c.Board.Convert(Gold, Skull); c.Board.Convert(Star, Skull) } },
	{ ID: "eclipse", Name: "Eclipse", Class: AETHER_RITES, Level: 10, Cost: cost(0, 6, 0, 0, 10), Cooldown: 7,
		Desc: "Silence the foe for 2 turns and burn 5 of each of its reagents.",
		Cast: func(c Ctx) {
			c.Foe.AddEffect(Effect{ Silence, 2, 0 })
			for i := range c.Foe.Mana {
				c.Foe.Mana[i] = max(0, c.Foe.Mana[i]-5)
			}
		} },
	{ ID: "oblivion", Name: "Oblivion", Class: AETHER_RITES, Level: 13, Cost: cost(6, 0, 0, 0, 16), Cooldown: 6,
		Desc: "Deal 25 + {aether} damage. You lose 8 HP.", Worth: func(c Ctx) bool { return c.Me.HP > 24 },
		Cast: func(c Ctx) { hurt(c, 25+c.Me.Stats[Aether]); c.D.Backlash(c.Me, OBLIVION_BACKLASH) } },

	// Monsters
	{ ID: "bite", Name: "Cold Touch", Level: 1, Cost: cost(0, 0, 0, 5), Cooldown: 3, Desc: "Deal 4 damage.",
		Cast: func(c Ctx) { hurt(c, 4) } },
	{ ID: "venom", Name: "Miasma", Level: 1, Cost: cost(0, 0, 0, 7), Cooldown: 6, Desc: "Poison: 2 per turn for 4 turns.",
		Cast: func(c Ctx) { c.Foe.AddEffect(Effect{ Poison, 4, 2 }) } },
	{ ID: "firebreath", Name: "Balefire", Level: 1, Cost: cost(12, 0, 0, 0), Cooldown: 4, Desc: "Deal 8 + Sulphur/2 damage.",
		Cast: func(c Ctx) { hurt(c, 8+c.mastery(Fire, 2)) } },
	{ ID: "drain_life", Name: "Drain Life", Level: 1, Cost: cost(0, 8, 4, 0), Cooldown: 4, Desc: "Deal 8 and heal 8.",
		Cast: func(c Ctx) { hurt(c, 8); c.D.Heal(c.Me, 8) } },
	{ ID: "mana_burn", Name: "Curdle", Level: 1, Cost: cost(0, 6, 4, 0), Cooldown: 5, Desc: "Burn up to 6 of each foe reagent.", Worth: foeHasMana,
		Cast: func(c Ctx) {
			for i := range c.Foe.Mana {
				c.Foe.Mana[i] = max(0, c.Foe.Mana[i]-6)
			}
		} },
	{ ID: "bone_ward", Name: "Bone Ward", Level: 1, Cost: cost(0, 0, 0, 8), Cooldown: 7, Free: true, Desc: "Free. Block 2 per hit for 4 turns.",
		Cast: func(c Ctx) { c.Me.AddEffect(Effect{ Shield, 4, 2 }) } },
	{ ID: "roar", Name: "Keening", Level: 1, Cost: cost(6, 0, 8, 0), Cooldown: 7, Desc: "Stun the foe for 1 turn.",
		Cast: func(c Ctx) { c.Foe.AddEffect(Effect{ Stun, 1, 0 }) } },
	{ ID: "frenzy", Name: "Bloodlust", Level: 1, Cost: cost(8, 0, 0, 0), Cooldown: 6, Free: true, Desc: "Free. Skull matches deal +3 for 4 turns.",
		Cast: func(c Ctx) { c.Me.AddEffect(Effect{ Might, 4, 3 }) } },
	{ ID: "hollow_hymn", Name: "Hollow Hymn", Level: 1, Cost: cost(0, 4, 0, 0, 8), Cooldown: 5, Desc: "Deal 6 and burn 4 of each foe reagent.",
		Cast: func(c Ctx) {
			hurt(c, 6)
			for i := range c.Foe.Mana {
				c.Foe.Mana[i] = max(0, c.Foe.Mana[i]-4)
			}
		} },
	{ ID: "void_lash", Name: "Veil Lash", Level: 1, Cost: cost(0, 0, 0, 0, 10), Cooldown: 4, Desc: "Deal 10 + {aether}/2 damage.",
		Cast: func(c Ctx) { hurt(c, 10+c.mastery(Aether, 2)) } },
	{ ID: "dark_pact", Name: "Dark Pact", Level: 1, Cost: cost(5, 5, 5, 5), Cooldown: 5, Desc: "Turn all gold into skulls.",
		Cast: func(c Ctx) { c.Board.Convert(Gold, Skull) } },
}

func SpellByID(id string) *Spell {
	for i := range Spells {
		if Spells[i].ID == id {
			return &Spells[i]
		}
	}
	return nil
}
