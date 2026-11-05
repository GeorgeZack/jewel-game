package rpg

import (
	"fmt"
	"jewel-game/board"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func step(cleared []int, matches ...board.Match) *board.StepResult {
	res := &board.StepResult{Chain: 1, Matches: matches}
	for _, c := range cleared {
		res.Cleared = append(res.Cleared, board.Cleared{Gem: board.Gem{Color: c}})
	}
	return res
}

func testDuel() *Duel {
	d := NewDuel(NewHeroCombatant(NewHero("Knight")), NewFoeCombatant(FoeByID("rat"), 1), 1)
	d.Board = board.NewBoard(BoardSettings(), 1)
	d.Turn = Player
	return d
}

func TestStatBonusUsesTheCurrentValue(t *testing.T) {
	if got := StatBonus(Fire, 8); got != "+8 max Fire mana, +1 per match" {
		t.Fatalf("fire: %s", got)
	}
	if got := StatBonus(Battle, 9); got != "+2 damage per skull match" {
		t.Fatalf("battle: %s", got)
	}
	if got := StatBonus(Cunning, 4); got != "+8% gold and XP" {
		t.Fatalf("cunning: %s", got)
	}
	if got := StatBonus(Morale, 10); got != "+30 max HP, blocks 1" {
		t.Fatalf("morale: %s", got)
	}
}

func TestManaGoesToTheSideToMove(t *testing.T) {
	d := testDuel()
	d.Sides[Player].Stats[Fire] = 0
	d.ApplyStep(step([]int{Fire, Fire, Fire}, board.Match{Color: Fire, Size: 3, Mult: 1}))
	if d.Sides[Player].Mana[Fire] != 3 {
		t.Fatalf("player mana %v", d.Sides[Player].Mana)
	}
	d.Turn = Enemy
	d.Sides[Enemy].Stats[Water] = 0
	d.ApplyStep(step([]int{Water, Water, Water}, board.Match{Color: Water, Size: 3, Mult: 1}))
	if d.Sides[Enemy].Mana[Water] != 3 || d.Sides[Player].Mana[Water] != 0 {
		t.Fatalf("enemy turn should credit the enemy")
	}
}

func TestWildMultipliesAndManaCaps(t *testing.T) {
	d := testDuel()
	me := d.Sides[Player]
	me.Stats[Fire] = 0
	// two reds and a x3 Wild: 3 gems x3
	d.ApplyStep(step([]int{Fire, Fire, -1}, board.Match{Color: Fire, Size: 3, Mult: 3, Wilds: 1}))
	if me.Mana[Fire] != 9 {
		t.Fatalf("got %d", me.Mana[Fire])
	}
	me.Mana[Fire] = me.MaxMana[Fire] - 1
	if g := d.ApplyStep(step([]int{Fire, Fire, Fire}, board.Match{Color: Fire, Size: 3, Mult: 1})); g.Mana[Fire] != 1 {
		t.Fatalf("should cap, got %+v", g)
	}
}

func TestExtraTurnOnFour(t *testing.T) {
	d := testDuel()
	d.ApplyStep(step([]int{Gold, Gold, Gold, Gold}, board.Match{Color: Gold, Size: 4, Mult: 1}))
	d.EndTurn()
	if d.Turn != Player {
		t.Fatalf("four in a row keeps the turn")
	}
	d.EndTurn()
	if d.Turn != Enemy {
		t.Fatalf("the extra turn is only one turn")
	}
}

func TestSkullsHurtAndCanFinish(t *testing.T) {
	d := testDuel()
	foe := d.Sides[Enemy]
	hp := foe.HP
	g := d.ApplyStep(step([]int{Skull, Skull, Skull}, board.Match{Color: Skull, Size: 3, Mult: 1}))
	if g.Damage <= 0 || foe.HP != hp-g.Damage {
		t.Fatalf("skulls should hurt: %+v", g)
	}
	foe.HP = 1
	d.ApplyStep(step([]int{Skull, Skull, Skull}, board.Match{Color: Skull, Size: 3, Mult: 1}))
	if d.Winner != Player {
		t.Fatalf("expected a win")
	}
}

func TestCastPaysAndCoolsDown(t *testing.T) {
	d := testDuel()
	me := d.Sides[Player]
	s := SpellByID("smite")
	if d.Cast(s) {
		t.Fatalf("no mana yet")
	}
	me.Mana[Earth] = 6
	hp := d.Sides[Enemy].HP
	if !d.Cast(s) {
		t.Fatalf("should cast")
	}
	if me.Mana[Earth] != 0 || d.Sides[Enemy].HP >= hp || me.CanCast(s) {
		t.Fatalf("cast did not apply: %+v", me)
	}
}

func TestStunSkipsTurnAndPoisonTicks(t *testing.T) {
	d := testDuel()
	foe := d.Sides[Enemy]
	foe.AddEffect(Effect{Stun, 1, 0})
	foe.AddEffect(Effect{Poison, 2, 3})
	hp := foe.HP
	d.EndTurn()
	if d.Turn != Player || foe.HP != hp-3 {
		t.Fatalf("stun should pass back after a poison tick: turn %d hp %d", d.Turn, foe.HP)
	}
	d.EndTurn()
	if d.Turn != Enemy || foe.HP != hp-6 || len(foe.Effects) != 0 {
		t.Fatalf("second turn: %d %d %+v", d.Turn, foe.HP, foe.Effects)
	}
}

func TestSpellsAllCastOnARealBoard(t *testing.T) {
	for i := range Spells {
		s := &Spells[i]
		d := testDuel()
		me := d.Sides[Player]
		me.Spells = []*Spell{s}
		me.Mana = [NUM_MANA]int{50, 50, 50, 50}
		if !d.Cast(s) {
			t.Fatalf("%s did not cast", s.ID)
		}
		for d.Board.HasMatches() {
			if d.Board.Step() == nil {
				break
			}
			d.Board.Collapse()
		}
	}
}

type diskFiles struct{}

func (diskFiles) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (diskFiles) ReadDir(name string) ([]string, error) {
	entries, err := os.ReadDir(name)
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out, err
}

// Tests run against the real story in assets/story.
func TestMain(m *testing.M) {
	problems, err := LoadStory(diskFiles{}, "../assets/story")
	if err != nil {
		fmt.Println("loading story:", err)
		os.Exit(1)
	}
	storyProblems = problems
	os.Exit(m.Run())
}

var storyProblems []string

func TestStoryIsValid(t *testing.T) {
	for _, p := range storyProblems {
		t.Error(p)
	}
}

// memFiles is a story written inline, for testing the loader.
type memFiles map[string]string

func (m memFiles) ReadFile(name string) ([]byte, error) {
	if s, ok := m[name]; ok {
		return []byte(s), nil
	}
	return nil, os.ErrNotExist
}

func (m memFiles) ReadDir(dir string) ([]string, error) {
	var out []string
	for name := range m {
		if d, f := path.Split(name); path.Clean(d) == dir {
			out = append(out, f)
		}
	}
	return out, nil
}

func TestLoaderReportsMistakes(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	problems, err := LoadStory(memFiles{
		"s/story.yaml":     "start: nowhere\nopening: missing_scene",
		"s/foes.yaml":      "- { id: rat, name: Rat, stats: { fire: 1, luck: 2 }, spells: [bite, zap] }",
		"s/locations.yaml": "- { id: a, name: A, open: true, roads: [b], foes: [rat] }\n- { id: b, name: B, roads: [], foes: [bat] }",
		"s/quests.yaml":    "- { id: q, title: Q, giver: a, at: b, foe: rat, requires: [quest:nope, \"!flag:x\", level:two] }",
		"s/scenes/x.yaml":  "talk:\n  - who: stranger\n    text: hi\n  - goto: elsewhere\n  - choices: [ { text: ok, do: [ { accept_quest: q }, { item: no_such } ] } ]",
	}, "s")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"start location", "missing_scene", "luck", "zap", "one-way", "bat",
		"quest:nope", "level:two", "stranger", "elsewhere", "no_such",
	}
	all := strings.Join(problems, "\n")
	for _, w := range want {
		if !strings.Contains(all, w) {
			t.Errorf("no problem mentioning %q in:\n%s", w, all)
		}
	}
	if strings.Contains(all, "!flag:x") {
		t.Errorf("valid condition reported: %s", all)
	}
}

func TestConditions(t *testing.T) {
	h := NewHero("Druid")
	h.Done = []string{"goblins"}
	h.Flags["met"] = true
	h.Level = 3
	for cond, want := range map[string]bool{
		"quest:goblins": true, "quest:toll": false, "!quest:toll": true,
		"flag:met": true, "!flag:met": false, "flag:other": false,
		"level:3": true, "level:4": false, "class:druid": true,
		"at:" + Story.Start: true, "active:toll": false,
	} {
		if got := h.Meets([]string{cond}); got != want {
			t.Errorf("%s = %v, want %v", cond, got, want)
		}
	}
}

// The whole story must be reachable: every main quest's giver and target open in time.
func TestStoryCanBeCompleted(t *testing.T) {
	h := NewHero("Wizard")
	for _, q := range Quests {
		if !q.Main {
			continue
		}
		offered := false
		if h.Route(q.Giver) != nil || h.Location == q.Giver {
			for _, o := range h.OfferedQuests(q.Giver) {
				offered = offered || o.ID == q.ID
			}
		}
		if !offered {
			t.Fatalf("%s isn't offered", q.ID)
		}
		h.Location = q.Giver
		h.Accept(&q)
		if q.At != h.Location && h.Route(q.At) == nil {
			t.Fatalf("%s: can't reach %s", q.ID, q.At)
		}
		h.Location = q.At
		for range q.Count {
			h.RecordKill(q.Foe, q.At)
		}
	}
	if !h.Finished {
		t.Fatalf("story not finished")
	}
}

func TestLevelUpAndStats(t *testing.T) {
	h := NewHero("Warrior")
	if n := h.AddXP(XPToNext(1)); n != 1 || h.Level != 2 || h.Points != POINTS_PER_LEVEL {
		t.Fatalf("level up: %d %+v", n, h)
	}
	before := h.Stats[Battle]
	if !h.RaiseStat(Battle) || h.Stats[Battle] != before+1 || h.Points != POINTS_PER_LEVEL-1 {
		t.Fatalf("raise: %+v", h)
	}
	if len(h.Spells) != 2 {
		t.Fatalf("level 2 should auto-equip the new spell: %v", h.Spells)
	}
}

func TestShopAndMountsNeedStables(t *testing.T) {
	h := NewHero("Knight")
	h.Gold, h.Level = 5000, 20
	pony := ItemByID("pony")
	if h.Buy(pony) {
		t.Fatalf("no stables yet")
	}
	h.Build(BuildingByID("stables"))
	if !h.Buy(pony) || h.Gear[SlotMount] != "pony" {
		t.Fatalf("should buy")
	}
	gold := h.Gold
	h.Buy(pony)
	if h.Gold != gold {
		t.Fatalf("re-equipping is free")
	}
	if NewHeroCombatant(h).Stats[Cunning] != h.Stats[Cunning]+5 {
		t.Fatalf("mount bonus missing")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "hero.json")
	if h, err := LoadHero(path); h != nil || err != nil {
		t.Fatalf("no save yet: %v %v", h, err)
	}
	h := NewHero("Druid")
	h.Gold = 123
	h.Active["goblins"] = 1
	if err := h.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadHero(path)
	if err != nil || got.Gold != 123 || got.Active["goblins"] != 1 || got.Class != "Druid" {
		t.Fatalf("got %+v %v", got, err)
	}
}

// playOut runs a duel the way the view does, with the AI on both sides.
func playOut(d *Duel) int {
	turns := 0
	for !d.Over() && turns < 2000 {
		turns++
		if s := d.ChooseSpell(); s != nil {
			d.Cast(s)
		} else if a, c, ok := d.ChooseMove(); ok {
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
			d.ManaDrain()
			for d.Board.HasMatches() {
				d.Board.Step()
				d.Board.Collapse()
			}
		}
		d.EndTurn()
	}
	return turns
}

// bestShopGear equips the priciest shop item the level allows in each slot, as a player would.
func bestShopGear(h *Hero) {
	for i := range Items {
		it := &Items[i]
		if it.QuestOnly || it.Slot == SlotMount || it.Level > h.Level {
			continue
		}
		if cur := ItemByID(h.Gear[it.Slot]); cur == nil || it.Price > cur.Price {
			h.Gear[it.Slot] = it.ID
		}
	}
}

// Balance guard: a geared hero played by the AI should usually beat ordinary foes, have a real
// fight with story bosses, and fights shouldn't drag. Turns count both sides.
func TestDuelsFinishAndAreFair(t *testing.T) {
	const N = 6
	foes := []struct {
		id    string
		level int
	}{{"goblin", 1}, {"barrow_lord", 5}, {"orc", 8}, {"high_priest", 12}, {"hollow_king", 16}}
	for _, class := range Classes {
		for _, f := range foes {
			wins, turns := 0, 0
			for seed := range N {
				h := NewHero(class.Name)
				for h.Level < f.level {
					h.AddXP(XPToNext(h.Level))
				}
				for stat := 0; h.Points > 0 && stat < 200; stat++ {
					h.RaiseStat(stat % NUM_STATS)
				}
				bestShopGear(h)
				d := NewDuel(NewHeroCombatant(h), NewFoeCombatant(FoeByID(f.id), f.level), int64(seed))
				d.Board = board.NewBoard(BoardSettings(), int64(seed))
				turns += playOut(d)
				if d.Winner == Player {
					wins++
				} else if d.Winner < 0 {
					t.Fatalf("%s vs %s never finished", class.Name, f.id)
				}
			}
			t.Logf("%-8s vs %-12s L%-2d: %2d/%d wins, %d turns avg", class.Name, f.id, f.level, wins, N, turns/N)
			if turns/N > 160 {
				t.Errorf("%s vs %s drags: %d turns", class.Name, f.id, turns/N)
			}
		}
	}
}

// moveKinds lists combo swaps and ordinary swaps on b. A combo is a run of 4+, a wild, or a +5 skull.
func moveKinds(b *board.Board) (combo, other map[[2]board.Point]bool) {
	combo, other = map[[2]board.Point]bool{}, map[[2]board.Point]bool{}
	for y := range b.H() {
		for x := range b.W() {
			p := board.Point{X: x, Y: y}
			for _, q := range []board.Point{{X: x + 1, Y: y}, {X: x, Y: y + 1}} {
				if !b.In(q) || !b.IsValidSwap(p, q) {
					continue
				}
				kind := other
				for _, m := range b.PreviewSwap(p, q) {
					if m.Size >= 4 || m.Wilds > 0 || (m.Color == Skull && m.Bombs > 0) {
						kind = combo
						break
					}
				}
				kind[[2]board.Point{p, q}] = true
			}
		}
	}
	return combo, other
}

func TestWeakFoesLeaveCombos(t *testing.T) {
	var b *board.Board
	var combos map[[2]board.Point]bool
	for seed := int64(1); seed < 400; seed++ {
		candidate := board.NewBoard(BoardSettings(), seed)
		combo, other := moveKinds(candidate)
		if len(combo) > 0 && len(other) > 0 {
			b, combos = candidate, combo
			break
		}
	}
	if b == nil {
		t.Fatal("no board had both a combo and an ordinary match")
	}
	rate := func(level, n int) int {
		foe := NewFoeCombatant(FoeByID("rat"), level)
		d := NewDuel(NewHeroCombatant(NewHero("Knight")), foe, 1)
		d.Board = b
		d.Turn = Enemy
		hits := 0
		for range n {
			a, c, ok := d.ChooseMove()
			if !ok {
				t.Fatal("no move")
			}
			if combos[[2]board.Point{a, c}] {
				hits++
			}
		}
		return hits
	}
	const n = 200
	low, high := rate(1, n), rate(16, n)
	if low > n*2/5 {
		t.Errorf("a level 1 foe took the combo %d/%d times", low, n)
	}
	if high < n*11/20 {
		t.Errorf("a level 16 foe took the combo only %d/%d times", high, n)
	}
	if high < low {
		t.Errorf("higher level noticed fewer combos: L1 %d, L16 %d", low, high)
	}
}
