package rpg

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// useProfile points CurrentProfile at p for one test.
func useProfile(t *testing.T, p *Profile) {
	t.Helper()
	saved := CurrentProfile
	t.Cleanup(func() { CurrentProfile = saved })
	CurrentProfile = p
}

// A profile finishes the fixture campaign, returns, and finishes it again as another class on the
// other branch. Its Grimoire, endings and banked currency carry over; the run does not.
func TestProfileFinishesReturnsAndKeepsItsGrimoire(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	loadFixture(t)
	p := &Profile{}
	useProfile(t, p)

	h := finishMainQuestsAs(t, StartRun("Wizard"), "crown")
	if !h.Seen["tutorial"] || h.Seen["the_return"] {
		t.Errorf("first run should play the tutorial and no return scene: %v", h.Seen)
	}
	// route 5, lock 15, boss 10, class 20, ending 40: all first times, all banked by the end
	if p.Meta != 90 || h.Forfeit() != 0 {
		t.Fatalf("first run banked %d, %d unbanked", p.Meta, h.Forfeit())
	}
	if !p.Buy("purse") || !p.Buy("marked") || p.Buy("purse") || p.Meta != 70 {
		t.Fatalf("buying: %+v", p)
	}
	if n := Return(h); n != 0 || p.Returns != 1 || !p.TutorialDone {
		t.Fatalf("return at the ending: forfeit %d, %+v", n, p)
	}

	h2 := StartRun("Knight")
	if h2.Gold != 150 || !h2.Flags["marked"] || h2.Residue[Colorless] != 5 || h2.Allegiance != "" || h2.CurrentAct() != 1 {
		t.Fatalf("second run doesn't start fresh with the Grimoire: %+v", h2)
	}
	h2 = finishMainQuestsAs(t, h2, "free")
	if h2.Class != "Knight" || h2.Allegiance != "free" {
		t.Errorf("second run: %s, %s", h2.Class, h2.Allegiance)
	}
	if h2.Seen["tutorial"] || !h2.Seen["the_return"] {
		t.Errorf("a later run skips the tutorial and plays the return scene: %v", h2.Seen)
	}
	if !slices.Equal(p.Grimoire, []string{ "purse", "marked" }) || !slices.Equal(p.Endings, []string{ "crown_ending", "free_ending" }) {
		t.Errorf("profile lost something: %+v", p)
	}
	// route and boss repeat at 20% (1 and 2); lock free, class Knight and the free ending are new
	if p.Meta != 70+1+15+2+20+40 {
		t.Errorf("second run banked to %d", p.Meta)
	}
}

// Returning mid-act forfeits only what the current act has earned; what earlier acts banked stays,
// and a first time that was forfeited still pays in full later.
func TestEarlyReturnLosesOnlyTheCurrentAct(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	loadFixture(t)
	p := &Profile{}
	useProfile(t, p)

	h := StartRun("Druid")
	playScene(t, h, Story.Opening, "crown")	// the vault's route: 5
	if h.Forfeit() != 5 || p.Meta != 0 {
		t.Fatalf("act 1 earned %d, banked %d", h.Forfeit(), p.Meta)
	}
	if a := h.Apply(&Step{ Act: 2 }); !a.Banked || p.Meta != 5 || h.Forfeit() != 0 {
		t.Fatalf("the act's end didn't bank: %+v, banked %d", a, p.Meta)
	}
	h.Apply(&Step{ JoinSociety: "crown" })
	if h.Forfeit() != 15 {
		t.Fatalf("the lock earned %d", h.Forfeit())
	}
	if n := Return(h); n != 15 || p.Meta != 5 {
		t.Fatalf("early return forfeited %d, kept %d", n, p.Meta)
	}
	if p.Firsts["lock:crown"] != 0 || p.Firsts["route:vault/force"] != 1 {
		t.Errorf("firsts %v", p.Firsts)
	}
	h2 := StartRun("Druid")
	h2.Apply(&Step{ JoinSociety: "crown" })
	if h2.Forfeit() != 15 {
		t.Errorf("a forfeited first should pay in full next run, paid %d", h2.Forfeit())
	}
}

// A first time pays in full once, nothing again in the same run, and the repeat fraction in a later run.
func TestFirstTimesPayOnceThenAFraction(t *testing.T) {
	p := &Profile{}
	useProfile(t, p)
	pay := Grimoire.Pay
	if pay.Boss <= 0 || pay.RepeatPct <= 0 || pay.RepeatPct >= 100 {
		t.Fatalf("shipped payouts look wrong: %+v", pay)
	}
	h := NewHero("Warrior")
	h.RecordKill("hollow_king", "nowhere")
	h.RecordKill("hollow_king", "nowhere")
	h.RecordKill("goblin", "nowhere")	// not a boss
	if h.Forfeit() != pay.Boss {
		t.Fatalf("first run earned %d, want %d", h.Forfeit(), pay.Boss)
	}
	h.bank()
	h2 := NewHero("Warrior")
	h2.RecordKill("hollow_king", "nowhere")
	if want := pay.Boss * pay.RepeatPct / 100; h2.Forfeit() != want {
		t.Errorf("repeat earned %d, want %d", h2.Forfeit(), want)
	}
	// an ending pays the most, and finishing pays for the class
	h3 := NewHero("Druid")
	h3.Apply(&Step{ Ending: "king_falls" })
	if p.Meta != pay.Boss+pay.Ending+pay.Class || pay.Ending <= pay.Boss {
		t.Errorf("ending banked to %d", p.Meta)
	}
}

// Every shipped Grimoire entry does what it says from the start of a run.
func TestGrimoireEntriesApplyAtRunStart(t *testing.T) {
	if len(Grimoire.Entries) == 0 {
		t.Fatal("no Grimoire entries shipped")
	}
	kinds := map[string]bool{}
	for i := range Grimoire.Entries {
		e := &Grimoire.Entries[i]
		kinds[e.Lean] = true
		useProfile(t, &Profile{ Grimoire: []string{ e.ID } })
		base := NewHero("Knight")
		h := StartRun("Knight")
		changed := false
		if h.Gold != base.Gold+e.Gold {
			t.Errorf("%s: gold %d", e.ID, h.Gold)
		}
		changed = changed || e.Gold != 0
		for _, f := range e.Flags {
			if !h.Flags[f] {
				t.Errorf("%s: flag %s not set", e.ID, f)
			}
			changed = true
		}
		for f, n := range e.Facilities {
			if h.FacilityLevel(f) != n {
				t.Errorf("%s: %s at level %d", e.ID, f, h.FacilityLevel(f))
			}
			changed = true
		}
		for _, q := range e.Quests {
			if !h.Meets([]string{ "quest:" + q }) {
				t.Errorf("%s: quest %s not done", e.ID, q)
			}
			for _, l := range QuestByID(q).Opens {
				if !h.LocationOpen(LocationByID(l)) {
					t.Errorf("%s: %s not open", e.ID, l)
				}
			}
			changed = true
		}
		for r, n := range e.Residue {
			if h.Residue[residueByName(r)] != n {
				t.Errorf("%s: residue %s %d", e.ID, r, h.Residue[residueByName(r)])
			}
			changed = true
		}
		for _, id := range e.Nodes {
			if !h.Owns(id) {
				t.Errorf("%s: doesn't own %s", e.ID, id)
			}
			h.refundNodes()	// a respec keeps the Grimoire's gifts
			if !h.Owns(id) {
				t.Errorf("%s: a respec took %s", e.ID, id)
			}
			changed = true
		}
		if !changed {
			t.Errorf("%s does nothing", e.ID)
		}
	}
	if !kinds[""] || !kinds[LEAN_TAKE] || !kinds[LEAN_RESIST] {
		t.Errorf("need entries of each kind, have %v", kinds)
	}
}

// Stage 2 entries are hidden, and can't be bought, until the profile has returned enough.
func TestStageTwoOpensAfterTheThreshold(t *testing.T) {
	var take *GrimoireEntry
	for i := range Grimoire.Entries {
		if Grimoire.Entries[i].Lean == LEAN_TAKE {
			take = &Grimoire.Entries[i]
		}
	}
	if take == nil || Grimoire.StageTwoReturns < 1 {
		t.Fatal("no stage 2 entry, or no threshold")
	}
	p := &Profile{ Returns: Grimoire.StageTwoReturns - 1, Meta: 1000 }
	if p.EntryShown(take) || p.Buy(take.ID) {
		t.Fatal("stage 2 entry open before the threshold")
	}
	p.Returns++
	if !p.EntryShown(take) || !p.Buy(take.ID) || p.Lean != 1 || p.LeanName() != LEAN_TAKE {
		t.Fatalf("stage 2 at the threshold: %+v", p)
	}
	useProfile(t, p)
	if !NewHero("Knight").Meets([]string{ "lean:take", "!lean:resist" }) {
		t.Error("lean condition")
	}
}

// A run saved before the meta currency loads as is; a profile saved before it is paid for the
// endings it had seen, once.
func TestMetaSavesMigrate(t *testing.T) {
	dir := t.TempDir()
	path, ppath := filepath.Join(dir, "hero.json"), filepath.Join(dir, "profile.json")
	h := NewHero("Wizard")
	h.Gold = 77
	if err := SaveRun(path, h); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	v5 := strings.Replace(string(data), `"Version": `+strconv.Itoa(SAVE_VERSION), `"Version": 5`, 1)
	if v5 == string(data) {
		t.Fatal("couldn't find the version in the save")
	}
	os.WriteFile(path, []byte(v5), 0o644)
	got, err := LoadRun(path)
	if err != nil || got == nil || got.Gold != 77 || got.Forfeit() != 0 || got.Gifts != nil {
		t.Fatalf("v5 run: %+v %v", got, err)
	}

	old := `{"Version":1,"Profile":{"Endings":["king_falls"],"Returns":0,"TutorialDone":false}}`
	os.WriteFile(ppath, []byte(old), 0o644)
	p, err := LoadProfile(ppath)
	if err != nil || p.Meta != Grimoire.Pay.Ending || p.Firsts["ending:king_falls"] != 1 {
		t.Fatalf("v1 profile: %+v %v", p, err)
	}
	if kept, _ := os.ReadFile(ppath + ".bak"); string(kept) != old {
		t.Errorf("the old profile wasn't backed up: %q", kept)
	}
	SaveProfile(ppath, p)
	if p2, err := LoadProfile(ppath); err != nil || p2.Meta != p.Meta {
		t.Errorf("migrated profile paid again or didn't load: %+v %v", p2, err)
	}
	if _, err := LoadProfile(ppath + ".none"); err != nil {
		t.Error(err)
	}
}
