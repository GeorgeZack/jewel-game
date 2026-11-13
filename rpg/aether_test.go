package rpg

import (
	"jewel-game/board"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// churn plays a board with the foe AI on both sides for moves turns, calling see after each.
func churn(d *Duel, moves int, see func()) {
	for range moves {
		if a, c, ok := d.ChooseMove(); ok {
			d.Board.Swap(a, c)
			d.Board.BeginMove(a, c)
		}
		for {
			res := d.Board.Step()
			if res == nil {
				break
			}
			d.ApplyStep(res)
			d.Board.Collapse()
		}
		if d.Board.Shuffle() {
			for d.Board.HasMatches() {
				d.Board.Step()
				d.Board.Collapse()
			}
		}
		see()
		d.Turn = 1 - d.Turn
	}
}

func aetherOnBoard(b *board.Board) int {
	return len(b.CellsOfColor(Aether))
}

func TestAetherOnlyWhenTheFlagIsSet(t *testing.T) {
	off := NewHero("Knight")
	on := NewHero("Knight")
	on.Flags[AETHER_FLAG] = true
	if off.AetherOn() || !on.AetherOn() || len(off.Mana()) != 4 || len(on.Mana()) != NUM_MANA {
		t.Fatalf("mana in play: off %v, on %v", off.Mana(), on.Mana())
	}
	if off.StatShown(Aether) || !on.StatShown(Aether) || !off.StatShown(Battle) {
		t.Fatalf("the Aether stat shows in the wrong run")
	}
	for _, h := range []*Hero{ off, on } {
		for seed := range int64(6) {
			d := NewDuel(NewHeroCombatant(h), NewFoeCombatant(FoeByID("hollow_king"), 1), seed)
			d.Sides[Enemy].HP, d.Sides[Player].HP = 9999, 9999	// nobody falls before Aether has had its chance
			d.SetBoard(board.NewBoard(BoardSettings(nil, h.AetherOn()), seed))
			if d.AetherOn() != h.AetherOn() || len(d.Mana()) != len(h.Mana()) {
				t.Fatalf("the duel's Aether is %v for a run with it %v", d.AetherOn(), h.AetherOn())
			}
			seen := aetherOnBoard(d.Board)
			churn(d, 40, func() { seen += aetherOnBoard(d.Board) })
			got := d.Sides[Player].Mana[Aether] + d.Sides[Enemy].Mana[Aether]
			switch {
			case !h.AetherOn() && (seen > 0 || got > 0):
				t.Fatalf("Aether reached a board without the flag: %d gems, %d mana", seen, got)
			case h.AetherOn() && (seen == 0 || got == 0):
				t.Fatalf("Aether never came with the flag: %d gems, %d mana", seen, got)
			}
		}
	}
	if s := BoardSettings(nil, true); s.Width != AETHER_DUEL_SIZE || s.Height != AETHER_DUEL_SIZE {
		t.Fatalf("Aether's board is %dx%d", s.Width, s.Height)
	}
	if s := BoardSettings(RuleCardByID("hollow_halls"), true); s.Width != 10 || s.Height != 8 {
		t.Fatalf("a card's size gave way to Aether's: %dx%d", s.Width, s.Height)
	}
}

func TestAetherMatchesFillItsPool(t *testing.T) {
	d := testDuel()
	me := d.Sides[Player]
	me.Stats[Aether] = 0
	g := d.ApplyStep(step(gems(Aether, 4), run(Aether, 4)))
	if g.Mana[Aether] != 4 || me.Mana[Aether] != 4 || g.Residue[Aether] != RESIDUE_FOUR || !g.Extra {
		t.Fatalf("an Aether four: %+v, mana %v", g, me.Mana)
	}
}

func TestSpellsPayTheFifthPool(t *testing.T) {
	d := testDuel()
	me := d.Sides[Player]
	s := SpellByID("void_bolt")
	me.Spells = []*Spell{ s }
	me.Mana = [NUM_MANA]int{ 50, 50, 50, 50, s.Cost[Aether] - 1 }
	if me.CanCast(s) || d.Cast(s) {
		t.Fatalf("cast %s an Aether short", s.ID)
	}
	me.Mana[Aether] = s.Cost[Aether] + 2
	if !d.Cast(s) || me.Mana[Aether] != 2 || me.Mana[Fire] != 50 {
		t.Fatalf("void bolt paid wrong: %v", me.Mana)
	}
	if c := cost(1, 2, 3, 4, 5); c != [NUM_MANA]int{ 1, 2, 3, 4, 5 } || cost(0, 0, 0, 6) != [NUM_MANA]int{ 0, 0, 0, 6, 0 } {
		t.Fatalf("cost order: %v", c)
	}

	// blood price pays what Aether is short from HP, but never the last of it
	d = perkDuel(t, "blood_price")
	me = d.Sides[Player]
	me.Spells = []*Spell{ s }
	p := NodeByID("blood_price").Perk
	me.Mana = [NUM_MANA]int{}
	me.Mana[Aether] = s.Cost[Aether] - 3
	hp := me.HP
	if !me.CanCast(s) || !d.Cast(s) || me.Mana[Aether] != 0 || me.HP != hp-3*p.Amount {
		t.Fatalf("blood price: mana %v, HP %d -> %d", me.Mana, hp, me.HP)
	}
	me.Cooldowns = map[string]int{}
	me.HP = 3*p.Amount
	me.Mana[Aether] = s.Cost[Aether] - 3
	if me.CanCast(s) {
		t.Fatalf("blood price would take the last HP")
	}
}

func TestAetherPerksCostSomething(t *testing.T) {
	d := perkDuel(t, "void_lance")
	me, foe := d.Sides[Player], d.Sides[Enemy]
	p := NodeByID("void_lance").Perk
	hp, foeHP := me.HP, foe.HP
	d.ApplyStep(step(gems(Aether, 3), run(Aether, 3)))
	if foe.HP != foeHP-p.Amount || me.HP != hp-p.Backlash {
		t.Fatalf("void lance: foe %d -> %d, hero %d -> %d", foeHP, foe.HP, hp, me.HP)
	}

	d = perkDuel(t, "hunger")
	me = d.Sides[Player]
	p = NodeByID("hunger").Perk
	me.Stats[Aether] = 0
	me.Mana[p.ToMana] = 10
	d.ApplyStep(step(gems(Aether, 3), run(Aether, 3)))
	if me.Mana[p.ToMana] != 10-p.Amount || me.Mana[Aether] != 3+p.Amount {
		t.Fatalf("hunger: %v", me.Mana)
	}
}

func TestAetherRegionLockedUntilTheFlag(t *testing.T) {
	var core []*OrreryNode
	for i := range Orrery {
		if n := &Orrery[i]; n.Mana == Aether {
			core = append(core, n)
			if n.Seat != "" {
				t.Fatalf("%s seats a class in Aether", n.ID)
			}
		}
	}
	if len(core) < 10 || len(core) > 12 {
		t.Fatalf("the Aether region has %d nodes", len(core))
	}
	for _, c := range Classes {
		seat := SeatOf(c.Name).ID
		locked, open := reachable(seat, false), reachable(seat, true)
		for i := range Orrery {
			n := &Orrery[i]
			if n.Mana == Aether && locked[n.ID] {
				t.Fatalf("%s reaches %s while Aether is locked", c.Name, n.ID)
			}
			if !open[n.ID] || (n.Mana != Aether && !locked[n.ID]) {
				t.Fatalf("%s can't reach %s", c.Name, n.ID)
			}
		}
	}
	// a hero next to the core can't buy into it until the flag is set
	h := NewHero("Knight")
	h.Residue[Colorless] = 9999
	gate := core[slices.IndexFunc(core, func(n *OrreryNode) bool {
		return slices.ContainsFunc(n.Near, func(id string) bool { return NodeByID(id).Mana != Aether })
	})]
	var outside string
	for _, id := range gate.Near {
		if NodeByID(id).Mana != Aether {
			outside = id
		}
	}
	for _, id := range pathTo(h.Nodes[0], outside) {
		if !h.BuyNode(id) {
			t.Fatalf("couldn't buy %s on the way in", id)
		}
	}
	if h.Reaches(gate) || h.BuyNode(gate.ID) {
		t.Fatalf("bought %s with Aether locked", gate.ID)
	}
	h.Flags[AETHER_FLAG] = true
	if !h.BuyNode(gate.ID) {
		t.Fatalf("couldn't buy %s with Aether open", gate.ID)
	}
}

// A run saved before Aether, with four mana colors in every array, loads with Aether off and at
// zero, and everything else where it was.
func TestFourManaRunLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hero.json")
	old := `{"Version":3,"Run":{"Class":"Warrior","Level":7,"Gold":40,"Stats":[4,1,2,2,7,2,3],` +
		`"Nodes":["fire_seat","ember"],"Residue":[3,1,2,4,9],"Spent":[12,0,0,0,0],` +
		`"Henchmen":[{"Rank":1,"Report":{"At":"barrow","Line":"x","Gold":5,"Residue":[1,2,3,4]}}],` +
		`"Gear":{},"Spells":["cleave"],"Location":"nowhere","Active":{}}}`
	os.WriteFile(path, []byte(old), 0o644)
	h, err := LoadRun(path)
	if err != nil || h == nil {
		t.Fatalf("four-mana run: %v", err)
	}
	if h.Stats != [NUM_STATS]int{ 4, 1, 2, 2, 0, 7, 2, 3 } || h.Residue != [NUM_RESIDUE]int{ 3, 1, 2, 4, 0, 9 } ||
		h.Spent != [NUM_RESIDUE]int{ 12, 0, 0, 0, 0, 0 } {
		t.Fatalf("arrays: stats %v, residue %v, spent %v", h.Stats, h.Residue, h.Spent)
	}
	if r := h.Henchmen[0].Report; r == nil || r.Residue != [NUM_MANA]int{ 1, 2, 3, 4, 0 } {
		t.Fatalf("report: %+v", r)
	}
	if h.AetherOn() || !slices.Equal(h.Nodes, []string{ "fire_seat", "ember" }) {
		t.Fatalf("aether %v, nodes %v", h.AetherOn(), h.Nodes)
	}
	if kept, _ := os.ReadFile(path + ".bak"); string(kept) != old {
		t.Fatalf("the four-mana run wasn't backed up")
	}
	SaveRun(path, h)
	if again, _ := LoadRun(path); again.Stats != h.Stats || again.Residue != h.Residue {
		t.Fatalf("a current run changed on loading: %v %v", again.Stats, again.Residue)
	}
}

func TestBoardSizesReachTheBoardAndResolve(t *testing.T) {
	for _, size := range [][2]int{ {8, 8}, {9, 9}, {10, 10}, {10, 6}, {6, 10}, {10, 8} } {
		c := &RuleCard{ ID: "size", Width: size[0], Height: size[1] }
		d := testDuel()
		d.SetBoard(board.NewBoard(BoardSettings(c, true), 3))
		if d.Board.W() != size[0] || d.Board.H() != size[1] {
			t.Fatalf("asked %v, got %dx%d", size, d.Board.W(), d.Board.H())
		}
		churn(d, 30, func() {
			if d.Board.HasMatches() || !d.Board.HasMove() {
				t.Fatalf("%v: the board didn't settle to a playable state", size)
			}
		})
	}
}

// The foe AI ranks swaps by PreviewSwap; on every card the first step of the swap it picks must
// make the matches the preview promised, gravity and overcharge included.
func TestFoeAIPreviewHoldsOnEveryCard(t *testing.T) {
	for i := range RuleCards {
		c := &RuleCards[i]
		for _, aether := range []bool{ false, true } {
			for seed := range int64(8) {
				d := testDuel()
				d.Turn = Enemy
				d.SetBoard(board.NewBoard(BoardSettings(c, aether), seed))
				a, q, ok := d.ChooseMove()
				if !ok {
					t.Fatalf("%s: no move", c.ID)
				}
				want := d.Board.PreviewSwap(a, q)
				d.Board.Swap(a, q)
				d.Board.BeginMove(a, q)
				res := d.Board.Step()
				if res == nil || len(want) == 0 {
					t.Fatalf("%s: the chosen swap made nothing", c.ID)
				}
				for _, m := range want {
					if !slices.ContainsFunc(res.Matches, func(r board.Match) bool { return r.Color == m.Color && r.Size >= m.Size }) {
						t.Fatalf("%s: previewed %+v, the step made %+v", c.ID, m, res.Matches)
					}
				}
			}
		}
	}
}

func TestBossCards(t *testing.T) {
	bosses := 0
	for _, f := range Foes {
		if f.Card == "" {
			continue
		}
		c := RuleCardByID(f.Card)
		if c == nil || c.Desc == "" {
			t.Fatalf("%s fights on %q, which has no rule to show", f.ID, f.Card)
		}
		bosses++
	}
	if bosses < 4 {
		t.Fatalf("only %d foes use a card", bosses)
	}
	if problems := checkCards(); len(problems) > 0 {
		t.Fatalf("cards: %v", problems)
	}
}

// TestAetherBoardKeepsMatches measures what an eighth color does to a board: the swaps on offer
// and how often it runs out of moves, on the standard board, on it with Aether, and on Aether's
// larger board (see the numbers with -v). The larger board must keep about as many moves as the
// standard one.
func TestAetherBoardKeepsMatches(t *testing.T) {
	t.Parallel()
	type cfg struct {
		name	string
		size	int
		aether	bool
	}
	// swaps on offer after each move, and the share of moves that left none (a reshuffle)
	measure := func(c cfg) (moves, dry float64) {
		const seeds, turns = 6, 60
		n, empty := 0, 0
		for seed := range int64(seeds) {
			s := BoardSettings(nil, c.aether)
			s.Width, s.Height = c.size, c.size
			d := testDuel()
			d.SetBoard(board.NewBoard(s, seed))
			for range turns {
				if a, q, ok := d.ChooseMove(); ok {
					d.Board.Swap(a, q)
					d.Board.BeginMove(a, q)
				}
				for d.Board.Step() != nil {
					d.Board.Collapse()
				}
				if d.Board.Shuffle() {
					empty++
					for d.Board.HasMatches() {
						d.Board.Step()
						d.Board.Collapse()
					}
				}
				n += len(validSwaps(d.Board))
			}
		}
		return float64(n) / (seeds * turns), 100 * float64(empty) / (seeds * turns)
	}
	var base float64
	for _, c := range []cfg{ {"8x8, seven colors", 8, false}, {"8x8, with Aether", 8, true}, {"9x9, with Aether", AETHER_DUEL_SIZE, true}, {"10x10, with Aether", 10, true} } {
		moves, sh := measure(c)
		t.Logf("%-20s %5.1f swaps on offer a turn, %4.1f%% of turns run out of moves", c.name, moves, sh)
		switch {
		case !c.aether:
			base = moves
		case c.size == AETHER_DUEL_SIZE && moves < base*0.75:
			t.Errorf("Aether's board offers %.1f swaps a turn against %.1f on the standard one", moves, base)
		}
	}
}

func validSwaps(b *board.Board) (out [][2]board.Point) {
	for y := range b.H() {
		for x := range b.W() {
			p := board.Point{X: x, Y: y}
			for _, q := range []board.Point{ {X: x + 1, Y: y}, {X: x, Y: y + 1} } {
				if b.In(q) && b.IsValidSwap(p, q) {
					out = append(out, [2]board.Point{ p, q })
				}
			}
		}
	}
	return out
}
