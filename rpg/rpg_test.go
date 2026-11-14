package rpg

import (
	"fmt"
	"jewel-game/board"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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
	d.Board = board.NewBoard(BoardSettings(nil, false), 1)
	d.Turn = Player
	return d
}

func TestManaDrainHalvesBothSides(t *testing.T) {
	d := testDuel()
	d.Sides[Player].Mana = [NUM_MANA]int{8, 7, 6, 5}
	d.Sides[Enemy].Mana = [NUM_MANA]int{4, 3, 2, 1}
	d.ManaDrain()
	if d.Sides[Player].Mana != [NUM_MANA]int{4, 3, 3, 2} {
		t.Fatalf("player mana %v, want [4 3 3 2]", d.Sides[Player].Mana)
	}
	if d.Sides[Enemy].Mana != [NUM_MANA]int{2, 1, 1, 0} {
		t.Fatalf("enemy mana %v, want [2 1 1 0]", d.Sides[Enemy].Mana)
	}
}

func TestStatBonusUsesTheCurrentValue(t *testing.T) {
	if got := StatBonus(Fire, 8); got != "+8 max Fire, +1 per match" {
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
		me.Mana = [NUM_MANA]int{50, 50, 50, 50, 50}
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

// The whole story must be reachable: every main quest's giver and target open in time, on every
// branch, with the done scenes played (see finishMainQuests in society_test.go).
func TestStoryCanBeCompleted(t *testing.T) {
	for _, a := range eachAllegiance() {
		finishMainQuests(t, a)
	}
}

func TestLevelUp(t *testing.T) {
	h := NewHero("Warrior")
	hp := NewHeroCombatant(h).MaxHP
	if n := h.AddXP(XPToNext(1)); n != 1 || h.Level != 2 || NewHeroCombatant(h).MaxHP <= hp {
		t.Fatalf("level up: %d %+v", n, h)
	}
	if len(h.Spells) != 2 {
		t.Fatalf("level 2 should auto-equip the new spell: %v", h.Spells)
	}
}

func TestShopBuysAndReequipsFree(t *testing.T) {
	h := NewHero("Knight")
	h.Gold, h.Level = 5000, 20
	coin := ItemByID("lucky_coin")
	if !h.Buy(coin) || h.Gear[SlotTrinket] != "lucky_coin" {
		t.Fatalf("should buy")
	}
	gold := h.Gold
	h.Buy(coin)
	if h.Gold != gold {
		t.Fatalf("re-equipping is free")
	}
	if NewHeroCombatant(h).GoldPct != 15 {
		t.Fatalf("item bonus missing")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	path, ppath := filepath.Join(dir, "hero.json"), filepath.Join(dir, "profile.json")
	if h, err := LoadRun(path); h != nil || err != nil {
		t.Fatalf("no save yet: %v %v", h, err)
	}
	if p, err := LoadProfile(ppath); p.Returns != 0 || err != nil {
		t.Fatalf("no profile yet: %v %v", p, err)
	}
	h := NewHero("Druid")
	h.Gold = 123
	h.Active["goblins"] = 1
	if err := SaveRun(path, h); err != nil {
		t.Fatal(err)
	}
	if err := SaveProfile(ppath, Profile{ Returns: 2, Endings: []string{ "a" } }); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRun(path)
	if err != nil || got.Gold != 123 || got.Active["goblins"] != 1 || got.Class != "Druid" {
		t.Fatalf("got %+v %v", got, err)
	}
	if err := DeleteRun(path); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadRun(path); err != nil || got != nil {
		t.Fatalf("run not deleted: %+v %v", got, err)
	}
	if p, err := LoadProfile(ppath); err != nil || p.Returns != 2 {
		t.Fatalf("deleting the run should keep the profile: %+v %v", p, err)
	}
}

// A save from before versioning is a bare hero; it loads as the run.
func TestOldSaveMigrates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hero.json")
	old := `{"Class":"Wizard","Level":4,"XP":10,"Gold":321,"Points":5,"Stats":[9,5,4,1,1,3,2],` +
		`"Gear":null,"Owned":null,"Spells":["fireball"],"Buildings":null,"Location":"nowhere",` +
		`"Active":null,"Done":["goblins"],"Wins":3,"Losses":1,"Finished":false}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := LoadRun(path)
	if err != nil || h == nil || h.Gold != 321 || h.Level != 4 || h.Done[0] != "goblins" {
		t.Fatalf("old save: %+v %v", h, err)
	}
	if h.Location != Story.Start || h.Gear == nil || h.Active == nil {
		t.Fatalf("old save not fixed up: %+v", h)
	}
	// raised stats and unspent points from before the Orrery go back to the class's, the levels
	// pay colorless residue, and spells the hero no longer knows are unequipped
	if h.Stats != ClassByName("Wizard").Stats || !slices.Equal(h.Nodes, []string{ SeatOf("Wizard").ID }) || h.Residue[Colorless] != 3*OLD_RUN_RESIDUE {
		t.Fatalf("old stats not moved onto the Orrery: %+v", h)
	}
	if slices.Contains(h.Spells, "fireball") || !slices.Contains(h.Spells, "firebolt") {
		t.Fatalf("old spells not fixed up: %v", h.Spells)
	}
	// the run as it was before converting is kept
	if kept, _ := os.ReadFile(path + ".bak"); string(kept) != old {
		t.Fatalf("the pre-Orrery run wasn't backed up: %q", kept)
	}
	// a versioned run from before the Orrery is backed up too, and a converted one isn't again
	os.Remove(path + ".bak")
	v1 := `{"Version":1,"Run":` + old + `}`
	os.WriteFile(path, []byte(v1), 0o644)
	if h, err := LoadRun(path); err != nil || h.Nodes == nil {
		t.Fatalf("versioned old run: %+v %v", h, err)
	}
	if kept, _ := os.ReadFile(path + ".bak"); string(kept) != v1 {
		t.Fatalf("the versioned pre-Orrery run wasn't backed up: %q", kept)
	}
	os.Remove(path + ".bak")
	if err := SaveRun(path, h); err != nil {
		t.Fatal(err)
	}
	if h, err := LoadRun(path); err != nil || h.Gold != 321 {
		t.Fatalf("migrated save didn't round trip: %+v %v", h, err)
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Fatalf("a converted run was backed up again")
	}

	for name, data := range map[string]string{ "broken": "{not json", "newer": `{"Version":99}` } {
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		if h, err := LoadRun(path); err == nil || h != nil {
			t.Errorf("%s save should be an error and start fresh: %+v %v", name, h, err)
		}
		if kept, _ := os.ReadFile(path + ".bak"); string(kept) != data {
			t.Errorf("%s save wasn't kept as a backup: %q", name, kept)
		}
	}
}

func TestCardsSetTheBoardAndPayouts(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	files := memFiles{
		"s/story.yaml":     "start: a",
		"s/foes.yaml":      "- { id: rat, name: Rat, card: chaos }\n- { id: bat, name: Bat, card: nope }",
		"s/locations.yaml": "- { id: a, name: A, open: true }",
		"s/quests.yaml":    "[]",
		"s/cards.yaml": "- { id: chaos, name: Chaos, width: 10, height: 6, gravity: left, chaos_gravity: true, overcharge: true, gold_pct: 100, mana_pct: -50, skull_pct: 50 }\n" +
			"- { id: chaos, name: Twice }\n- { id: huge, width: 12, gravity: sideways, xp_pct: -200 }",
	}
	problems, err := LoadStory(files, "s")
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(problems, "\n")
	for _, w := range []string{ "nope", "defined twice", "size 12", "sideways", "-200" } {
		if !strings.Contains(all, w) {
			t.Errorf("no problem mentioning %q in:\n%s", w, all)
		}
	}

	card := RuleCardByID(FoeByID("rat").Card)
	b := board.NewBoard(BoardSettings(card, false), 1)
	if b.W() != 10 || b.H() != 6 || !b.Settings.Overcharge || !b.Settings.ChaosGravity || b.Settings.Gravity != board.Left {
		t.Fatalf("card didn't reach the board: %+v", b.Settings)
	}

	// the same step pays differently on the card
	pay := func(c *RuleCard) Gain {
		d := NewDuel(NewHeroCombatant(NewHero("Knight")), NewFoeCombatant(FoeByID("rat"), 1), 1)
		d.Turn, d.Card = Player, c
		return d.ApplyStep(step([]int{ Gold, Gold, Gold, Fire, Fire, Fire, Fire, Skull, Skull, Skull },
			board.Match{ Color: Fire, Size: 4, Mult: 1 }, board.Match{ Color: Skull, Size: 3, Mult: 1 }))
	}
	plain, carded := pay(nil), pay(card)
	if carded.Gold != plain.Gold*2 || carded.Mana[Fire] != plain.Mana[Fire]/2 || carded.Damage <= plain.Damage {
		t.Fatalf("payout modifiers: plain %+v, carded %+v", plain, carded)
	}
}

// playOut runs a duel the way the view does, with the AI on both sides.
func playOut(d *Duel) int {
	turns := 0
	for !d.Over() && turns < 2000 {
		turns++
		if s := d.ChooseSpell(); s != nil {
			d.Cast(s)
		} else if a, c, ok := chooseMove(d); ok {
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
		if it.QuestOnly || it.Level > h.Level {
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
	t.Parallel()	// it only reads the story, so it shares the CPU with the other simulator runs
	const N = 6
	foes := []struct {
		id    string
		level int
	}{{"goblin", 1}, {"barrow_lord", 5}, {"orc", 8}, {"high_priest", 12}, {"hollow_king", 16}}
	for _, class := range Classes {
		for _, f := range foes {
			wins, turns, perks := 0, 0, 0
			for seed := range N {
				h := NewHero(class.Name)
				for h.Level < f.level {
					h.AddXP(XPToNext(h.Level))
				}
				h.Residue[Colorless] = (h.Level - 1) * FAIR_RESIDUE_PER_LEVEL
				buyNodes(h, greedy(h.Class))	// perks included: a perked hero mustn't drag either
				bestShopGear(h)
				perks = len(heroPerks(h))
				d := NewDuel(NewHeroCombatant(h), NewFoeCombatant(FoeByID(f.id), f.level), int64(seed))
				d.SetBoard(board.NewBoard(BoardSettings(nil, false), int64(seed)))
				turns += playOut(d)
				if d.Winner == Player {
					wins++
				} else if d.Winner < 0 {
					t.Fatalf("%s vs %s never finished", class.Name, f.id)
				}
			}
			t.Logf("%-8s vs %-12s L%-2d: %2d/%d wins, %d turns avg, %d perks", class.Name, f.id, f.level, wins, N, turns/N, perks)
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
		candidate := board.NewBoard(BoardSettings(nil, false), seed)
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

// FAIR_RESIDUE_PER_LEVEL is roughly what TestProgression banks per level, to stand in for the
// Orrery when a test makes a hero of some level from nothing.
const FAIR_RESIDUE_PER_LEVEL = 20

// nodeWorth ranks a buyable node for the simulator: perks first, then own-class rites the hero
// can cast now, other castable rites, stats, and last rites for later, with the build's first
// region and nodes next to a wanted perk raised a step. Cheaper breaks ties.
func nodeWorth(h *Hero, n *OrreryNode, b build) int {
	w := 0
	if s := SpellByID(n.Rite); s != nil {
		if s.Level <= h.Level {
			w = 3000
			if s.Class == h.Class {
				w = 4000
			}
		}
	} else if n.Perk != nil {
		w = 5000
	} else {
		w = 2000
	}
	if n.Mana == b.regions[0] {
		w += 1500
	}
	// a build headed for Aether's core goes inward
	if len(b.regions) > 1 && b.regions[1] == Aether {
		w += (4 - n.Ring) * 600
	}
	// a step toward a perk the build wants is worth more than a stat on its own
	if n.Perk == nil && slices.ContainsFunc(n.Near, func(id string) bool {
		o := NodeByID(id)
		return o.Perk != nil && b.allows(h, o) && !h.Owns(id)
	}) {
		w += 1000
	}
	return w - n.Cost
}

// build is a buying policy for the simulator: the regions it buys in, the first preferred, and
// whether it takes perk nodes.
type build struct {
	name	string
	regions	[]int
	noPerks	bool
}

func (b build) allows(h *Hero, n *OrreryNode) bool {
	return slices.Contains(b.regions, n.Mana) && !(b.noPerks && n.Perk != nil)
}

// regionOrder is the regions around the Orrery, clockwise from the top.
var regionOrder = []int{ Fire, Water, Earth, Air }

// greedy buys anywhere, its own region first and Aether's core last, once it opens.
func greedy(class string) build {
	own := SeatOf(class).Mana
	others := slices.DeleteFunc(slices.Clone(regionOrder), func(c int) bool { return c == own })
	return build{ "greedy", append(append([]int{ own }, others...), Aether), false }
}

// builds are the policies TestBuilds compares for a class: anywhere, deep in its own region, its
// own region and the next one clockwise, and its own region and then Aether's core once it opens.
func builds(class string) []build {
	own := SeatOf(class).Mana
	next := regionOrder[(slices.Index(regionOrder, own)+1)%len(regionOrder)]
	return []build{
		greedy(class),
		{ "deep", []int{ own }, false },
		{ "cross:" + strings.ToLower(ManaNames[next]), []int{ own, next }, false },
		{ "core", []int{ own, Aether }, false },
	}
}

// buyNodes buys the best node the build allows next to what the hero owns. When it can't afford
// that yet it saves that color and colorless for it, and spends other colors on their own best
// nodes. Then it converts the residue of colors with nothing left to buy next to what the hero
// owns, and tries again.
func buyNodes(h *Hero, b build) {
	pick := func(ok func(n *OrreryNode) bool) *OrreryNode {
		var best *OrreryNode
		for i := range Orrery {
			n := &Orrery[i]
			if b.allows(h, n) && !h.Owns(n.ID) && h.Reaches(n) && ok(n) && (best == nil || nodeWorth(h, n, b) > nodeWorth(h, best, b)) {
				best = n
			}
		}
		return best
	}
	for {
		best := pick(func(*OrreryNode) bool { return true })
		if best != nil && h.BuyNode(best.ID) {
			continue
		}
		if best != nil {
			other := pick(func(n *OrreryNode) bool { return n.Mana != best.Mana && h.Residue[n.Mana] >= n.Cost })
			if other != nil && h.BuyNode(other.ID) {
				continue
			}
		}
		converted := false
		for c := range NUM_MANA {
			wanted := false
			for i := range Orrery {
				n := &Orrery[i]
				wanted = wanted || (n.Mana == c && b.allows(h, n) && !h.Owns(n.ID) && h.Reaches(n))
			}
			for !wanted && h.Convert(c) {
				converted = true
			}
		}
		if !converted {
			return
		}
	}
}

// spendLikeAPlayer is the simulator's fixed policy: residue buys Orrery nodes (buyNodes); gold
// buys the best gear the hero can afford in each slot, then henchmen (hire to the cap, train one
// rank at a time), then facility levels in list order; quest rewards are worn when they beat
// what's on. At the academy it claims reports and dues, and posts every free henchman to site,
// where the hero is about to fight. It returns the gold claimed from chapters and the ledger.
func spendLikeAPlayer(h *Hero, site string, b build) (dues int) {
	buyNodes(h, b)
	// quest items cost nothing, so rank by level
	worth := func(id string) int {
		if it := ItemByID(id); it != nil {
			return it.Level
		}
		return 0
	}
	for _, slot := range Slots {
		var best *Item
		for i := range Items {
			it := &Items[i]
			owned := slices.Contains(h.Owned, it.ID)
			if it.Slot != slot || (!owned && (it.QuestOnly || it.Price > h.Gold || it.Level > h.Level)) {
				continue
			}
			if worth(it.ID) > worth(h.Gear[slot]) && (best == nil || worth(it.ID) > worth(best.ID)) {
				best = it
			}
		}
		if best != nil {
			h.Buy(best)
		}
	}
	if !h.AcademyOpen() {
		return 0
	}
	gold := h.Gold
	for i := range h.Henchmen {
		h.Claim(i)
	}
	h.ClaimDues()
	dues = h.Gold - gold
	buyNodes(h, b)
	for h.Hire() {
	}
	for i := range h.Henchmen {
		h.Train(i)
		if h.Henchmen[i].Post != site {
			h.Recall(i)
			h.Post(i, site)
		}
	}
	for i := range Facilities {
		h.Build(&Facilities[i])
	}
	return dues
}

// TestProgression plays the main quests in order with the AI as the hero, one hero per class,
// and logs level, gold, residue earned so far in each color, Orrery nodes owned, gold claimed
// from chapters and dues so far, henchmen ranks and facility levels after each quest (see the
// table with -v). A lost duel is fought again, keeping what the board paid, as a player would.
// Economy numbers should come from here.
func TestProgression(t *testing.T) {
	t.Parallel()	// it only reads the story, so it shares the CPU with the other simulator runs
	for _, class := range Classes {
		h := progress(t, class.Name, greedy(class.Name), true)
		if h != nil && len(h.Nodes)*2 >= len(Orrery) {
			t.Errorf("%s owns %d of %d nodes after the main quests: choice doesn't matter", class.Name, len(h.Nodes), len(Orrery))
		}
	}
}

// TestBuilds follows a few buying policies per class through the main quests and prints where
// each ends, so builds can be compared (see the table with -v).
func TestBuilds(t *testing.T) {
	t.Parallel()	// it only reads the story, so it shares the CPU with the other simulator runs
	t.Logf("%-8s %-12s %3s %5s %4s %5s %5s  %s", "class", "build", "lvl", "duels", "lost", "turns", "nodes", "perks")
	// every run is its own hero, so they play at once and are reported in order
	var wg sync.WaitGroup
	heroes := map[string]*Hero{}
	var mu sync.Mutex
	for _, class := range Classes {
		for _, b := range builds(class.Name) {
			wg.Go(func() {
				h := progress(t, class.Name, b, false)
				mu.Lock()
				heroes[class.Name+b.name] = h
				mu.Unlock()
			})
		}
	}
	wg.Wait()
	for _, class := range Classes {
		for _, b := range builds(class.Name) {
			h := heroes[class.Name+b.name]
			if h == nil {
				continue
			}
			var perks []string
			for _, p := range heroPerks(h) {
				perks = append(perks, p.Node)
			}
			if len(perks) < 2 {
				t.Errorf("%s (%s) ends the main quests with %d perks", class.Name, b.name, len(perks))
			}
			t.Logf("%-8s %-12s %3d %5d %4d %5d %5d  %s", class.Name, b.name, h.Level, h.Wins+h.Losses, h.Losses,
				simTurns[h]/max(1, h.Wins+h.Losses), len(h.Nodes), strings.Join(perks, " "))
		}
	}
}

// simTurns is the turns each simulated hero's duels took, both sides counted.
var (
	simTurns	= map[*Hero]int{}
	simMu		sync.Mutex
)

// progress plays one hero of a class through the main quests, buying by a build, and returns them,
// or nil when they got stuck (which fails the test).
func progress(t *testing.T, class string, b build, verbose bool) *Hero {
	const MAX_TRIES = 30	// duels per kill before the hero counts as stuck
	h := NewHero(class)
	seed := int64(0)
	var earned [NUM_MANA]int
	dues := 0
	if verbose {
		t.Logf("%-8s %-14s %3s %5s %4s %5s %4s %4s %4s %4s %4s %5s %4s %4s %4s", class, "quest", "lvl", "duels", "lost", "gold", "fire", "watr", "air", "erth", "aeth", "nodes", "dues", "hm", "fac")
	}
	for i := range Quests {
		q := &Quests[i]
		if !q.Main {
			continue
		}
		h.Location = q.Giver
		h.Accept(q)
		h.Location = q.At
		dues += spendLikeAPlayer(h, q.At, b)	// post henchmen where the fighting will be
		card := RuleCardByID(FoeByID(q.Foe).Card)
		duels, lost := 0, 0
		for !slices.Contains(h.Done, q.ID) && duels < MAX_TRIES*q.Count {
			seed++
			duels++
			d := NewDuel(NewHeroCombatant(h), NewFoeCombatant(FoeByID(q.Foe), q.FoeLevel), seed)
			d.SetBoard(board.NewBoard(BoardSettings(card, h.AetherOn()), seed))
			d.Card = card
			n := playOut(d)
			simMu.Lock()
			simTurns[h] += n
			simMu.Unlock()
			if d.Winner < 0 {
				t.Errorf("%s vs %s never finished", class, q.Foe)
				return nil
			}
			r := d.Settle(h)
			for c, n := range r.Residue {
				earned[c] += n
			}
			if r.Won {
				h.RecordWin(q.At)
				for _, done := range h.RecordKill(q.Foe, q.At) {
					// the scene a quest ends with may set flags, the academy's founding among them
					for _, st := range Scenes[done.DoneScene] {
						if st.SetFlag != "" {
							h.Flags[st.SetFlag] = true
						}
					}
				}
			} else {
				lost++
			}
			dues += spendLikeAPlayer(h, q.At, b)
		}
		ranks, levels := 0, 0
		for _, m := range h.Henchmen {
			ranks += m.Rank
		}
		for _, n := range h.Facilities {
			levels += n
		}
		if verbose {
			t.Logf("%-8s %-14s %3d %5d %4d %5d %4d %4d %4d %4d %4d %5d %4d %4d %4d", class, q.ID, h.Level, duels, lost, h.Gold,
				earned[Fire], earned[Water], earned[Air], earned[Earth], earned[Aether], len(h.Nodes), dues, ranks, levels)
		}
		if !slices.Contains(h.Done, q.ID) {
			t.Errorf("%s (%s) is stuck on %s at level %d", class, b.name, q.ID, h.Level)
			return nil
		}
	}
	return h
}

func TestResiduePaysForQuality(t *testing.T) {
	d := testDuel()
	if g := d.ApplyStep(step([]int{ Fire, Fire, Fire }, board.Match{ Color: Fire, Size: 3, Mult: 1 })); g.Residue != [NUM_MANA]int{} {
		t.Fatalf("a plain three paid residue: %v", g.Residue)
	}
	g := d.ApplyStep(step([]int{ Fire, Fire, Fire, Fire, Water, Water, Water, Water, Water },
		board.Match{ Color: Fire, Size: 4, Mult: 1 }, board.Match{ Color: Water, Size: 5, Mult: 1 }))
	if g.Residue != [NUM_MANA]int{ RESIDUE_FOUR, RESIDUE_FIVE, 0, 0 } {
		t.Fatalf("four and five: %v", g.Residue)
	}
	chained := step([]int{ Earth, Earth, Earth }, board.Match{ Color: Earth, Size: 3, Mult: 1 })
	chained.Chain = 3
	if g := d.ApplyStep(chained); g.Residue[Earth] != 2*RESIDUE_CHAIN {
		t.Fatalf("a third cascade step: %v", g.Residue)
	}
	// the foe's matches pay the hero nothing, and leave no residue anywhere
	d.Turn = Enemy
	d.ApplyStep(step([]int{ Air, Air, Air, Air, Air }, board.Match{ Color: Air, Size: 5, Mult: 1 }))
	if d.Sides[Enemy].Residue != [NUM_MANA]int{} {
		t.Fatalf("the foe collected residue: %v", d.Sides[Enemy].Residue)
	}

	// banked whether the duel is won or lost, like the rest of the board's pay
	h := NewHero("Knight")
	d.Winner = Enemy
	r := d.Settle(h)
	want := [NUM_RESIDUE]int{ RESIDUE_FOUR, RESIDUE_FIVE, 0, 2 * RESIDUE_CHAIN, 0 }
	if r.Won || h.Residue != want {
		t.Fatalf("banked %v after a loss, want %v", h.Residue, want)
	}
}

func TestOrreryBuying(t *testing.T) {
	h := NewHero("Knight")
	seat := SeatOf("Knight")
	if seat == nil || !slices.Equal(h.Nodes, []string{ seat.ID }) {
		t.Fatalf("knight not on a seat: %v", h.Nodes)
	}
	var near, far *OrreryNode
	for i := range Orrery {
		n := &Orrery[i]
		if n.Seat != "" {
			continue
		}
		if slices.Contains(seat.Near, n.ID) && near == nil {
			near = n
		} else if !slices.Contains(seat.Near, n.ID) && far == nil {
			far = n
		}
	}
	h.Residue = [NUM_RESIDUE]int{ 999, 999, 999, 999, 0 }
	if h.BuyNode(far.ID) {
		t.Fatalf("bought %s, which isn't next to anything owned", far.ID)
	}
	h.Residue = [NUM_RESIDUE]int{}
	h.Residue[near.Mana] = near.Cost - 1
	if h.BuyNode(near.ID) {
		t.Fatalf("bought %s without enough residue", near.ID)
	}
	// colorless makes up what the color is short
	h.Residue[Colorless] = 3
	if !h.BuyNode(near.ID) || h.Residue[near.Mana] != 0 || h.Residue[Colorless] != 2 {
		t.Fatalf("buying %s: %v %v", near.ID, h.Nodes, h.Residue)
	}
	if h.BuyNode(near.ID) {
		t.Fatalf("bought %s twice", near.ID)
	}
}

// pathTo is the shortest run of nodes from a seat to a node, the seat left out.
func pathTo(from, to string) []string {
	prev := map[string]string{ from: "" }
	todo := []string{ from }
	for len(todo) > 0 && prev[to] == "" {
		n := NodeByID(todo[0])
		todo = todo[1:]
		for _, id := range n.Near {
			if _, ok := prev[id]; !ok {
				prev[id] = n.ID
				todo = append(todo, id)
			}
		}
	}
	var out []string
	for id := to; id != from; id = prev[id] {
		out = append([]string{ id }, out...)
	}
	return out
}

func TestTwoSeatsReachTheSameNode(t *testing.T) {
	a, b := NewHero("Knight"), NewHero("Wizard")
	if a.Nodes[0] == b.Nodes[0] {
		t.Fatalf("both start on %s", a.Nodes[0])
	}
	target := SeatOf("Druid").Near[0]
	for _, h := range []*Hero{ a, b } {
		h.Residue[Colorless] = 999
		for _, id := range pathTo(h.Nodes[0], target) {
			if !h.BuyNode(id) {
				t.Fatalf("%s couldn't buy %s on the way to %s", h.Class, id, target)
			}
		}
		if !h.Owns(target) {
			t.Fatalf("%s didn't reach %s", h.Class, target)
		}
	}
}

func TestRitesComeFromTheOrrery(t *testing.T) {
	h := NewHero("Knight")
	for h.Level < 13 {
		h.AddXP(XPToNext(h.Level))
	}
	// a Warrior rite on the Orrery: the Knight can learn it there, but only there
	n := &Orrery[slices.IndexFunc(Orrery, func(n OrreryNode) bool { return n.Rite == "hamstring" })]
	if h.Knows(SpellByID("hamstring")) || h.Knows(SpellByID("judgement")) {
		t.Fatalf("knew rites before buying them")
	}
	if !h.Knows(SpellByID("smite")) || !h.Knows(SpellByID("lay_hands")) {
		t.Fatalf("the Knight's starting kit is missing")
	}
	h.Residue[Colorless] = 999
	for _, id := range pathTo(h.Nodes[0], n.ID) {
		h.BuyNode(id)
	}
	if !h.Knows(SpellByID("hamstring")) {
		t.Fatalf("owning %s didn't teach hamstring", n.ID)
	}
	low := NewHero("Knight")
	low.Nodes = h.Nodes
	if low.Knows(SpellByID("hamstring")) {
		t.Fatalf("a level 1 hero knows a level %d rite", SpellByID("hamstring").Level)
	}
}

func TestRespecRefunds(t *testing.T) {
	h := NewHero("Druid")
	h.Gold = RESPEC_GOLD + 5
	h.Residue = [NUM_RESIDUE]int{ 100, 100, 100, 100, 0 }
	if h.Respec() {
		t.Fatalf("respec with nothing bought")
	}
	before := h.Residue
	for _, id := range pathTo(h.Nodes[0], SeatOf("Wizard").Near[0]) {
		if !h.BuyNode(id) {
			t.Fatalf("couldn't buy %s", id)
		}
	}
	if !h.Respec() || h.Residue != before || len(h.Nodes) != 1 || h.Gold != 5 {
		t.Fatalf("respec: residue %v (was %v), nodes %v, gold %d", h.Residue, before, h.Nodes, h.Gold)
	}
	// colorless that paid for a node goes back to colorless
	h.Gold = RESPEC_GOLD
	n := NodeByID(h.Nodes[0]).Near[0]
	h.Residue = [NUM_RESIDUE]int{ 0, 0, 0, 0, 0 }
	h.Residue[NodeByID(n).Mana] = 1
	h.Residue[Colorless] = 50
	before = h.Residue
	if !h.BuyNode(n) || !h.Respec() || h.Residue != before {
		t.Fatalf("respec after paying in colorless: %v, was %v", h.Residue, before)
	}
	for _, id := range h.Spells {
		if !h.Knows(SpellByID(id)) {
			t.Fatalf("%s is still equipped after the respec", id)
		}
	}
	if h.Respec() {
		t.Fatalf("respec without the gold")
	}
}

func TestConvertLosesResidue(t *testing.T) {
	h := NewHero("Wizard")
	h.Residue[Fire] = CONVERT_STEP + 1
	if !h.Convert(Fire) || h.Residue[Fire] != 1 || h.Residue[Colorless] != CONVERT_STEP*CONVERT_PCT/100 {
		t.Fatalf("convert: %v", h.Residue)
	}
	if CONVERT_PCT >= 100 || h.Convert(Fire) || h.Convert(Colorless) {
		t.Fatalf("conversion should cost, and need a full step of a color: %v", h.Residue)
	}
}

func TestOrreryValidation(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	files := memFiles{
		"s/story.yaml":     "start: a",
		"s/foes.yaml":      "- { id: rat, name: Rat }",
		"s/locations.yaml": "- { id: a, name: A, open: true }",
		"s/quests.yaml":    "[]",
		"s/orrery.yaml": "- { id: k, color: water, ring: 1, seat: Knight, links: [x] }\n" +
			"- { id: w, color: air, ring: 1, seat: Wizard, links: [k, nowhere] }\n" +
			"- { id: x, color: mud, ring: 1, cost: 5, rite: smite, stats: { luck: 2 } }\n" +
			"- { id: island, color: fire, ring: 2, cost: 0 }",
	}
	problems, err := LoadStory(files, "s")
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(problems, "\n")
	for _, w := range []string{ "island: can't be reached", "nowhere", "mud", "luck", "Knight kit", "cost 0", "class Druid has no seat" } {
		if !strings.Contains(all, w) {
			t.Errorf("no problem mentioning %q in:\n%s", w, all)
		}
	}
}
