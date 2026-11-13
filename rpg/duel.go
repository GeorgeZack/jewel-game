package rpg

import (
	"jewel-game/board"
	"jewel-game/text"
	"math/rand"
	"slices"
)

const (
	BASE_MANA		= 20	// max of each color before mastery
	SKULL_DAMAGE	= 2		// per skull cleared
	BIG_SKULL_BONUS	= 8		// extra damage from a +5 skull, which also blasts its surroundings
	BIG_SKULL_RATE	= 0.12	// share of refilled skulls that are +5 skulls
)

const (
	Player = 0
	Enemy  = 1
)

// Combatant is one side of a duel, built from a hero or a monster.
type Combatant struct {
	Name			string
	Level			int
	Stats			[NUM_STATS]int
	HP, MaxHP		int
	Mana, MaxMana	[NUM_MANA]int
	Spells			[]*Spell
	Cooldowns		map[string]int
	Effects			[]Effect
	SkullDamage		int
	Armor			int
	GoldPct, XPPct	int
	Gold, XP		int		// collected this duel
	Residue			[NUM_MANA]int	// collected this duel, on the hero's turns only
	Foe				*Foe	// nil for the hero
	Perks			[]*Perk	// from the hero's Orrery nodes; foes have none
}

func newCombatant(name string, level int, stats [NUM_STATS]int, hp, startMana int, spells []*Spell) *Combatant {
	c := &Combatant{ Name: name, Level: level, Stats: stats, Spells: spells, Cooldowns: map[string]int{} }
	c.MaxHP = hp + stats[Morale]*3
	c.HP = c.MaxHP
	for i := range c.Mana {
		c.MaxMana[i] = BASE_MANA + stats[i]
		c.Mana[i] = min(startMana, c.MaxMana[i])
	}
	return c
}

// HeroHP is a hero's HP before Morale and gear.
func HeroHP(level int) int { return 50 + 10*level }

// NewHeroCombatant applies gear, facilities and the Orrery to the hero's stats.
func NewHeroCombatant(h *Hero) *Combatant {
	b := h.TotalBonus()
	stats := h.Stats
	for i := range stats {
		stats[i] += b.Stats[i]
	}
	var spells []*Spell
	for _, id := range h.Spells {
		if s := SpellByID(id); s != nil {
			spells = append(spells, s)
		}
	}
	c := newCombatant(ClassByName(h.Class).Label(), h.Level, stats, HeroHP(h.Level)+b.HP, b.StartMana, spells)
	c.SkullDamage, c.Armor, c.GoldPct, c.XPPct = b.SkullDamage, b.Armor, b.GoldPct, b.XPPct
	c.Perks = heroPerks(h)
	for _, p := range c.perksOf(PerkOverflow) {
		c.MaxMana[p.Color] -= p.Amount
		c.Mana[p.Color] = min(c.Mana[p.Color], c.MaxMana[p.Color])
	}
	return c
}

func NewFoeCombatant(f *Foe, level int) *Combatant {
	var stats [NUM_STATS]int
	for i := range stats {
		stats[i] = f.Stats[i] + (level-1)*f.Growth[i]/2
	}
	var spells []*Spell
	for _, id := range f.Spells {
		spells = append(spells, SpellByID(id))
	}
	c := newCombatant(f.Name, level, stats, (30+8*level)*f.HPPct/100, 0, spells)
	c.Foe = f
	return c
}

func (c *Combatant) TotalMana() int {
	return c.Mana[0] + c.Mana[1] + c.Mana[2] + c.Mana[3]
}

// AddEffect refreshes an effect of the same kind rather than stacking it.
func (c *Combatant) AddEffect(e Effect) {
	for i := range c.Effects {
		if c.Effects[i].Kind == e.Kind {
			c.Effects[i] = e
			return
		}
	}
	c.Effects = append(c.Effects, e)
}

func (c *Combatant) effect(k EffectKind) *Effect {
	for i := range c.Effects {
		if c.Effects[i].Kind == k {
			return &c.Effects[i]
		}
	}
	return nil
}

func (c *Combatant) power(k EffectKind) int {
	if e := c.effect(k); e != nil {
		return e.Power
	}
	return 0
}

// Duel is one fight on a board both sides play on, taking turns.
type Duel struct {
	Sides	[2]*Combatant
	Turn	int
	Winner	int		// -1 while the fight is on
	Turns	int
	Extra	bool	// the side to move goes again when this turn ends
	Log		[]LogLine	// newest first
	Board	*board.Board
	Card	*RuleCard	// nil for the standard duel
	OnHit	func(side, amount int)	// lets the view show damage from any source
	OnHeal	func(side, amount int)
	OnPerk	func(p *Perk)	// lets the view mark a hero perk that just worked
	rng		*rand.Rand
}

func NewDuel(a, b *Combatant, seed int64) *Duel {
	d := &Duel{ Sides: [2]*Combatant{ a, b }, Winner: -1, rng: rand.New(rand.NewSource(seed)) }
	// initiative: higher Cunning usually moves first
	if d.rng.Intn(10+a.Stats[Cunning]) < d.rng.Intn(10+b.Stats[Cunning]) {
		d.Turn = Enemy
	}
	d.logf("log.moves_first", "%s moves first.", d.Sides[d.Turn].Name)
	return d
}

func (d *Duel) Side(c *Combatant) int {
	if d.Sides[Player] == c {
		return Player
	}
	return Enemy
}

func (d *Duel) Over() bool { return d.Winner >= 0 }

// Damage hits a combatant. Shields reduce every hit; at least 1 always lands.
func (d *Duel) Damage(c *Combatant, n int) int {
	if n <= 0 || d.Over() {
		return 0
	}
	n = min(max(1, n-c.power(Shield)), c.HP)
	c.HP -= n
	if d.OnHit != nil {
		d.OnHit(d.Side(c), n)
	}
	if c.HP == 0 {
		d.Winner = 1 - d.Side(c)
		d.logf("log.falls", "%s falls!", c.Name)
	}
	return n
}

func (d *Duel) Heal(c *Combatant, n int) int {
	n = min(n, c.MaxHP-c.HP)
	c.HP += n
	if n > 0 && d.OnHeal != nil {
		d.OnHeal(d.Side(c), n)
	}
	return n
}

// Gain is what one cascade step gave the side to move, for popups.
type Gain struct {
	Side	int
	Mana	[NUM_MANA]int
	Damage	int
	Gold	int
	XP		int
	Residue	[NUM_MANA]int
	Extra	bool
}

// MatchResidue is what one mana match pays in its color: a plain three on the first step pays
// nothing, longer runs and later cascade steps pay more.
func MatchResidue(size, chain int) int {
	n := max(chain-1, 0) * RESIDUE_CHAIN
	switch {
	case size >= 5:
		n += RESIDUE_FIVE
	case size == 4:
		n += RESIDUE_FOUR
	}
	return n
}

// ApplyStep credits a cascade step to whoever's turn it is, whether a swap or a spell started it.
func (d *Duel) ApplyStep(res *board.StepResult) Gain {
	me, foe := d.Sides[d.Turn], d.Sides[1-d.Turn]
	g := Gain{ Side: d.Turn }
	if d.Over() {
		return g
	}
	var counts [NUM_GEMS]int
	bigSkulls := 0
	for _, c := range res.Cleared {
		if c.Gem.Color >= 0 && c.Gem.Color < NUM_GEMS {
			counts[c.Gem.Color]++
			if c.Gem.Color == Skull && c.Gem.Special == board.Bomb {
				bigSkulls++
			}
		}
	}
	skullMatches := 0
	fired := map[*Perk]bool{}
	for _, m := range res.Matches {
		traded := false
		if m.Color < NUM_MANA {
			traded = d.matchPerks(me, foe, m, fired)
		}
		if m.Size >= 4 && !d.wildFour(me, m, fired) && !traded {
			g.Extra = true
		}
		switch {
		case m.Color < NUM_MANA:
			if me.Foe == nil {
				n := MatchResidue(m.Size, res.Chain)
				me.Residue[m.Color] += n
				g.Residue[m.Color] += n
			}
			// the Wild itself isn't in Cleared by color; count it, then multiply the run
			counts[m.Color] += m.Size*m.Mult - (m.Size - m.Wilds)
			counts[m.Color] += me.Stats[m.Color] / 6
		case m.Color == Skull:
			skullMatches++
		}
	}

	card := RuleCard{}
	if d.Card != nil {
		card = *d.Card
	}
	for _, p := range me.perksOf(PerkForsake) {
		if counts[p.Color] > 0 {
			counts[p.Color] = 0
			d.perkFired(fired, p)
		}
	}
	var spill [NUM_MANA]int
	for i := range me.Mana {
		got := pct(counts[i], card.ManaPct)
		n := min(got, me.MaxMana[i]-me.Mana[i])
		me.Mana[i] += n
		g.Mana[i] = n
		for _, p := range me.perksOf(PerkOverflow) {
			if p.Color == i && got > n {
				spill[p.ToMana] += got - n
				d.perkFired(fired, p)
			}
		}
	}
	for i, s := range spill {
		n := min(s, me.MaxMana[i]-me.Mana[i])
		me.Mana[i] += n
		g.Mana[i] += n
	}
	g.Gold = pct(counts[Gold] * (2 + me.Level/2) * (100 + me.GoldPct + 2*me.Stats[Cunning]) / 100, card.GoldPct)
	g.XP = pct(counts[Star] * (1 + me.Level/3) * (100 + me.XPPct + 2*me.Stats[Cunning]) / 100, card.XPPct)
	me.Gold += g.Gold
	me.XP += g.XP

	if counts[Skull] > 0 {
		perMatch := me.Level/2 + me.SkullDamage + me.Stats[Battle]/4 + me.power(Might) - foe.Armor - foe.Stats[Morale]/10
		for _, p := range me.perksOf(PerkForsake) {
			if skullMatches > 0 {
				perMatch += p.Amount
				d.perkFired(fired, p)
			}
		}
		dmg := counts[Skull]*SKULL_DAMAGE + bigSkulls*BIG_SKULL_BONUS + perMatch*max(skullMatches, 1)
		g.Damage = d.Damage(foe, max(1, pct(dmg, card.SkullPct)))
	}
	if skullMatches > 0 && d.Board != nil {
		for _, p := range me.perksOf(PerkSkullPaint) {
			if paintBeside(d.Board, res, p.Color, p.Amount*skullMatches) > 0 {
				d.perkFired(fired, p)
			}
		}
	}
	if g.Extra && !d.Over() {
		d.Extra = true
		d.logf("log.extra_turn", "%s earns an extra turn!", me.Name)
	}
	return g
}

// wildFour handles a hero's wild_four perk for a run of four: it plants the Wild and reports that
// the run gives no extra turn.
func (d *Duel) wildFour(me *Combatant, m board.Match, fired map[*Perk]bool) bool {
	if m.Size != 4 || m.Color >= NUM_MANA {
		return false
	}
	for _, p := range me.perksOf(PerkWildFour) {
		if p.Color == m.Color {
			if d.Board != nil {
				if at := d.Board.RandomCells(1); len(at) > 0 {
					d.Board.PlaceWild(at[0], p.Amount)
				}
			}
			d.perkFired(fired, p)
			return true
		}
	}
	return false
}

// matchPerks runs a hero's strike and mend perks for one mana run, and reports whether a trade
// perk took the run's extra turn.
func (d *Duel) matchPerks(me, foe *Combatant, m board.Match, fired map[*Perk]bool) (traded bool) {
	for _, p := range me.Perks {
		if p.Color != m.Color || m.Size < p.Size {
			continue
		}
		switch p.Kind {
		case PerkStrike:
			d.Damage(foe, p.Amount)
		case PerkMend:
			d.Heal(me, p.Amount)
		default:
			continue
		}
		d.perkFired(fired, p)
		traded = traded || p.Trade
	}
	return traded
}

// CanCast reports whether a side could cast a spell now, ignoring whose turn it is.
func (c *Combatant) CanCast(s *Spell) bool {
	if c.effect(Silence) != nil || c.Cooldowns[s.ID] > 0 {
		return false
	}
	for i, n := range s.Cost {
		if c.Mana[i] < n {
			return false
		}
	}
	return true
}

// Cast pays for and runs a spell on the side to move. The caller then resolves the board and
// ends the turn as usual; a free spell sets Extra so the same side moves again.
func (d *Duel) Cast(s *Spell) bool {
	me := d.Sides[d.Turn]
	if d.Over() || !me.CanCast(s) || !slices.Contains(me.Spells, s) {
		return false
	}
	for i, n := range s.Cost {
		me.Mana[i] -= n
	}
	if s.Cooldown > 0 {
		me.Cooldowns[s.ID] = s.Cooldown
	}
	d.logf("log.casts", "%s casts %s.", me.Name, s.Label())
	if s.Free {
		d.Extra = true
	}
	s.Cast(Ctx{ D: d, Me: me, Foe: d.Sides[1-d.Turn], Board: d.Board })
	return true
}

// EndTurn runs once a move has resolved: it hands over the board unless the side earned an
// extra turn, applying the new side's poison, regen, cooldowns and stuns.
func (d *Duel) EndTurn() {
	if d.Over() {
		return
	}
	if d.Extra {
		d.Extra = false
		return
	}
	// a stun can pass the turn straight back; two rounds always settle it since stuns run out
	for range 8 {
		d.Turn = 1 - d.Turn
		d.Turns++
		stunned := d.startTurn(d.Sides[d.Turn])
		if !stunned || d.Over() {
			return
		}
		d.logf("log.stunned", "%s is stunned!", d.Sides[d.Turn].Name)
	}
}

func (d *Duel) startTurn(c *Combatant) (stunned bool) {
	for id, n := range c.Cooldowns {
		if n > 0 {
			c.Cooldowns[id] = n - 1
		}
	}
	stunned = c.effect(Stun) != nil
	d.Damage(c, c.power(Poison))
	d.Heal(c, c.power(Regen))
	kept := c.Effects[:0]
	for _, e := range c.Effects {
		if e.Turns--; e.Turns > 0 {
			kept = append(kept, e)
		}
	}
	c.Effects = kept
	return stunned
}

// ManaDrain cuts both sides' mana in half when the board runs out of moves, as it reshuffles.
func (d *Duel) ManaDrain() {
	for _, c := range d.Sides {
		for i := range c.Mana {
			c.Mana[i] /= 2
		}
	}
	d.logf("log.mana_drain", "No moves left: half your mana drains away!")
}

// LogLine is one battle-log entry. It is formatted when drawn, so editing
// the language file changes lines that are already on screen.
type LogLine struct {
	Key, Fallback string
	Args          []any
}

func (l LogLine) Text() string { return text.Tf(l.Key, l.Fallback, l.Args...) }

func (d *Duel) logf(key, fallback string, a ...any) {
	d.Log = append([]LogLine{{ Key: key, Fallback: fallback, Args: a }}, d.Log...)
	if len(d.Log) > 30 {
		d.Log = d.Log[:30]
	}
}

// Reward is what the hero banks from a finished duel.
type Reward struct {
	Won			bool
	Gold, XP	int
	Residue		[NUM_MANA]int
	Levels		int
}

// Settle banks a finished duel. Whatever was collected from the board, residue included, is kept
// either way, as a loss costs nothing but the time; only a win pays the bounty.
func (d *Duel) Settle(h *Hero) Reward {
	me, foe := d.Sides[Player], d.Sides[Enemy]
	r := Reward{ Won: d.Winner == Player, Gold: me.Gold, XP: me.XP, Residue: me.Residue }
	if r.Won {
		// a bounty in the seat's color, so a hero who wins fast isn't starved of their own region
		if seat := SeatOf(h.Class); seat != nil && seat.Mana >= 0 {
			r.Residue[seat.Mana] += WIN_RESIDUE
		}
		h.Wins++
		r.Gold += foe.Level * 8 * (100 + me.GoldPct + 2*me.Stats[Cunning]) / 100
		r.XP += foe.Level * 12 * (100 + me.XPPct + 2*me.Stats[Cunning]) / 100
	} else {
		h.Losses++
	}
	h.Gold += r.Gold
	for i, n := range r.Residue {
		h.Residue[i] += n
	}
	r.Levels = h.AddXP(r.XP)
	return r
}
