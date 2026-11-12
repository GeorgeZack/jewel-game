package rpg

import (
	"fmt"
	"jewel-game/board"
	"strings"
)

// MAX_CARD_SIZE is the widest and tallest board a card may ask for: ten columns is already
// small gems on a phone.
const MAX_CARD_SIZE = 10

// RuleCard changes the board a duel is played on, and what a cascade pays. It is cards.yaml.
// Every field left out keeps the standard duel, so an empty card is today's 8x8 board.
type RuleCard struct {
	ID				string	`yaml:"id"`
	Name			string	`yaml:"name"`
	Width			int		`yaml:"width"`
	Height			int		`yaml:"height"`
	Gravity			string	`yaml:"gravity"`	// down, left, up or right
	ChaosGravity	bool	`yaml:"chaos_gravity"`
	Overcharge		bool	`yaml:"overcharge"`

	// payout modifiers, in percent added to what a cascade step gives: 50 is half again, -50 half
	GoldPct			int		`yaml:"gold_pct"`
	XPPct			int		`yaml:"xp_pct"`
	ManaPct			int		`yaml:"mana_pct"`
	SkullPct		int		`yaml:"skull_pct"`
}

var RuleCards []RuleCard

// RuleCardByID is nil for "" or an unknown id; a nil card is the standard duel.
func RuleCardByID(id string) *RuleCard {
	for i := range RuleCards {
		if RuleCards[i].ID == id {
			return &RuleCards[i]
		}
	}
	return nil
}

func gravityByName(name string) (board.Direction, bool) {
	for d := board.Down; d <= board.Right; d++ {
		if strings.EqualFold(d.String(), name) {
			return d, true
		}
	}
	return board.Down, name == ""
}

// BoardSettings is the board a duel is played on: the standard duel, changed by the card if any.
func BoardSettings(c *RuleCard) board.Settings {
	s := board.Settings{
		Width:		8,
		Height:		8,
		NumColors:	NUM_GEMS,
		Duel:		true,
		WildColors:	NUM_MANA,
		BombColor:	Skull,
		BombRate:	BIG_SKULL_RATE,
	}
	if c == nil {
		return s
	}
	if c.Width > 0 {
		s.Width = c.Width
	}
	if c.Height > 0 {
		s.Height = c.Height
	}
	s.Gravity, _ = gravityByName(c.Gravity)
	s.ChaosGravity = c.ChaosGravity
	s.Overcharge = c.Overcharge
	return s
}

// pct scales n by a card's percentage modifier.
func pct(n, mod int) int {
	return n * (100 + mod) / 100
}

func checkCards() []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	seen := map[string]bool{}
	for _, c := range RuleCards {
		where := "card " + c.ID
		if c.ID == "" {
			bad("cards.yaml: a card has no id")
		} else if seen[c.ID] {
			bad("%s: defined twice", where)
		}
		seen[c.ID] = true
		for _, n := range []int{ c.Width, c.Height } {
			if n != 0 && (n < 4 || n > MAX_CARD_SIZE) {
				bad("%s: board size %d should be 4 to %d", where, n, MAX_CARD_SIZE)
			}
		}
		if _, ok := gravityByName(c.Gravity); !ok {
			bad("%s: unknown gravity %q (down, left, up or right)", where, c.Gravity)
		}
		for _, n := range []int{ c.GoldPct, c.XPPct, c.ManaPct, c.SkullPct } {
			if n < -100 {
				bad("%s: a modifier of %d%% would pay less than nothing", where, n)
			}
		}
	}
	return p
}
