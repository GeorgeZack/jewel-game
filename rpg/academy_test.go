package rpg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func foundedHero() *Hero {
	h := NewHero("Knight")
	h.Flags[FOUNDING_FLAG] = true
	return h
}

func TestAcademyIsLockedUntilFounded(t *testing.T) {
	h := NewHero("Knight")
	h.Gold = 5000
	if h.AcademyOpen() || h.Hire() || h.Build(FacilityByID("quarters")) {
		t.Fatalf("a new hero used the academy: %+v", h)
	}
	// the first main quest's done scene founds it
	q := QuestByID("goblins")
	if q == nil || !setsFlag(Scenes[q.DoneScene], FOUNDING_FLAG) {
		t.Fatalf("goblins doesn't found the academy")
	}
	h.Flags[FOUNDING_FLAG] = true
	if !h.AcademyOpen() || !h.Hire() {
		t.Fatalf("the founded academy is still shut")
	}
}

func TestFacilityNeedsItsConditionAndGold(t *testing.T) {
	h := foundedHero()
	f := FacilityByID("treasury")
	h.Gold = 5000
	if h.FacilityAvailable(f) || h.Build(f) {
		t.Fatalf("built the treasury before its quest")
	}
	h.Done = append(h.Done, "toll")
	h.Gold = f.Levels[0].Cost - 1
	if h.Build(f) {
		t.Fatalf("built without the gold")
	}
	h.Gold = 5000
	before := NewHeroCombatant(h).GoldPct
	for i, l := range f.Levels {
		if !h.Build(f) || h.FacilityLevel(f.ID) != i+1 {
			t.Fatalf("level %d didn't build", i+1)
		}
		if got := NewHeroCombatant(h).GoldPct; got != before+l.GoldPct {
			t.Fatalf("level %d gold %d, want %d", i+1, got, before+l.GoldPct)
		}
		before += l.GoldPct
	}
	gold := h.Gold
	if h.Build(f) || h.Gold != gold {
		t.Fatalf("built past the top level")
	}
	// the quarters raise the henchmen cap
	q := FacilityByID("quarters")
	h.Build(q)
	if h.HenchmenCap() != HENCHMEN_CAP+q.Levels[0].Henchmen {
		t.Fatalf("cap %d", h.HenchmenCap())
	}
}

func TestHenchmenHireTrainEquip(t *testing.T) {
	h := foundedHero()
	h.Gold = HIRE_GOLD - 1
	if h.Hire() {
		t.Fatalf("hired without the gold")
	}
	h.Gold = 5000
	for range HENCHMEN_CAP {
		if !h.Hire() {
			t.Fatalf("couldn't hire")
		}
	}
	if h.Hire() {
		t.Fatalf("hired past the cap")
	}
	for r := 1; r < MAX_RANK; r++ {
		gold := h.Gold
		if !h.Train(0) || h.Henchmen[0].Rank != r+1 || gold-h.Gold != TRAIN_GOLD*r {
			t.Fatalf("training from rank %d: %+v", r, h.Henchmen[0])
		}
	}
	if h.Train(0) {
		t.Fatalf("trained past the top rank")
	}
	// gear must be owned and not in use
	if h.Equip(0, "chain") {
		t.Fatalf("equipped an item the hero doesn't own")
	}
	h.Buy(ItemByID("leather"))
	if h.Equip(0, "leather") {
		t.Fatalf("equipped an item the hero is wearing")
	}
	h.GiveItem("chain")
	if !h.Equip(0, "chain") || h.Henchmen[0].Gear != "chain" {
		t.Fatalf("couldn't equip a spare item")
	}
	// the hero putting it on takes it back
	h.Buy(ItemByID("chain"))
	if h.Henchmen[0].Gear != "" {
		t.Fatalf("the hero and a henchman share an item")
	}
}

func TestChapterAdvancesOnWinsAndPaysByRank(t *testing.T) {
	pay := func(rank int, gear string) *Report {
		h := foundedHero()
		h.Henchmen = []Henchman{ { Rank: rank } }
		h.GiveItem(gear)
		h.Equip(0, gear)
		if h.Post(0, "barrow") {
			t.Fatalf("posted to a site that isn't open")
		}
		if !h.Post(0, "mossgate") || h.Post(0, "mossgate") {
			t.Fatalf("posting: %+v", h.Henchmen)
		}
		// a win elsewhere, or anything but a win, moves nothing
		h.RecordWin("emberfield")
		if h.Henchmen[0].Wins != 0 {
			t.Fatalf("a win elsewhere moved the chapter")
		}
		for i := 1; i < CHAPTER_WINS; i++ {
			if len(h.RecordWin("mossgate")) != 0 {
				t.Fatalf("completed early")
			}
		}
		if h.Henchmen[0].Wins != CHAPTER_WINS-1 || h.Held("mossgate") {
			t.Fatalf("progress %+v", h.Henchmen[0])
		}
		done := h.RecordWin("mossgate")
		if len(done) != 1 || !h.Held("mossgate") || done[0].Line == "" {
			t.Fatalf("chapter didn't complete: %+v", h)
		}
		gold, res := h.Gold, h.Residue
		if !h.Claim(0) || h.Henchmen[0].Post != "" || h.Henchmen[0].Report != nil || h.Gold != gold+done[0].Gold {
			t.Fatalf("claim: %+v", h)
		}
		if c := SiteColor("mossgate"); c != Earth || h.Residue[c] != res[c]+done[0].Residue[c] || done[0].Residue[c] == 0 {
			t.Fatalf("residue: %v", h.Residue)
		}
		if h.Claim(0) || h.Recall(0) {
			t.Fatalf("claimed twice, or recalled a free henchman")
		}
		h.Post(0, "emberfield")
		h.RecordWin("emberfield")
		if !h.Recall(0) || h.Henchmen[0].Post != "" || h.Henchmen[0].Wins != 0 {
			t.Fatalf("recall: %+v", h.Henchmen[0])
		}
		return done[0]
	}
	low, high, geared := pay(1, ""), pay(3, ""), pay(1, "plate")
	if low.Gold != CHAPTER_GOLD || high.Gold != 3*CHAPTER_GOLD || geared.Gold <= low.Gold {
		t.Fatalf("pay by rank and gear: %d %d %d", low.Gold, high.Gold, geared.Gold)
	}
}

func TestDuesAccumulateAndClaimEmpties(t *testing.T) {
	h := foundedHero()
	h.RecordWin("mossgate")
	if h.DuesOwed() != 0 {
		t.Fatalf("dues from a site no one holds")
	}
	h.SetHolder("mossgate", OUR_SOCIETY)
	h.RecordWin("mossgate")
	h.RecordWin("mossgate")
	h.RecordWin("emberfield")
	if h.DuesOwed() != 2*DUES_PER_WIN {
		t.Fatalf("dues %v", h.Dues)
	}
	gold := h.Gold
	if h.ClaimDues() != 2*DUES_PER_WIN || h.Gold != gold+2*DUES_PER_WIN || h.DuesOwed() != 0 || len(h.Dues) != 0 {
		t.Fatalf("claim: %+v", h)
	}
	h.SetHolder("mossgate", "")
	if h.Held("mossgate") || len(h.Holders) != 0 {
		t.Fatalf("holder not cleared: %v", h.Holders)
	}
}

// A run from before the academy refunds its buildings and mounts, and is backed up once.
func TestOldRunConvertsToTheAcademy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hero.json")
	old := `{"Version":1,"Run":{"Class":"Knight","Level":3,"Gold":100,"Stats":[2,2,1,3,5,2,6],` +
		`"Nodes":["water_seat"],"Gear":{"Mount":"pony","Weapon":"rusty_blade"},"Owned":["pony","rusty_blade"],` +
		`"Spells":[],"Buildings":["treasury","stables"],"Location":"emberfield","Active":{},"Done":["goblins"]}}`
	os.WriteFile(path, []byte(old), 0o644)
	h, err := LoadRun(path)
	if err != nil || h == nil {
		t.Fatal(err)
	}
	if h.Gold != 100+200+250+150 || len(h.Owned) != 1 || h.Gear["Mount"] != "" || h.Gear[SlotWeapon] != "rusty_blade" {
		t.Fatalf("not converted: %+v", h)
	}
	if !h.AcademyOpen() || len(h.Facilities) != 0 {
		t.Fatalf("goblins was done, so the academy should be founded and empty: %+v", h)
	}
	if kept, _ := os.ReadFile(path + ".bak"); string(kept) != old {
		t.Fatalf("not backed up: %q", kept)
	}
	os.Remove(path + ".bak")
	SaveRun(path, h)
	if again, err := LoadRun(path); err != nil || again.Gold != h.Gold {
		t.Fatalf("converted run didn't round trip: %+v %v", again, err)
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Fatalf("a converted run was backed up again")
	}
}

func TestFacilitiesAndHolderStepValidate(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	files := memFiles{
		"s/story.yaml":     "start: a",
		"s/foes.yaml":      "- { id: rat, name: Rat }",
		"s/locations.yaml": "- { id: a, name: A, open: true, residue: water }\n- { id: b, name: B, residue: lava }",
		"s/quests.yaml":    "[]",
		"s/facilities.yaml": "- { id: f, name: F, requires: [quest:nope], levels: [ { cost: 10 }, { gold_pct: 5 } ] }\n" +
			"- { id: f, name: Twice, levels: [ { cost: 1 } ] }\n- { id: g, name: G }",
		"s/scenes/x.yaml": "x:\n  - set_holder: { at: nowhere, holder: academy }\n  - set_holder: { at: a, holder: crown }",
	}
	problems, err := LoadStory(files, "s")
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(problems, "\n")
	for _, w := range []string{ "unknown quest", "level 2 needs a gold cost", "defined twice", "facility g: needs at least one level", "set_holder at unknown location \"nowhere\"", "location b: residue \"lava\"" } {
		if !strings.Contains(all, w) {
			t.Errorf("no problem mentioning %q in:\n%s", w, all)
		}
	}
	if strings.Count(all, "set_holder") != 1 {
		t.Errorf("a good set_holder was flagged:\n%s", all)
	}
	if len(Facilities) != 3 {
		t.Fatalf("facilities not loaded: %v", Facilities)
	}
}
