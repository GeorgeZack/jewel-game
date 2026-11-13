package rpg

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// playScene runs a scene the way the scene player does, with no screen: every step's changes
// happen, a choice that joins `allegiance` is taken over the others (else the first offered), duels
// are won, and a puzzle is solved by its first route, replaying the stored solution.
func playScene(t *testing.T, h *Hero, id, allegiance string) {
	t.Helper()
	budget := 1000
	var run func(steps []Step) (stop bool)
	jump := func(id string) bool {
		if id != "" {
			h.Seen[id] = true
			run(Scenes[id])
			return true
		}
		return false
	}
	run = func(steps []Step) bool {
		for i := range steps {
			st := &steps[i]
			if budget--; budget < 0 {
				t.Fatalf("scene %s runs without stopping", id)
			}
			if len(st.If) > 0 {
				ok := h.Meets(st.If)
				if st.Then != nil || st.Else != nil {
					branch := st.Else
					if ok {
						branch = st.Then
					}
					if run(branch) {
						return true
					}
					continue
				}
				if !ok {
					continue
				}
			}
			h.Apply(st)
			if len(st.Choices) > 0 {
				var pick *Choice
				for j := range st.Choices {
					c := &st.Choices[j]
					if !h.Meets(c.If) {
						continue
					}
					joins := slices.ContainsFunc(c.Do, func(s Step) bool { return s.JoinSociety == allegiance && allegiance != "" })
					if pick == nil || joins {
						pick = c
					}
					if joins {
						break
					}
				}
				if pick != nil {
					if run(pick.Do) || jump(pick.Goto) {
						return true
					}
				}
			}
			if st.Run != "" {
				h.Seen[st.Run] = true
				run(Scenes[st.Run])
			}
			if jump(st.Goto) {
				return true
			}
			if d := st.Duel; d != nil && jump(d.Win) {
				return true
			}
			if pz := st.Puzzle; pz != nil {
				p := PuzzleByID(pz.ID)
				r, err := p.Replay(p.Routes[0].Solution)
				if err != nil || r.State != PuzzleWon {
					t.Fatalf("puzzle %s: %v", p.ID, err)
				}
				r.Finish(h)
				if jump(pz.Routes[r.Route.ID]) {
					return true
				}
			}
		}
		return false
	}
	h.Seen[id] = true
	run(Scenes[id])
}

// finishMainQuests plays the main quests in order as a player on one branch would: each quest's
// done scene runs, choices take `allegiance` at the lock, and a main quest that asks for another
// allegiance belongs to another branch and is passed over.
func finishMainQuests(t *testing.T, allegiance string) *Hero {
	t.Helper()
	h := NewHero("Wizard")
	if Story.Opening != "" {
		playScene(t, h, Story.Opening, allegiance)
	}
	for i := range Quests {
		q := &Quests[i]
		if !q.Main {
			continue
		}
		if other := slices.ContainsFunc(q.Requires, func(c string) bool {
			return strings.HasPrefix(c, "allegiance:") && c != "allegiance:"+allegiance
		}); other {
			continue
		}
		offered := false
		if h.Route(q.Giver) != nil || h.Location == q.Giver {
			for _, o := range h.OfferedQuests(q.Giver) {
				offered = offered || o.ID == q.ID
			}
		}
		if !offered {
			t.Fatalf("%s (allegiance %q) isn't offered", q.ID, allegiance)
		}
		h.Location = q.Giver
		h.Accept(q)
		if q.At != h.Location && h.Route(q.At) == nil {
			t.Fatalf("%s: can't reach %s", q.ID, q.At)
		}
		h.Location = q.At
		for range q.Count {
			h.RecordKill(q.Foe, q.At)
		}
		if q.DoneScene != "" {
			playScene(t, h, q.DoneScene, allegiance)
		}
	}
	if !h.Finished {
		t.Fatalf("story not finished (allegiance %q)", allegiance)
	}
	return h
}

// eachAllegiance is every branch of the current story: its allegiances, or one pass without.
func eachAllegiance() []string {
	if a := Allegiances(); len(a) > 0 {
		return a
	}
	return []string{ "" }
}

func loadFixture(t *testing.T) {
	t.Helper()
	problems, err := LoadStory(diskFiles{}, "testdata/branching")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// The fixture campaign uses every part of the format the shipped story doesn't, and loads clean.
// Each of its branches can be finished in order, and ends differently.
func TestBranchingCampaignCanBeCompleted(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	saved := CurrentProfile
	defer func() { CurrentProfile = saved }()
	loadFixture(t)
	if got := eachAllegiance(); !slices.Equal(got, []string{ "crown", "free" }) {
		t.Fatalf("allegiances %v", got)
	}
	endings := map[string]string{ "crown": "crown_ending", "free": "free_ending" }
	for _, a := range eachAllegiance() {
		CurrentProfile = &Profile{}
		h := finishMainQuests(t, a)
		if h.Allegiance != a || h.CurrentAct() != 3 {
			t.Errorf("%s: allegiance %q, act %d", a, h.Allegiance, h.CurrentAct())
		}
		if !slices.Equal(CurrentProfile.Endings, []string{ endings[a] }) {
			t.Errorf("%s: endings on the profile %v", a, CurrentProfile.Endings)
		}
		// the lock sends away the companion who leans to the other society, and keeps the unaligned
		want := map[string][]string{ "crown": { "wren", "tam" }, "free": { "moss", "tam" } }[a]
		if !slices.Equal(h.Companions, want) {
			t.Errorf("%s: companions %v, want %v", a, h.Companions, want)
		}
	}
}

func TestStandingConditions(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	loadFixture(t)
	h := NewHero("Knight")
	h.Apply(&Step{ Standing: map[string]int{ "crown": 3, "free": -1 } })
	h.Apply(&Step{ Standing: map[string]int{ "crown": -1 } })
	for cond, want := range map[string]bool{
		"standing:crown>=2": true, "standing:crown>=3": false, "standing:crown<3": true,
		"standing:free<0": true, "standing:free=-1": true, "standing:crown>free": true,
		"standing:college=0": true, "!standing:crown<=1": true, "standing:free>=crown": false,
	} {
		if got := h.Meets([]string{ cond }); got != want {
			t.Errorf("%s = %v, want %v", cond, got, want)
		}
	}
}

func TestActsAndProfileConditions(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	saved := CurrentProfile
	defer func() { CurrentProfile = saved }()
	loadFixture(t)
	h := NewHero("Knight")
	if !h.Meets([]string{ "act:1", "act:<2", "returns:=0" }) || h.Meets([]string{ "act:2" }) {
		t.Fatalf("a new run is in act 1")
	}
	a := h.Apply(&Step{ Act: 2 })
	if a.Act == nil || a.Act.Title != "Act II" || a.Act.Subtitle != "The academy" || !h.Meets([]string{ "act:2", "act:=2", "act:<3" }) {
		t.Fatalf("act 2: %+v, hero in act %d", a.Act, h.Act)
	}
	if again := h.Apply(&Step{ Act: 2 }); again.Act != nil {
		t.Fatalf("staying in an act shows no title card")
	}
	CurrentProfile = &Profile{ Returns: 2 }
	if !h.Meets([]string{ "returns:2", "returns:>1", "!ending:free_ending" }) || h.Meets([]string{ "returns:3" }) {
		t.Fatalf("returns")
	}
	if a := h.Apply(&Step{ Ending: "free_ending" }); a.Ending != "free_ending" || !h.Finished || !h.Meets([]string{ "ending:free_ending" }) {
		t.Fatalf("ending not recorded: %+v %v", a, CurrentProfile)
	}
	h.Apply(&Step{ Ending: "free_ending" })
	if len(CurrentProfile.Endings) != 1 {
		t.Fatalf("an ending is recorded once: %v", CurrentProfile.Endings)
	}
}

func TestHoldersAndHostileSocietiesChangeASite(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	loadFixture(t)
	h := NewHero("Knight")
	town := LocationByID("town")
	if got := h.AsHeld(town); got.Desc != "A market town." || !slices.Equal(got.Foes, []string{ "rat" }) {
		t.Fatalf("no one holds the town: %+v", got)
	}
	if !h.Sells(town, "leather") || h.Sells(town, "steel_sword") {
		t.Fatalf("the town's own stock")
	}
	h.SetHolder("town", "crown")
	if got := h.AsHeld(town); got.Desc != "The Crown's flag flies over the market." || !slices.Equal(got.Foes, []string{ "rat" }) {
		t.Fatalf("the Crown holds the town: %+v", got)
	}
	if !h.Sells(town, "steel_sword") || h.Sells(town, "leather") || !h.Meets([]string{ "holder:town=crown", "holder:mill=" }) {
		t.Fatalf("the Crown's stock")
	}
	h.Join("free")
	if !h.Hostile("crown") || h.Hostile("free") {
		t.Fatalf("hostility after joining the Independents")
	}
	if got := h.AsHeld(town); got.Desc != "Crown guards bar the gate." || !slices.Equal(got.Foes, []string{ "crown_guard" }) || len(got.Shop) != 2 {
		t.Fatalf("a hostile Crown: %+v", got)
	}
	if town.Desc != "A market town." {
		t.Fatalf("AsHeld changed the location itself")
	}
	if h.Join("crown") || h.Allegiance != "free" {
		t.Fatalf("an allegiance is for the rest of the run")
	}
}

func TestCardPrecedence(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	loadFixture(t)
	q, f := QuestByID("founding"), FoeByID("engine")
	for _, c := range []struct{ step, quest, foe, want string }{
		{ "tall", q.Card, f.Card, "tall" },
		{ "", q.Card, f.Card, "wide" },
		{ "", "", f.Card, "tight" },
		{ "", "", "", "" },
	} {
		got := CardFor(c.step, c.quest, c.foe)
		if (got == nil && c.want != "") || (got != nil && got.ID != c.want) {
			t.Errorf("%+v: got %+v", c, got)
		}
	}
}

func TestCompanionsComeIntoFights(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	loadFixture(t)
	h := NewHero("Knight")
	base := NewHeroCombatant(h)
	if h.Bring("wren") {
		t.Fatalf("brought a companion not in the company")
	}
	h.Apply(&Step{ CompanionJoins: "wren" })
	h.Apply(&Step{ CompanionJoins: "moss" })
	h.Apply(&Step{ Favour: map[string]int{ "wren": 2 } })
	if !h.Meets([]string{ "companion:wren", "favour:wren>=2", "favour:moss=0" }) {
		t.Fatalf("company and favour: %v %v", h.Companions, h.Favour)
	}
	if c := NewHeroCombatant(h); len(c.Spells) != len(base.Spells) || len(c.Perks) != len(base.Perks) {
		t.Fatalf("no one was picked, so no one comes")
	}
	h.Bring("wren")
	c := NewHeroCombatant(h)
	if len(c.Spells) != len(base.Spells)+1 || c.Spells[len(c.Spells)-1].ID != "shield_bash" || c.Spells[len(c.Spells)-1].Cooldown != 9 {
		t.Fatalf("wren's rite, on her cooldown: %+v", c.Spells)
	}
	if SpellByID("shield_bash").Cooldown == 9 {
		t.Fatalf("the companion's cooldown changed the spell itself")
	}
	h.Bring("moss")
	c = NewHeroCombatant(h)
	if len(c.Perks) != len(base.Perks)+1 || c.Perks[len(c.Perks)-1].Color != Earth || PerkLabel(c.Perks[len(c.Perks)-1]) != "Moss" {
		t.Fatalf("moss's passive: %+v", c.Perks)
	}
	// the lock: Moss leans to the Independents and leaves, and so stops coming into fights
	h.Apply(&Step{ JoinSociety: "crown" })
	if h.Companion != "" || !slices.Equal(h.Companions, []string{ "wren" }) || h.FightCompanion() != nil {
		t.Fatalf("after the lock: %v, bringing %q", h.Companions, h.Companion)
	}
	if h.CompanionJoins("moss") {
		t.Fatalf("a companion of the other society joined after the lock")
	}
	h.Apply(&Step{ CompanionLeaves: "wren" })
	if len(h.Companions) != 0 {
		t.Fatalf("wren didn't leave")
	}
}

func TestPuzzleStepRoutesToScenes(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	loadFixture(t)
	pz := Scenes["opening"][1].Puzzle
	if pz == nil || pz.ID != "vault" || pz.Routes["force"] != "vault_forced" || pz.Lose != "vault_left" {
		t.Fatalf("puzzle step: %+v", pz)
	}
	h := NewHero("Knight")
	p := PuzzleByID("vault")
	r, _ := p.Replay(p.Routes[0].Solution)
	r.Finish(h)
	if !h.Flags["vault_alarm"] || h.Standing["crown"] != -1 {
		t.Fatalf("the force route's flag and standing: %v %v", h.Flags, h.Standing)
	}
	// the short form still loads
	if s := LoadStoryScene(t, "x:\n  - puzzle: vault\n"); s[0].Puzzle == nil || s[0].Puzzle.ID != "vault" || s[0].Puzzle.Routes != nil {
		t.Fatalf("short form: %+v", s[0].Puzzle)
	}
}

// LoadStoryScene parses one scenes file's YAML the way the loader does.
func LoadStoryScene(t *testing.T, src string) []Step {
	t.Helper()
	files := memFiles{
		"s/story.yaml": "start: a", "s/foes.yaml": "- { id: rat, name: Rat }",
		"s/locations.yaml": "- { id: a, name: A, open: true }", "s/quests.yaml": "[]",
		"s/scenes/x.yaml": src,
	}
	if _, err := LoadStory(files, "s"); err != nil {
		t.Fatal(err)
	}
	return Scenes["x"]
}

func TestStoryFormatMistakes(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	vault, err := os.ReadFile("../assets/story/puzzles/vault.yaml")
	if err != nil {
		t.Fatal(err)
	}
	files := memFiles{
		"s/story.yaml":      "start: a\nendings: [ { id: e }, { id: e } ]",
		"s/foes.yaml":       "- { id: rat, name: Rat }",
		"s/cards.yaml":      "- { id: c, name: C, desc: C }",
		"s/societies.yaml":  "- { id: crown, name: Crown, hostile: [\"hostile:crown\"] }\n- { id: academy }\n- { id: free, allegiance: true, hostile: [\"standing:nope>1\"] }",
		"s/companions.yaml": "- { id: x, society: guild }\n- { id: y, rite: bite }\n- { id: z, passive: { kind: mend, color: mauve, size: 3, amount: 1 } }",
		"s/locations.yaml": "- id: a\n  name: A\n  open: true\n  shop: [no_item]\n  variants:\n    - { holder: guild, foes: [bat], shop: [nor_this] }\n    - { if: [\"standing:crown\"] }",
		"s/quests.yaml":    "- { id: q, title: Q, giver: a, at: a, foe: rat, card: no_card, requires: [\"act:two\", \"favour:x>=a\", \"holder:a=guild\", \"ending:gone\", allegiance:guild] }",
		"s/puzzles/v.yaml": string(vault),
		"s/scenes/x.yaml": "x:\n" +
			"  - duel: { foe: rat, card: nope_card }\n" +
			"  - puzzle: { id: vault, routes: { smash: x, quiet: missing_scene } }\n" +
			"  - standing: { guild: 1 }\n" +
			"  - join_society: crown\n" +
			"  - ending: unwritten\n" +
			"  - favour: { nobody: 1 }\n" +
			"  - companion_joins: ghost\n" +
			"  - set_holder: { at: a, holder: guild }\n" +
			"  - act: -1\n",
	}
	problems, err := LoadStory(files, "s")
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(problems, "\n")
	for _, w := range []string{
		"ending 2 needs an id", "hostile conditions can't test hostile:", "\"academy\" is the player's own",
		"unknown society \"nope\"", "companion x: unknown society \"guild\"", "companion x: needs a rite or a passive",
		"rite \"bite\" should be a hero spell", "passive color \"mauve\"", "shop sells unknown item \"no_item\"",
		"unknown holder \"guild\"", "variant 1: unknown foe \"bat\"", "\"nor_this\"", "\"standing:crown\": should compare",
		"quest q: unknown card \"no_card\"", "\"act:two\"", "\"favour:x>=a\"", "\"holder:a=guild\": unknown society",
		"\"ending:gone\"", "\"allegiance:guild\"", "duel on unknown card \"nope_card\"", "puzzle vault has no route \"smash\"",
		"missing_scene", "standing with unknown society \"guild\"", "join_society \"crown\" should be a society with allegiance",
		"unknown ending \"unwritten\"", "favour with unknown companion \"nobody\"", "unknown companion \"ghost\"",
		"holder \"guild\" is not a society", "act -1",
	} {
		if !strings.Contains(all, w) {
			t.Errorf("no problem mentioning %q in:\n%s", w, all)
		}
	}
}

// A run saved at version 4 loads as it was, in the first act, and isn't widened again for Aether.
func TestVersion4RunLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hero.json")
	old := `{"Version":4,"Run":{"Class":"Warrior","Level":7,"Gold":40,"Stats":[4,1,2,2,0,7,2,3],` +
		`"Nodes":["fire_seat"],"Residue":[3,1,2,4,5,9],"Gear":{},"Spells":["cleave"],"Location":"emberfield","Active":{},` +
		`"Holders":{"barrow":"academy"}}}`
	os.WriteFile(path, []byte(old), 0o644)
	h, err := LoadRun(path)
	if err != nil || h == nil {
		t.Fatalf("version 4 run: %v", err)
	}
	if h.Stats != [NUM_STATS]int{ 4, 1, 2, 2, 0, 7, 2, 3 } || h.Residue != [NUM_RESIDUE]int{ 3, 1, 2, 4, 5, 9 } {
		t.Fatalf("arrays moved: stats %v, residue %v", h.Stats, h.Residue)
	}
	if h.Act != 1 || h.Allegiance != "" || len(h.Companions) != 0 || !h.Held("barrow") {
		t.Fatalf("migrated run: %+v", h)
	}
	if kept, _ := os.ReadFile(path + ".bak"); string(kept) != old {
		t.Fatalf("the version 4 run wasn't backed up")
	}
}
