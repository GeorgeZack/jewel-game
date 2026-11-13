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
	v, traded := m.Size, false
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
					v += 2 * p.Amount
				case p.Kind == PerkMend && me.HP < me.MaxHP:
					v += min(2*p.Amount, me.MaxHP-me.HP)
				case p.Kind == PerkHunger && me.Mana[m.Color] < me.MaxMana[m.Color]:
					v += p.Amount
				}
				v -= 2 * p.Backlash
				traded = traded || (p.Trade && (p.Kind == PerkStrike || p.Kind == PerkMend))
			}
		}
	}
	if m.Size >= 4 && !traded {
		if m.Size == 4 && m.Color < NUM_MANA && me.HasPerk(PerkWildFour, m.Color) {
			v += 10
		} else {
			v += 20
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
	d.SetBoard(board.NewBoard(BoardSettings(nil, false), 1))
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
	g := d.ApplyStep(step(gems(m.Perk.Color, m.Perk.Size), run(m.Perk.Color, m.Perk.Size)))
	if me.HP != me.MaxHP-10+m.Perk.Amount || !logged(d, NodeLabel(m)) {
		t.Fatalf("%s: HP %d of %d", m.ID, me.HP, me.MaxHP)
	}
	if m.Perk.Trade == g.Extra {
		t.Fatalf("%s (trade %v) gave extra turn %v", m.ID, m.Perk.Trade, g.Extra)
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
	if plain := perkDuel(t, "").Sides[Player]; me.MaxMana[p.Color] != plain.MaxMana[p.Color]-p.Amount {
		t.Fatalf("%s: max %s %d, without it %d", n.ID, ManaNames[p.Color], me.MaxMana[p.Color], plain.MaxMana[p.Color])
	}
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
	if !slices.Equal(h.Nodes, []string{ "fire_seat" }) || h.Residue != [NUM_RESIDUE]int{ 43, 0, 0, 0, 0, 13 } || h.Spent != [NUM_RESIDUE]int{} {
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

// fairBoss is a fairness matchup: a boss at its level against heroes of a lower level, picked so
// a hero without perks wins about half the time.
type fairBoss struct {
	id			string
	level, hero	int
}

// fullSim reports whether JEWEL_FULL_SIM asks for the slower, fuller simulator runs.
func fullSim() bool { return os.Getenv("JEWEL_FULL_SIM") != "" }

// fairRate fights every class with the given perk nodes against each boss, seeds duels per class,
// all at once, on the card's board with or without Aether. It returns the win rate per boss and
// overall, in percent, and the average turns.
func fairRate(t *testing.T, bosses []fairBoss, seeds int, nodes []string, card *RuleCard, aether bool) (per []int, all, turns int) {
	type result struct{ boss, won, turns int; done bool }
	out := make(chan result)
	jobs := 0
	for bi, f := range bosses {
		for _, class := range Classes {
			for seed := range seeds {
				jobs++
				go func() {
					h := fairHero(class.Name, f.hero)
					h.Nodes = append(h.Nodes, nodes...)
					d := NewDuel(NewHeroCombatant(h), NewFoeCombatant(FoeByID(f.id), f.level), int64(seed))
					d.SetBoard(board.NewBoard(BoardSettings(card, aether), int64(seed)))
					d.Card = card
					n := playOut(d)
					w := 0
					if d.Winner == Player {
						w = 1
					}
					out <- result{bi, w, n, d.Winner >= 0}
				}()
			}
		}
	}
	wins := make([]int, len(bosses))
	total, won := 0, 0
	for range jobs {
		r := <-out
		if !r.done {
			t.Errorf("a duel against %s with %v never finished", bosses[r.boss].id, nodes)
		}
		wins[r.boss] += r.won
		won += r.won
		turns += r.turns
	}
	n := seeds * len(Classes)
	for _, w := range wins {
		per = append(per, 100*w/n)
	}
	total = n * len(bosses)
	return per, 100 * won / total, turns / total
}

// fairPairs are perks that plausibly combine, checked together as well as alone.
var fairPairs = [][]string{ {"tidewell", "still_pool"}, {"kindling", "pyre_oath"}, {"thunderhead", "static"}, {"rootgrasp", "seedbed"} }

// TestPerksStayFair gives each perk node alone, and a few pairs, to heroes of every class a few
// levels under a boss, matched so they win about half the time without perks. Every duel must
// finish, no row may drag past MAX_TURNS, and the win rate must stay within band points of
// the heroes without perks (see the table with -v). JEWEL_FULL_SIM=1 runs more bosses and seeds.
func TestPerksStayFair(t *testing.T) {
	t.Parallel()	// it only reads the story, so it shares the CPU with the other simulator runs
	const MAX_TURNS = 160	// average turns, both sides counted, as TestDuelsFinishAndAreFair
	// the quick run is a smoke test with few duels, so its band is wider
	band := 35	// percentage points a perk, or a pair, may move the win rate
	bosses := []fairBoss{{"orc", 9, 3}, {"hollow_king", 16, 11}}
	seeds := 2
	if fullSim() {
		band = 20
		bosses = []fairBoss{{"barrow_lord", 6, 3}, {"orc", 9, 3}, {"high_priest", 12, 6}, {"hollow_king", 16, 11}}
		seeds = 12
	}
	head := fmt.Sprintf("%-24s", "perk")
	for _, f := range bosses {
		head += fmt.Sprintf(" %12s", fmt.Sprintf("%s %d/%d", f.id, f.hero, f.level))
	}
	t.Logf("%s %5s %5s", head, "all", "turns")
	row := func(name string, per []int, all, turns int) {
		line := fmt.Sprintf("%-24s", name)
		for _, p := range per {
			line += fmt.Sprintf(" %11d%%", p)
		}
		t.Logf("%s %4d%% %5d", line, all, turns)
	}
	basePer, base, baseTurns := fairRate(t, bosses, seeds, nil, nil, false)
	row("(none)", basePer, base, baseTurns)
	if base < 30 || base > 70 {
		t.Errorf("the baseline wins %d%%; the matchups should be near 50%% to separate perks", base)
	}
	// Aether's perks only work on a board with Aether, so they are measured against one
	aetherPer, aetherBase, aetherTurns := fairRate(t, bosses, seeds, nil, nil, true)
	row("(none, with Aether)", aetherPer, aetherBase, aetherTurns)
	var sets [][]string
	for i := range Orrery {
		if Orrery[i].Perk != nil {
			sets = append(sets, []string{ Orrery[i].ID })
		}
	}
	sets = append(sets, fairPairs...)
	for _, nodes := range sets {
		aether := slices.ContainsFunc(nodes, func(id string) bool { return NodeByID(id).Mana == Aether })
		base := base
		if aether {
			base = aetherBase
		}
		per, all, turns := fairRate(t, bosses, seeds, nodes, nil, aether)
		name := strings.Join(nodes, "+")
		row(name, per, all, turns)
		if all < base-band || all > base+band {
			t.Errorf("%s moves the win rate from %d%% to %d%%", name, base, all)
		}
		if turns > MAX_TURNS {
			t.Errorf("%s drags: %d turns", name, turns)
		}
	}
}

// TestCardsStayFair plays every rule card the way TestPerksStayFair plays a perk: heroes of
// every class against bosses they beat about half the time on the standard duel, with and
// without Aether. Every duel must finish, no row may drag past MAX_TURNS, and the win rate must
// stay within band points of the same matchups without a card (see the table with -v).
// JEWEL_FULL_SIM=1 runs more bosses and seeds.
func TestCardsStayFair(t *testing.T) {
	t.Parallel()	// it only reads the story, so it shares the CPU with the other simulator runs
	const MAX_TURNS = 160
	band := 35
	bosses := []fairBoss{{"orc", 9, 3}, {"hollow_king", 16, 11}}
	seeds := 2
	if fullSim() {
		band = 20
		bosses = []fairBoss{{"barrow_lord", 6, 3}, {"orc", 9, 3}, {"high_priest", 12, 6}, {"hollow_king", 16, 11}}
		seeds = 12
	}
	t.Logf("%-28s %5s %5s", "card", "all", "turns")
	for _, aether := range []bool{ false, true } {
		_, base, baseTurns := fairRate(t, bosses, seeds, nil, nil, aether)
		mode := ""
		if aether {
			mode = ", Aether"
		}
		t.Logf("%-28s %4d%% %5d", "(no card"+mode+")", base, baseTurns)
		for i := range RuleCards {
			c := &RuleCards[i]
			_, all, turns := fairRate(t, bosses, seeds, nil, c, aether)
			name := c.ID + mode
			t.Logf("%-28s %4d%% %5d", name, all, turns)
			if all < base-band || all > base+band {
				t.Errorf("%s moves the win rate from %d%% to %d%%", name, base, all)
			}
			if turns > MAX_TURNS {
				t.Errorf("%s drags: %d turns", name, turns)
			}
		}
	}
}

// A trade strike pays its damage instead of the extra turn; a three doesn't trigger it.
func TestPerkStrikeTrade(t *testing.T) {
	var n *OrreryNode
	for i := range Orrery {
		if p := Orrery[i].Perk; p != nil && p.Kind == PerkStrike && p.Trade {
			n = &Orrery[i]
		}
	}
	if n == nil {
		t.Skip("no trade strike on the Orrery")
	}
	c := n.Perk.Color
	d := perkDuel(t, n.ID)
	foe := d.Sides[Enemy]
	hp := foe.HP
	if g := d.ApplyStep(step(gems(c, 3), run(c, 3))); g.Extra || foe.HP != hp {
		t.Fatalf("%s fired on a three", n.ID)
	}
	if g := d.ApplyStep(step(gems(c, 4), run(c, 4))); g.Extra || foe.HP != hp-n.Perk.Amount {
		t.Fatalf("%s four: extra %v, foe HP %d -> %d", n.ID, g.Extra, hp, foe.HP)
	}
}
