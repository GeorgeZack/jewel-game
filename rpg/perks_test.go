package rpg

import (
	"fmt"
	"jewel-game/board"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// HERO_BEST_PCT is how often the simulated hero takes the swap heroMove likes best; the rest of
// the time it plays like the foe AI, so the economy isn't tuned for a perfect player.
const HERO_BEST_PCT = 50

// chooseMove is the simulator's move for the side to move: the foe AI for the foe, and for the
// hero sometimes heroMove.
func chooseMove(d *Duel) (a, c board.Point, ok bool) {
	if d.Turn == Player && d.rng.Intn(100) < HERO_BEST_PCT {
		return heroMove(d)
	}
	return d.ChooseMove()
}

// heroMove is the simulator's hero: it takes the swap whose matches are worth the most to it,
// counting what its perks reward. Ties go to a random one.
func heroMove(d *Duel) (a, c board.Point, ok bool) {
	me, b := d.Sides[d.Turn], d.Board
	best := -1
	var moves [][2]board.Point
	for y := range b.H() {
		for x := range b.W() {
			p := board.Point{X: x, Y: y}
			for _, q := range []board.Point{p.Add(board.Point{X: 1}), p.Add(board.Point{Y: 1})} {
				if !b.In(q) || !b.IsValidSwap(p, q) {
					continue
				}
				v := 0
				for _, m := range b.PreviewSwap(p, q) {
					v += matchWorth(me, m)
				}
				if v > best {
					best, moves = v, nil
				}
				if v == best {
					moves = append(moves, [2]board.Point{p, q})
				}
			}
		}
	}
	if len(moves) == 0 {
		return a, c, false
	}
	m := moves[d.rng.Intn(len(moves))]
	return m[0], m[1], true
}

// matchWorth is what one run is worth to the simulated hero.
func matchWorth(me *Combatant, m board.Match) int {
	v := m.Size
	switch {
	case m.Color == Skull:
		v = 3*m.Size + 6*m.Bombs
		for _, p := range me.perksOf(PerkForsake) {
			v += 2 * p.Amount
		}
		for _, p := range me.perksOf(PerkSkullPaint) {
			v += 2 * p.Amount
		}
	case m.Color < NUM_MANA:
		v = 0
		if me.Mana[m.Color] < me.MaxMana[m.Color] || me.HasPerk(PerkOverflow, m.Color) {
			v = 2 * m.Size * max(m.Mult, 1)
		}
		if me.HasPerk(PerkForsake, m.Color) {
			v = 0
		}
		for _, p := range me.Perks {
			if p.Color == m.Color && m.Size >= p.Size {
				switch {
				case p.Kind == PerkStrike:
					v += 3 * p.Amount
				case p.Kind == PerkMend && me.HP < me.MaxHP:
					v += 2 * p.Amount
				}
			}
		}
	}
	if m.Size >= 4 {
		if m.Size == 4 && m.Color < NUM_MANA && me.HasPerk(PerkWildFour, m.Color) {
			v += 8
		} else {
			v += 12
		}
	}
	return v + 4*m.Wilds
}

// perkHero is a Knight at level 10 who owns one perk node and nothing else of the Orrery.
func perkDuel(t *testing.T, node string) *Duel {
	t.Helper()
	h := NewHero("Knight")
	for h.Level < 10 {
		h.AddXP(XPToNext(h.Level))
	}
	if node != "" {
		if NodeByID(node) == nil || NodeByID(node).Perk == nil {
			t.Fatalf("no perk node %s", node)
		}
		h.Nodes = append(h.Nodes, node)
	}
	d := NewDuel(NewHeroCombatant(h), NewFoeCombatant(FoeByID("rat"), 1), 1)
	d.SetBoard(board.NewBoard(BoardSettings(nil), 1))
	d.Turn = Player
	return d
}

func perkNode(kind string) *OrreryNode {
	for i := range Orrery {
		if p := Orrery[i].Perk; p != nil && p.Kind == kind {
			return &Orrery[i]
		}
	}
	return nil
}

func run(color, size int) board.Match { return board.Match{ Color: color, Size: size, Mult: 1 } }

func gems(color, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = color
	}
	return out
}

func logged(d *Duel, name string) bool {
	return slices.ContainsFunc(d.Log, func(l LogLine) bool { return strings.Contains(l.Text(), name) })
}

func TestEveryPerkKindIsOnTheOrrery(t *testing.T) {
	for _, k := range PerkKinds {
		if perkNode(k) == nil {
			t.Errorf("no node has a %s perk", k)
		}
	}
}

func TestPerkStrikeAndMend(t *testing.T) {
	n := perkNode(PerkStrike)
	p := n.Perk
	d := perkDuel(t, n.ID)
	foe := d.Sides[Enemy]
	hp := foe.HP
	d.ApplyStep(step(gems(p.Color, p.Size-1), run(p.Color, p.Size-1)))
	other := (p.Color + 1) % NUM_MANA
	d.ApplyStep(step(gems(other, 5), run(other, 5)))
	if foe.HP != hp || logged(d, NodeLabel(n)) {
		t.Fatalf("%s hit on a short run or another color: %d -> %d", n.ID, hp, foe.HP)
	}
	d.ApplyStep(step(gems(p.Color, p.Size), run(p.Color, p.Size)))
	if foe.HP != hp-p.Amount || !logged(d, NodeLabel(n)) {
		t.Fatalf("%s: foe HP %d -> %d, want -%d, logged %v", n.ID, hp, foe.HP, p.Amount, logged(d, NodeLabel(n)))
	}
	// the foe's runs never use the hero's perks
	d.Turn = Enemy
	me := d.Sides[Player]
	mine := me.HP
	d.ApplyStep(step(gems(p.Color, 5), run(p.Color, 5)))
	if me.HP != mine {
		t.Fatalf("the foe's %s run used the hero's perk", ManaNames[p.Color])
	}

	m := perkNode(PerkMend)
	d = perkDuel(t, m.ID)
	me = d.Sides[Player]
	me.HP -= 10
	d.ApplyStep(step(gems(m.Perk.Color, m.Perk.Size), run(m.Perk.Color, m.Perk.Size)))
	if me.HP != me.MaxHP-10+m.Perk.Amount || !logged(d, NodeLabel(m)) {
		t.Fatalf("%s: HP %d of %d", m.ID, me.HP, me.MaxHP)
	}
}

func TestPerkWildFourTradesTheExtraTurn(t *testing.T) {
	n := perkNode(PerkWildFour)
	c := n.Perk.Color
	d := perkDuel(t, n.ID)
	wilds := func() int { return len(d.Board.CellsOfColor(-1)) }
	before := wilds()
	g := d.ApplyStep(step(gems(c, 4), run(c, 4)))
	if g.Extra || d.Extra || wilds() != before+1 || !logged(d, NodeLabel(n)) {
		t.Fatalf("%s four: extra %v, wilds %d -> %d", n.ID, g.Extra, before, wilds())
	}
	// a five, or a four in another color, still gives the extra turn and no Wild
	for _, m := range []board.Match{ run(c, 5), run((c+1)%NUM_MANA, 4), run(Skull, 4) } {
		d = perkDuel(t, n.ID)
		before = wilds()
		if g := d.ApplyStep(step(gems(m.Color, m.Size), m)); !g.Extra || wilds() != before {
			t.Fatalf("%s: a %d of %d gave extra %v, wilds %d -> %d", n.ID, m.Size, m.Color, g.Extra, before, wilds())
		}
	}
}

func TestPerkSkullPaint(t *testing.T) {
	n := perkNode(PerkSkullPaint)
	c := n.Perk.Color
	d := perkDuel(t, n.ID)
	b := d.Board
	// a matched skull at the middle of the board, its neighbours none of the perk's color
	mid := board.Point{X: 4, Y: 4}
	for _, q := range []board.Point{ {X: 3, Y: 4}, {X: 5, Y: 4}, {X: 4, Y: 3}, {X: 4, Y: 5} } {
		b.Paint(q, (c+1)%NUM_MANA, board.None)
	}
	before := len(b.CellsOfColor(c))
	plain := step(nil, run(Fire, 3))
	plain.Cleared = []board.Cleared{{ Pos: mid, Gem: board.Gem{ Color: (c + 2) % NUM_MANA }, Matched: true }}
	d.ApplyStep(plain)
	if len(b.CellsOfColor(c)) != before || logged(d, NodeLabel(n)) {
		t.Fatalf("%s painted without a skull match", n.ID)
	}
	res := step(nil, run(Skull, 3))
	res.Cleared = []board.Cleared{{ Pos: mid, Gem: board.Gem{ Color: Skull }, Matched: true }}
	d.ApplyStep(res)
	if got := len(b.CellsOfColor(c)) - before; got != n.Perk.Amount || !logged(d, NodeLabel(n)) {
		t.Fatalf("%s painted %d gems, want %d", n.ID, got, n.Perk.Amount)
	}
}

func TestPerkOverflowSpills(t *testing.T) {
	n := perkNode(PerkOverflow)
	p := n.Perk
	d := perkDuel(t, n.ID)
	me := d.Sides[Player]
	me.Mana[p.Color] = me.MaxMana[p.Color] - 4
	d.ApplyStep(step(gems(p.Color, 3), run(p.Color, 3)))
	if logged(d, NodeLabel(n)) {
		t.Fatalf("%s spilled with room left", n.ID)
	}
	to := me.Mana[p.ToMana]
	d.ApplyStep(step(gems(p.Color, 3), run(p.Color, 3)))
	if me.Mana[p.Color] != me.MaxMana[p.Color] || me.Mana[p.ToMana] <= to || !logged(d, NodeLabel(n)) {
		t.Fatalf("%s: %v of %v", n.ID, me.Mana, me.MaxMana)
	}
}

func TestPerkForsakeTradesManaForSkulls(t *testing.T) {
	n := perkNode(PerkForsake)
	c := n.Perk.Color
	plain, sworn := perkDuel(t, ""), perkDuel(t, n.ID)
	hits := [2]int{}
	for i, d := range []*Duel{ plain, sworn } {
		foe := d.Sides[Enemy]
		foe.HP, foe.MaxHP = 999, 999
		d.ApplyStep(step(gems(Skull, 3), run(Skull, 3)))
		hits[i] = 999 - foe.HP
		d.Sides[Player].Mana[c] = 0
		d.ApplyStep(step(gems(c, 3), run(c, 3)))
	}
	if hits[1] != hits[0]+n.Perk.Amount || !logged(sworn, NodeLabel(n)) {
		t.Fatalf("%s skull hit %d, plain %d", n.ID, hits[1], hits[0])
	}
	if sworn.Sides[Player].Mana[c] != 0 || plain.Sides[Player].Mana[c] == 0 {
		t.Fatalf("%s: %s mana sworn %d, plain %d", n.ID, ManaNames[c], sworn.Sides[Player].Mana[c], plain.Sides[Player].Mana[c])
	}
}

func TestPerkSeedPlantsAQuietBoard(t *testing.T) {
	for i := range Orrery {
		n := &Orrery[i]
		if n.Perk == nil || n.Perk.Kind != PerkSeed {
			continue
		}
		count := func(d *Duel) int {
			c := 0
			for y := range d.Board.H() {
				for x := range d.Board.W() {
					g := d.Board.At(board.Point{X: x, Y: y})
					if (n.Perk.Special == "wild" && g.Special == board.Wild) || (n.Perk.Special == "bomb" && g.Special == board.Bomb) {
						c++
					}
				}
			}
			return c
		}
		plain, seeded := perkDuel(t, ""), perkDuel(t, n.ID)
		if count(seeded) != count(plain)+n.Perk.Amount || seeded.Board.HasMatches() || !logged(seeded, NodeLabel(n)) {
			t.Fatalf("%s: %d planted over %d, matches %v", n.ID, count(seeded), count(plain), seeded.Board.HasMatches())
		}
	}
}

func TestFreeRespecWithTheFacility(t *testing.T) {
	var free *Facility
	for i := range Facilities {
		if slices.ContainsFunc(Facilities[i].Levels, func(l FacilityLevel) bool { return l.FreeRespec }) {
			free = &Facilities[i]
		}
	}
	if free == nil {
		t.Fatal("no facility makes respec free")
	}
	h := NewHero("Knight")
	h.Residue[Colorless] = 999
	h.BuyNode(SeatOf("Knight").Near[0])
	if h.RespecCost() != RESPEC_GOLD || h.Respec() {
		t.Fatalf("respec without gold or the facility: cost %d", h.RespecCost())
	}
	h.Flags[FOUNDING_FLAG] = true
	h.Done = append(h.Done, "barrow_lord")
	for _, c := range free.Requires {
		if q, ok := strings.CutPrefix(c, "quest:"); ok {
			h.Done = append(h.Done, q)
		}
	}
	h.Gold = 9999
	for h.FacilityLevel(free.ID) < len(free.Levels) {
		if !h.Build(free) {
			t.Fatalf("couldn't build %s", free.ID)
		}
	}
	h.Gold = 0
	if h.RespecCost() != 0 || !h.Respec() || len(h.Nodes) != 1 || h.Gold != 0 {
		t.Fatalf("free respec: cost %d, nodes %v, gold %d", h.RespecCost(), h.Nodes, h.Gold)
	}
}

// A run saved on the 24-node prototype grid loads, and what it paid for nodes, even ones that are
// gone, comes back to the pools that paid it.
func TestSmallOrreryRunIsRefunded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hero.json")
	old := `{"Version":2,"Run":{"Class":"Warrior","Level":7,"Gold":40,"Stats":[4,1,2,2,7,2,3],` +
		`"Nodes":["fire_seat","ember","hamstring_node","no_such_node"],"Residue":[3,0,0,0,1],"Spent":[40,0,0,0,12],` +
		`"Gear":{},"Spells":["cleave","hamstring"],"Location":"nowhere","Active":{}}}`
	os.WriteFile(path, []byte(old), 0o644)
	h, err := LoadRun(path)
	if err != nil || h == nil {
		t.Fatalf("old grid run: %v", err)
	}
	if !slices.Equal(h.Nodes, []string{ "fire_seat" }) || h.Residue != [NUM_RESIDUE]int{ 43, 0, 0, 0, 13 } || h.Spent != [NUM_RESIDUE]int{} {
		t.Fatalf("not refunded: nodes %v, residue %v, spent %v", h.Nodes, h.Residue, h.Spent)
	}
	if slices.Contains(h.Spells, "hamstring") || h.Gold != 40 {
		t.Fatalf("spells %v, gold %d", h.Spells, h.Gold)
	}
	if kept, _ := os.ReadFile(path + ".bak"); string(kept) != old {
		t.Fatalf("the old grid run wasn't backed up")
	}
	// a run saved now keeps what it bought
	h.Residue[Colorless] = 999
	near := SeatOf("Warrior").Near[0]
	h.BuyNode(near)
	SaveRun(path, h)
	if h, _ := LoadRun(path); !h.Owns(near) {
		t.Fatalf("a current run lost its nodes: %v", h.Nodes)
	}
}

func TestPerkValidation(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	files := memFiles{
		"s/story.yaml":     "start: a",
		"s/foes.yaml":      "- { id: rat, name: Rat }",
		"s/locations.yaml": "- { id: a, name: A, open: true }",
		"s/quests.yaml":    "[]",
		"s/orrery.yaml": "- { id: k, color: water, ring: 1, seat: Knight, links: [a, b, c, d, e] }\n" +
			"- { id: w, color: air, ring: 1, seat: Wizard, links: [k] }\n" +
			"- { id: r, color: earth, ring: 1, seat: Druid, links: [k] }\n" +
			"- { id: f, color: fire, ring: 1, seat: Warrior, links: [k] }\n" +
			"- { id: a, color: fire, ring: 2, cost: 5, perk: { kind: teleport } }\n" +
			"- { id: b, color: fire, ring: 2, cost: 5, perk: { kind: strike, size: 9, amount: 2 } }\n" +
			"- { id: c, color: water, ring: 2, cost: 5, perk: { kind: overflow, to: water } }\n" +
			"- { id: d, color: earth, ring: 2, cost: 5, perk: { kind: seed, special: lock, amount: 2 } }\n" +
			"- { id: e, color: air, ring: 2, cost: 5, perk: { kind: wild_four, amount: 7 } }",
	}
	problems, err := LoadStory(files, "s")
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(problems, "\n")
	for _, w := range []string{ "a: unknown perk kind \"teleport\"", "b: strike size 9", "c: overflow to \"water\"", "d: seed special \"lock\"", "e: wild_four amount 7" } {
		if !strings.Contains(all, w) {
			t.Errorf("no problem mentioning %q in:\n%s", w, all)
		}
	}
}

// fairHero is a hero of a class at a level, with the Orrery bought as the fairness tests do,
// leaving perks out, and the shop's best gear.
func fairHero(class string, level int) *Hero {
	h := NewHero(class)
	for h.Level < level {
		h.AddXP(XPToNext(h.Level))
	}
	h.Residue[Colorless] = (h.Level - 1) * FAIR_RESIDUE_PER_LEVEL
	b := greedy(class)
	b.noPerks = true
	buyNodes(h, b)
	bestShopGear(h)
	return h
}

// TestPerksStayFair gives each perk node alone to a level-appropriate hero of every class and
// fights a few bosses: every duel must finish, and the win rate must stay within PERK_BAND points
// of the same heroes without it (see the table with -v).
func TestPerksStayFair(t *testing.T) {
	const N = 6
	const PERK_BAND = 25	// percentage points a perk may move the win rate
	// the heroes are a little under the boss, so the fights are close enough to move
	bosses := []struct {
		id			string
		level, hero	int
	}{{"barrow_lord", 6, 4}, {"orc", 9, 6}, {"high_priest", 12, 9}, {"hollow_king", 16, 12}}
	rate := func(node string) (per []int, all int) {
		wins, total := 0, 0
		for _, f := range bosses {
			w := 0
			for _, class := range Classes {
				for seed := range N {
					h := fairHero(class.Name, f.hero)
					if node != "" {
						h.Nodes = append(h.Nodes, node)
					}
					d := NewDuel(NewHeroCombatant(h), NewFoeCombatant(FoeByID(f.id), f.level), int64(seed))
					d.SetBoard(board.NewBoard(BoardSettings(nil), int64(seed)))
					playOut(d)
					if d.Winner < 0 {
						t.Fatalf("%s vs %s with %q never finished", class.Name, f.id, node)
					}
					if d.Winner == Player {
						w++
					}
				}
			}
			n := N * len(Classes)
			per = append(per, 100*w/n)
			wins, total = wins+w, total+n
		}
		return per, 100 * wins / total
	}
	head := fmt.Sprintf("%-18s", "perk")
	for _, f := range bosses {
		head += fmt.Sprintf(" %12s", f.id)
	}
	t.Logf("%s %5s", head, "all")
	row := func(name string, per []int, all int) {
		line := fmt.Sprintf("%-18s", name)
		for _, p := range per {
			line += fmt.Sprintf(" %11d%%", p)
		}
		t.Logf("%s %4d%%", line, all)
	}
	basePer, base := rate("")
	row("(none)", basePer, base)
	for i := range Orrery {
		n := &Orrery[i]
		if n.Perk == nil {
			continue
		}
		per, all := rate(n.ID)
		row(n.ID, per, all)
		if all < base-PERK_BAND || all > base+PERK_BAND {
			t.Errorf("%s moves the win rate from %d%% to %d%%", n.ID, base, all)
		}
	}
}
