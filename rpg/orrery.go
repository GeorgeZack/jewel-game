package rpg

import (
	"fmt"
	"slices"
	"strings"
)

// Residue is what good matches leave behind, one pool per mana color plus a colorless pool. It is
// spent on the Orrery. Index it with Fire..Earth, or Colorless.
const (
	Colorless = NUM_MANA
	NUM_RESIDUE = NUM_MANA + 1
)

// Placeholder tuning for the Orrery prototype. Node costs are in orrery.yaml.
const (
	RESIDUE_FOUR	= 2		// a run of four in a mana color, on the hero's turn
	RESIDUE_FIVE	= 4		// a run of five or more
	RESIDUE_CHAIN	= 1		// more per mana match for each cascade step after the first
	CONVERT_STEP	= 10	// coloured residue taken by one conversion
	CONVERT_PCT		= 30	// colorless residue given per 100 coloured
	RESPEC_GOLD		= 100	// gold to refund every node and go back to the seat
	KIT_LEVEL		= 4		// a class knows its own rites up to this level without the Orrery
	OLD_RUN_RESIDUE	= 6		// colorless residue per level for a run saved before the Orrery
)

// OrreryNode is one body on the Orrery: orrery.yaml. Buying it gives its stats, its rite, or both.
type OrreryNode struct {
	ID			string			`yaml:"id"`
	Name		string			`yaml:"name"`
	Color		string			`yaml:"color"`	// the region: fire, water, air or earth
	Ring		int				`yaml:"ring"`	// 1 is the innermost
	Angle		float64			`yaml:"angle"`	// degrees clockwise from the top
	Links		[]string		`yaml:"links"`	// joined both ways
	Cost		int				`yaml:"cost"`	// residue of the region's color
	Seat		string			`yaml:"seat"`	// the class that starts here, owning it for free
	Rite		string			`yaml:"rite"`	// spell ID
	RawStats	map[string]int	`yaml:"stats"`

	Stats		[NUM_STATS]int	`yaml:"-"`
	Mana		int				`yaml:"-"`	// Color as a mana index, -1 if unknown
	Near		[]string		`yaml:"-"`	// Links plus the nodes that link here
}

var Orrery []OrreryNode

func NodeByID(id string) *OrreryNode {
	for i := range Orrery {
		if Orrery[i].ID == id {
			return &Orrery[i]
		}
	}
	return nil
}

// SeatOf is the node a class starts on, or nil.
func SeatOf(class string) *OrreryNode {
	for i := range Orrery {
		if Orrery[i].Seat == class {
			return &Orrery[i]
		}
	}
	return nil
}

func manaByName(name string) int {
	for i, n := range ManaNames {
		if strings.EqualFold(n, name) {
			return i
		}
	}
	return -1
}

// linkOrrery fills in what orrery.yaml leaves implied, and reports mistakes in its own terms.
func linkOrrery(nodes []OrreryNode) (problems []string) {
	for i := range nodes {
		n := &nodes[i]
		var bad []string
		n.Stats, bad = statArray(n.RawStats)
		problems = append(problems, prefix("orrery node "+n.ID+" stats", bad)...)
		n.Mana = manaByName(n.Color)
		n.Near = nil
	}
	for i := range nodes {
		for _, id := range nodes[i].Links {
			for j := range nodes {
				if nodes[j].ID == id && i != j {
					nodes[i].Near = append(nodes[i].Near, id)
					nodes[j].Near = append(nodes[j].Near, nodes[i].ID)
				}
			}
		}
	}
	for i := range nodes {
		slices.Sort(nodes[i].Near)
		nodes[i].Near = slices.Compact(nodes[i].Near)
	}
	return problems
}

// inKit reports whether a class knows a rite without buying it.
func inKit(class string, s *Spell) bool {
	return s.Class == class && s.Level <= KIT_LEVEL
}

// reachable lists the nodes that can be reached by links from a node.
func reachable(from string) map[string]bool {
	seen := map[string]bool{ from: true }
	todo := []string{ from }
	for len(todo) > 0 {
		n := NodeByID(todo[0])
		todo = todo[1:]
		for _, id := range n.Near {
			if !seen[id] {
				seen[id] = true
				todo = append(todo, id)
			}
		}
	}
	return seen
}

func checkOrrery() []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	if len(Orrery) == 0 {
		return nil
	}
	seen := map[string]bool{}
	for _, n := range Orrery {
		where := "orrery node " + n.ID
		if n.ID == "" {
			bad("orrery.yaml: a node has no id")
		} else if seen[n.ID] {
			bad("%s: defined twice", where)
		}
		seen[n.ID] = true
		if n.Mana < 0 {
			bad("%s: unknown color %q (fire, water, air or earth)", where, n.Color)
		}
		if n.Ring < 1 {
			bad("%s: ring %d should be 1 or more", where, n.Ring)
		}
		for _, id := range n.Links {
			if NodeByID(id) == nil || id == n.ID {
				bad("%s: bad link %q", where, id)
			}
		}
		if n.Seat != "" {
			if !slices.ContainsFunc(Classes, func(c Class) bool { return c.Name == n.Seat }) {
				bad("%s: seat for unknown class %q", where, n.Seat)
			}
		} else if n.Cost <= 0 {
			bad("%s: cost %d should be more than 0", where, n.Cost)
		}
		if n.Rite != "" {
			s := SpellByID(n.Rite)
			if s == nil || s.Class == "" {
				bad("%s: unknown rite %q", where, n.Rite)
			} else if inKit(s.Class, s) {
				bad("%s: rite %q is already in the %s kit", where, n.Rite, s.Class)
			}
		}
	}
	for _, c := range Classes {
		seat := SeatOf(c.Name)
		if seat == nil {
			bad("orrery.yaml: class %s has no seat", c.Name)
			continue
		}
		got := reachable(seat.ID)
		for _, n := range Orrery {
			if !got[n.ID] {
				bad("orrery node %s: can't be reached from the %s seat", n.ID, c.Name)
			}
		}
	}
	return p
}

// takeSeat starts a hero with no nodes on their class's seat.
func (h *Hero) takeSeat() {
	if h.Nodes != nil {
		return
	}
	h.Nodes = []string{}
	if seat := SeatOf(h.Class); seat != nil {
		h.Nodes = append(h.Nodes, seat.ID)
	}
}

func (h *Hero) Owns(node string) bool { return slices.Contains(h.Nodes, node) }

// Reaches reports whether a node is next to one the hero owns.
func (h *Hero) Reaches(n *OrreryNode) bool {
	return slices.ContainsFunc(n.Near, h.Owns)
}

// CanAfford: the region's residue pays first and colorless makes up the rest.
func (h *Hero) CanAfford(n *OrreryNode) bool {
	return n.Mana >= 0 && h.Residue[n.Mana]+h.Residue[Colorless] >= n.Cost
}

func (h *Hero) CanBuyNode(n *OrreryNode) bool {
	return n != nil && !h.Owns(n.ID) && h.Reaches(n) && h.CanAfford(n)
}

// BuyNode pays for a node next to one owned, and learns its rite.
func (h *Hero) BuyNode(id string) bool {
	n := NodeByID(id)
	if !h.CanBuyNode(n) {
		return false
	}
	paid := min(n.Cost, h.Residue[n.Mana])
	h.Residue[n.Mana] -= paid
	h.Residue[Colorless] -= n.Cost - paid
	h.Spent[n.Mana] += paid
	h.Spent[Colorless] += n.Cost - paid
	h.Nodes = append(h.Nodes, id)
	h.learnSpells()
	return true
}

// Convert turns CONVERT_STEP coloured residue into colorless, at a loss.
func (h *Hero) Convert(color int) bool {
	if color < 0 || color >= NUM_MANA || h.Residue[color] < CONVERT_STEP {
		return false
	}
	h.Residue[color] -= CONVERT_STEP
	h.Residue[Colorless] += CONVERT_STEP * CONVERT_PCT / 100
	return true
}

// Respec refunds the residue every bought node took, into the pools that paid it, for gold, and
// puts the hero back on the seat. Rites that go with the nodes are unequipped.
func (h *Hero) Respec() bool {
	if h.Gold < RESPEC_GOLD || len(h.Nodes) <= 1 {
		return false
	}
	h.Gold -= RESPEC_GOLD
	for i, n := range h.Spent {
		h.Residue[i] += n
	}
	h.Spent = [NUM_RESIDUE]int{}
	h.Nodes = nil
	h.takeSeat()
	h.Spells = slices.DeleteFunc(h.Spells, func(id string) bool {
		s := SpellByID(id)
		return s == nil || !h.Knows(s)
	})
	h.learnSpells()
	return true
}

// Knows reports whether a hero can cast a rite: it is in their class's kit or on a node they own,
// and they are of its level.
func (h *Hero) Knows(s *Spell) bool {
	if s.Class == "" || s.Level > h.Level {
		return false
	}
	if inKit(h.Class, s) {
		return true
	}
	return slices.ContainsFunc(h.Nodes, func(id string) bool {
		n := NodeByID(id)
		return n != nil && n.Rite == s.ID
	})
}
