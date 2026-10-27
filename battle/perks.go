package battle

import "fmt"

// Perk is a reward picked after each win. Several are sandbox rules turned into loot.
type Perk struct {
	Name	string
	Desc	string
	apply	func(*Battle)
	allowed	func(*Battle) bool
}

var perkPool = []Perk{
	{
		Name: "Vitality", Desc: "+25 max HP, and heal 25.",
		apply: func(b *Battle) { b.Player.MaxHP += 25; b.heal(25) },
	},
	{
		Name: "Sharpen", Desc: "Attack gems deal +1 damage.",
		apply: func(b *Battle) { b.Player.AttackPower++ },
	},
	{
		Name: "Remedy", Desc: "Heal gems restore +1 HP.",
		apply: func(b *Battle) { b.Player.HealPower++ },
	},
	{
		Name: "Focus", Desc: "Mana gems give +1 mana, and +10 max mana.",
		apply: func(b *Battle) { b.Player.ManaPower++; b.Player.MaxMana += 10 },
	},
	{
		Name: "Locksmith", Desc: "Hold one more locked gem at a time.",
		apply: func(b *Battle) { b.Mods.Locks++ },
		allowed: func(b *Battle) bool { return b.Mods.Locks < 8 },
	},
	{
		Name: "Slippery", Desc: "Swap gems diagonally.",
		apply: func(b *Battle) { b.Mods.Diagonal = true },
		allowed: func(b *Battle) bool { return !b.Mods.Diagonal },
	},
	{
		Name: "Overcharge", Desc: "Bombs blast 5x5, bolts sweep 3 lines.",
		apply: func(b *Battle) { b.Mods.Overcharge = true },
		allowed: func(b *Battle) bool { return !b.Mods.Overcharge },
	},
	{
		Name: "Lucky Refills", Desc: "+5% chance new gems arrive as specials.",
		apply: func(b *Battle) { b.Mods.SpecialRate += 0.05 },
		allowed: func(b *Battle) bool { return b.Mods.SpecialRate < 0.25 },
	},
}

func (b *Battle) rollPerks(n int) []Perk {
	var pool []Perk
	for _, p := range perkPool {
		if p.allowed == nil || p.allowed(b) {
			pool = append(pool, p)
		}
	}
	b.rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	return pool[:min(n, len(pool))]
}

type Skill int

const (
	Fireball Skill = iota
	Mend
	Conjure		// the board places a bomb; core does that part
)

var Skills = []Skill{ Fireball, Mend, Conjure }

func (s Skill) Cost() int {
	return [...]int{ 12, 10, 15 }[s]
}

func (s Skill) Name() string {
	return [...]string{ "Fireball", "Mend", "Conjure" }[s]
}

func (s Skill) Desc(p Player) string {
	switch s {
	case Fireball:
		return fmt.Sprintf("Deal %d damage", p.AttackPower*8)
	case Mend:
		return fmt.Sprintf("Heal %d HP", p.HealPower*10)
	}
	return "Make a Bomb"
}

func (b *Battle) CanCast(s Skill) bool {
	return b.State == PlayerTurn && b.Player.Mana >= s.Cost()
}

// Cast spends mana on a skill. Casting doesn't use up the turn.
func (b *Battle) Cast(s Skill) bool {
	if !b.CanCast(s) {
		return false
	}
	b.Player.Mana -= s.Cost()
	switch s {
	case Fireball:
		b.logf("Fireball for %d!", b.Player.AttackPower*8)
		b.hitEnemy(b.Player.AttackPower * 8)
	case Mend:
		b.logf("Mend heals %d.", b.heal(b.Player.HealPower*10))
	case Conjure:
		b.logf("A bomb takes shape.")
	}
	return true
}
