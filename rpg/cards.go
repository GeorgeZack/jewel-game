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
	Desc			string	`yaml:"desc"`	// the rule in one sentence, shown when the duel starts
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

// Placeholder board sizes: the standard duel's, and the larger one Aether arrives with so that
// matches stay about as common with eight colors as with seven.
const (
	DUEL_SIZE			= 8
	AETHER_DUEL_SIZE	= 9
)

// BoardSettings is the board a duel is played on: the standard duel, changed by the card if any.
// Without Aether the gem never arrives; with it the board is AETHER_DUEL_SIZE unless the card says.
func BoardSettings(c *RuleCard, aether bool) board.Settings {
	s := board.Settings{
		Width:		DUEL_SIZE,
		Height:		DUEL_SIZE,
		NumColors:	NUM_GEMS,
		Missing:	1 << Aether,
		Duel:		true,
		WildColors:	NUM_MANA,
		BombColor:	Skull,
		BombRate:	BIG_SKULL_RATE,
	}
	if aether {
		s.Width, s.Height, s.Missing = AETHER_DUEL_SIZE, AETHER_DUEL_SIZE, 0
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
		if c.Name == "" || c.Desc == "" {
			bad("%s: needs a name and a desc, the rule in one sentence", where)
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
