// Package battle holds the turn-based RPG rules layered on top of a board.
// Like the board package it has no rendering; core drives it and animates the results.
package battle

import (
	"fmt"
	"jewel-game/board"
	"math"
	"math/rand"
)

// Role is what clearing a gem of a given color does in battle. Set per gem type in gemtypes.yaml.
type Role int

const (
	RoleNone Role = iota
	RoleAttack
	RoleMana
	RoleHeal
)

func ParseRole(s string) Role {
	switch s {
	case "attack":
		return RoleAttack
	case "mana":
		return RoleMana
	case "heal":
		return RoleHeal
	}
	return RoleNone
}

type State int

const (
	PlayerTurn State = iota
	Victory			// enemy down, waiting for a perk pick
	Defeat
)

type Player struct {
	HP, MaxHP		int
	Mana, MaxMana	int
	AttackPower		int		// damage per attack gem
	HealPower		int		// HP per heal gem
	ManaPower		int		// mana per mana gem
	Perks			[]string
}

// Mods are the board rules the player has unlocked through perks.
type Mods struct {
	Locks		int
	Diagonal	bool
	Overcharge	bool
	SpecialRate	float64
}

type Battle struct {
	Player		Player
	Enemy		*Enemy
	Mods		Mods
	Wave		int		// 1-based
	State		State
	Turns		int
	PerkChoices	[]Perk
	Log			[]string	// newest first
	rng			*rand.Rand
}

func New(seed int64) *Battle {
	b := &Battle{
		Player: Player{
			HP: 100, MaxHP: 100,
			MaxMana: 30,
			AttackPower: 3,
			HealPower: 2,
			ManaPower: 1,
		},
		Mods:	Mods{ Locks: 2 },
		rng:	rand.New(rand.NewSource(seed)),
	}
	b.startWave(1)
	return b
}

func (b *Battle) startWave(n int) {
	b.Wave = n
	b.State = PlayerTurn
	b.PerkChoices = nil
	b.Enemy = NewEnemy(n)
	b.logf("Wave %d: a %s appears!", n, b.Enemy.Name)
}

// BoardSettings is the board each fight is played on: all gem colors, plus the player's perks.
func (b *Battle) BoardSettings(numColors int) board.Settings {
	s := board.DefaultSettings()
	s.NumColors = numColors
	s.MaxLocks = b.Mods.Locks
	s.DiagonalSwap = b.Mods.Diagonal
	s.Overcharge = b.Mods.Overcharge
	s.SpecialRate = b.Mods.SpecialRate
	return s
}

// ChainMultiplier rewards cascades: x1, x1.5, x2, ...
func ChainMultiplier(chain int) float64 {
	return 1 + 0.5*float64(max(chain, 1)-1)
}

// Gain is what one cascade step did, for popups.
type Gain struct {
	Damage, Heal, Mana	int
	Multiplier			float64
}

// ApplyStep turns the gems cleared in one cascade step into damage, healing and mana.
// roles maps a gem color to its role; colors past the end of it do nothing.
func (b *Battle) ApplyStep(res *board.StepResult, roles []Role) Gain {
	g := Gain{ Multiplier: ChainMultiplier(res.Chain) }
	if b.State != PlayerTurn {
		return g
	}
	counts := map[Role]int{}
	for _, c := range res.Cleared {
		if c.Gem.Color >= 0 && c.Gem.Color < len(roles) {
			counts[roles[c.Gem.Color]]++
		}
	}
	scale := func(n, power int) int {
		return int(math.Round(float64(n*power) * g.Multiplier))
	}
	g.Damage = scale(counts[RoleAttack], b.Player.AttackPower)
	g.Heal = b.heal(scale(counts[RoleHeal], b.Player.HealPower))
	g.Mana = b.addMana(scale(counts[RoleMana], b.Player.ManaPower))
	b.hitEnemy(g.Damage)
	return g
}

func (b *Battle) heal(n int) int {
	n = min(n, b.Player.MaxHP-b.Player.HP)
	b.Player.HP += n
	return n
}

func (b *Battle) addMana(n int) int {
	n = min(n, b.Player.MaxMana-b.Player.Mana)
	b.Player.Mana += n
	return n
}

func (b *Battle) hitEnemy(n int) {
	if n <= 0 || b.Enemy.HP <= 0 {
		return
	}
	b.Enemy.HP = max(0, b.Enemy.HP-n)
	if b.Enemy.HP == 0 {
		b.State = Victory
		b.PerkChoices = b.rollPerks(3)
		b.logf("The %s is defeated!", b.Enemy.Name)
	}
}

// EndTurn is called once a player's move has finished resolving. It reports whether the
// enemy acts now; if so, call EnemyAct (the caller may pause first for drama).
func (b *Battle) EndTurn() bool {
	if b.State != PlayerTurn {
		return false
	}
	b.Turns++
	b.Enemy.Countdown--
	return b.Enemy.Countdown <= 0
}

// EnemyAct carries out the enemy's telegraphed move.
func (b *Battle) EnemyAct() Action {
	e := b.Enemy
	a := e.Intent()
	e.advance()
	if a.Damage > 0 {
		b.Player.HP = max(0, b.Player.HP-a.Damage)
	}
	if a.Kind == Drain {
		a.Drained = min(b.Player.Mana, a.Damage)
		b.Player.Mana -= a.Drained
	}
	b.logf("%s: %s", e.Name, a.Describe())
	if b.Player.HP == 0 {
		b.State = Defeat
		b.logf("You fall on wave %d.", b.Wave)
	}
	return a
}

// ChoosePerk applies a perk from the victory screen and starts the next wave.
func (b *Battle) ChoosePerk(i int) bool {
	if b.State != Victory || i < 0 || i >= len(b.PerkChoices) {
		return false
	}
	p := b.PerkChoices[i]
	p.apply(b)
	b.Player.Perks = append(b.Player.Perks, p.Name)
	b.logf("Perk: %s", p.Name)
	// a breather between fights
	b.heal(b.Player.MaxHP * 3 / 10)
	b.startWave(b.Wave + 1)
	return true
}

func (b *Battle) logf(format string, a ...any) {
	b.Log = append([]string{ fmt.Sprintf(format, a...) }, b.Log...)
	if len(b.Log) > 30 {
		b.Log = b.Log[:30]
	}
}
