package battle

import (
	"fmt"
	"image/color"
	"math"
)

type ActionKind int

const (
	Strike	ActionKind = iota
	Heavy			// double damage
	Drain			// damage, and steals as much mana
	Quake			// half damage, and the board's gravity shifts
)

type Action struct {
	Kind	ActionKind
	Damage	int
	Drained	int
}

func (a Action) Describe() string {
	switch a.Kind {
	case Heavy:
		return fmt.Sprintf("smashes for %d", a.Damage)
	case Drain:
		return fmt.Sprintf("drains %d HP and %d mana", a.Damage, a.Drained)
	case Quake:
		return fmt.Sprintf("quakes for %d, gravity shifts!", a.Damage)
	}
	return fmt.Sprintf("strikes for %d", a.Damage)
}

// Short is the intent label shown above the enemy.
func (a Action) Short() string {
	return [...]string{ "Strike", "Smash", "Drain", "Quake" }[a.Kind] + fmt.Sprintf(" %d", a.Damage)
}

type EnemyKind struct {
	Name		string
	HP			int
	Attack		int
	Interval	int		// player turns between actions
	Pattern		[]ActionKind
	Color		color.RGBA
}

var enemyKinds = []EnemyKind{
	{ Name: "Slime", HP: 45, Attack: 6, Interval: 2, Pattern: []ActionKind{ Strike }, Color: color.RGBA{ 90, 200, 110, 255 } },
	{ Name: "Goblin", HP: 80, Attack: 8, Interval: 2, Pattern: []ActionKind{ Strike, Strike, Heavy }, Color: color.RGBA{ 150, 170, 60, 255 } },
	{ Name: "Wraith", HP: 110, Attack: 8, Interval: 2, Pattern: []ActionKind{ Drain, Strike }, Color: color.RGBA{ 140, 120, 220, 255 } },
	{ Name: "Golem", HP: 170, Attack: 12, Interval: 3, Pattern: []ActionKind{ Quake, Heavy }, Color: color.RGBA{ 150, 130, 110, 255 } },
	{ Name: "Dragon", HP: 260, Attack: 13, Interval: 2, Pattern: []ActionKind{ Strike, Quake, Heavy, Drain }, Color: color.RGBA{ 220, 70, 50, 255 } },
}

var titles = []string{ "Elder", "Ancient", "Mythic", "Eternal" }

type Enemy struct {
	EnemyKind
	MaxHP, HP	int
	Countdown	int		// player turns until the next action
	step		int
}

// NewEnemy builds the foe for a wave. After the last kind it loops, tougher each lap.
func NewEnemy(wave int) *Enemy {
	i := (wave - 1) % len(enemyKinds)
	lap := (wave - 1) / len(enemyKinds)
	kind := enemyKinds[i]
	if lap > 0 {
		scale := math.Pow(1.6, float64(lap))
		kind.Name = titles[min(lap-1, len(titles)-1)] + " " + kind.Name
		kind.HP = int(float64(kind.HP) * scale)
		kind.Attack = int(float64(kind.Attack) * scale)
	}
	return &Enemy{ EnemyKind: kind, MaxHP: kind.HP, HP: kind.HP, Countdown: kind.Interval }
}

// Intent is the action the enemy will take when its countdown runs out.
func (e *Enemy) Intent() Action {
	a := Action{ Kind: e.Pattern[e.step%len(e.Pattern)], Damage: e.Attack }
	switch a.Kind {
	case Heavy:
		a.Damage = e.Attack * 2
	case Quake:
		a.Damage = e.Attack / 2
	}
	return a
}

func (e *Enemy) advance() {
	e.step++
	e.Countdown = e.Interval
}
