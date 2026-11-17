package rpg

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"jewel-game/board"
)

// walkSteps calls see on every step of a list, inside branches and choices too.
func walkSteps(steps []Step, see func(*Step)) {
	for i := range steps {
		st := &steps[i]
		see(st)
		walkSteps(st.Then, see)
		walkSteps(st.Else, see)
		for _, c := range st.Choices {
			walkSteps(c.Do, see)
		}
	}
}

// actOneQuests is the main quests of Act 1, in order: up to and including the one whose done
// scene moves the story into act 2 (the founding quest).
func actOneQuests(t *testing.T) []*Quest {
	t.Helper()
	var out []*Quest
	for i := range Quests {
		q := &Quests[i]
		if !q.Main {
			continue
		}
		out = append(out, q)
		ends := false
		walkSteps(Scenes[q.DoneScene], func(st *Step) { ends = ends || st.Act == 2 })
		if ends {
			return out
		}
	}
	t.Fatal("no main quest's done scene moves the story into act 2")
	return nil
}

// solvableBy reports the route of a puzzle a hero of this class can take with only the tools
// their rites allow, replaying each route's stored solution as that hero.
func solvableBy(p *Puzzle, h *Hero) string {
	for _, r := range p.Routes {
		run, err := NewPuzzleRun(p, h)
		if err != nil {
			return ""
		}
		ok := true
		for _, m := range r.Solution {
			ok = ok && run.Play(m)
		}
		if ok && run.State == PuzzleWon && run.Route.ID == r.ID {
			return r.ID
		}
	}
	return ""
}

// TestActOneByEveryClass is the acceptance check for Act 1: an adept of each class, played by
// the AI, starts a fresh profile's first run (tutorial and first event), and walks Act 1's main
// quests in order through the data. Each quest must be offered where the adept can reach it,
// its offer scene must accept it, every kill is a simulated duel at the quest's level (a lost
// duel is fought again, keeping what the board paid, as a player would), and its done scene
// plays. Every puzzle board met on the way must have a route the class can finish with its own
// tools. At the end the academy is open, the story is in act 2, the founding lean is set and
// the three companions have joined.
func TestActOneByEveryClass(t *testing.T) {
	t.Parallel()	// it only reads the story, so it shares the CPU with the other simulator runs
	const MAX_TRIES = 30	// duels per kill before the adept counts as stuck
	quests := actOneQuests(t)
	for _, class := range Classes {
		t.Run(class.Name, func(t *testing.T) {
			h := NewHero(class.Name)
			saved := CurrentProfile
			CurrentProfile = &Profile{}
			defer func() { CurrentProfile = saved }()
			b := greedy(class.Name)
			seed := int64(0)
			duels, lost := 0, 0
			checkPuzzles := func(scene string) {
				walkSteps(Scenes[scene], func(st *Step) {
					if st.Puzzle == nil {
						return
					}
					if solvableBy(PuzzleByID(st.Puzzle.ID), h) == "" {
						t.Errorf("%s can't finish puzzle %s (in %s) by any route", class.Name, st.Puzzle.ID, scene)
					}
				})
			}
			for _, id := range RunScenes() {
				checkPuzzles(id)
				playScene(t, h, id, "")
			}
			for _, q := range quests {
				if h.Route(q.Giver) == nil && h.Location != q.Giver {
					t.Fatalf("%s: can't reach %s's giver %s", class.Name, q.ID, q.Giver)
				}
				if !slices.ContainsFunc(h.OfferedQuests(q.Giver), func(o *Quest) bool { return o.ID == q.ID }) {
					t.Fatalf("%s: %s isn't offered", class.Name, q.ID)
				}
				h.Location = q.Giver
				if q.OfferScene != "" {
					checkPuzzles(q.OfferScene)
					playScene(t, h, q.OfferScene, "")
				} else {
					h.Accept(q)
				}
				if _, ok := h.Active[q.ID]; !ok {
					t.Fatalf("%s: %s wasn't accepted", class.Name, q.ID)
				}
				if q.At != h.Location && h.Route(q.At) == nil {
					t.Fatalf("%s: %s: can't reach %s", class.Name, q.ID, q.At)
				}
				h.Location = q.At
				spendLikeAPlayer(h, q.At, b)
				card := RuleCardByID(q.Card)
				if card == nil {
					card = RuleCardByID(FoeByID(q.Foe).Card)
				}
				tries := 0
				for !slices.Contains(h.Done, q.ID) {
					if tries++; tries > MAX_TRIES*q.Count {
						t.Fatalf("%s is stuck on %s at level %d", class.Name, q.ID, h.Level)
					}
					seed++
					duels++
					d := NewDuel(NewHeroCombatant(h), NewFoeCombatant(FoeByID(q.Foe), q.FoeLevel), seed)
					d.SetBoard(board.NewBoard(BoardSettings(card, h.AetherOn()), seed))
					d.Card = card
					playOut(d)
					if d.Winner < 0 {
						t.Fatalf("%s vs %s never finished", class.Name, q.Foe)
					}
					if r := d.Settle(h); r.Won {
						h.RecordWin(q.At)
						for _, done := range h.RecordKill(q.Foe, q.At) {
							if done.DoneScene != "" {
								checkPuzzles(done.DoneScene)
								playScene(t, h, done.DoneScene, "")
							}
						}
					} else {
						lost++
					}
					spendLikeAPlayer(h, q.At, b)
				}
				t.Logf("%-8s %-12s level %d, %d duels so far, %d lost", class.Name, q.ID, h.Level, duels, lost)
			}
			if h.CurrentAct() != 2 || !h.AcademyOpen() {
				t.Errorf("%s: after Act 1, act %d, academy open %v", class.Name, h.CurrentAct(), h.AcademyOpen())
			}
			if !slices.ContainsFunc([]string{ "lean_crown", "lean_free", "lean_college", "lean_alone" }, func(f string) bool { return h.Flags[f] }) {
				t.Errorf("%s: the founding quest set no lean", class.Name)
			}
			for _, c := range []string{ "hart", "jory", "marlow" } {
				if !slices.Contains(h.Companions, c) {
					t.Errorf("%s: %s never joined (company %v)", class.Name, c, h.Companions)
				}
			}
			for _, s := range Societies {
				if h.Standing[s.ID] == 0 && s.ID != "college" {
					t.Errorf("%s: Act 1 left standing with %s untouched", class.Name, s.ID)
				}
			}
		})
	}
}

// Every society courts the adept in Act 1: each has a main quest whose done scene, or a puzzle
// route in it, moves its standing, and each companion who leans to a society joins in Act 1.
func TestEverySocietyCourtsInActOne(t *testing.T) {
	moved := map[string]bool{}
	joins := map[string]bool{}
	var see func(scene string, depth int)
	see = func(scene string, depth int) {
		if depth > 4 {
			return
		}
		walkSteps(Scenes[scene], func(st *Step) {
			for soc, n := range st.Standing {
				moved[soc] = moved[soc] || n > 0
			}
			if st.CompanionJoins != "" {
				joins[st.CompanionJoins] = true
			}
			if pz := st.Puzzle; pz != nil {
				for _, r := range PuzzleByID(pz.ID).Routes {
					for soc, n := range r.Standing {
						moved[soc] = moved[soc] || n > 0
					}
				}
				for _, s := range pz.Routes {
					see(s, depth+1)
				}
				see(pz.Lose, depth+1)
			}
			see(st.Goto, depth+1)
			see(st.Run, depth+1)
			for _, c := range st.Choices {
				see(c.Goto, depth+1)
			}
		})
	}
	for _, q := range actOneQuests(t) {
		see(q.OfferScene, 0)
		see(q.DoneScene, 0)
	}
	for _, s := range Societies {
		if !moved[s.ID] {
			t.Errorf("no Act 1 scene or route raises standing with %s", s.ID)
		}
	}
	for _, c := range Companions {
		if !joins[c.ID] {
			t.Errorf("companion %s doesn't join in Act 1", c.ID)
		}
	}
}

// The shipped story's references all exist, checked here from the data's side as well as by the
// loader (TestStoryIsValid): every quest's giver, target, foe, item, card, opened places and
// scenes; every location's roads, foes, people and arrival scene; every step's scene, foe,
// puzzle, quest, companion and society. And every scene is played from somewhere.
func TestShippedStoryReferencesExist(t *testing.T) {
	bad := func(format string, a ...any) { t.Errorf(format, a...) }
	scene := func(where, id string) {
		if _, ok := Scenes[id]; id != "" && !ok {
			bad("%s: no scene %q", where, id)
		}
	}
	place := func(where, id string) {
		if LocationByID(id) == nil {
			bad("%s: no location %q", where, id)
		}
	}
	foe := func(where, id string) {
		if FoeByID(id) == nil {
			bad("%s: no foe %q", where, id)
		}
	}
	for _, q := range Quests {
		where := "quest " + q.ID
		place(where, q.Giver)
		place(where, q.At)
		foe(where, q.Foe)
		for _, o := range q.Opens {
			place(where, o)
		}
		scene(where, q.OfferScene)
		scene(where, q.DoneScene)
		if q.Item != "" && ItemByID(q.Item) == nil {
			bad("%s: no item %q", where, q.Item)
		}
		if q.Card != "" && RuleCardByID(q.Card) == nil {
			bad("%s: no card %q", where, q.Card)
		}
	}
	for _, l := range Locations {
		where := "location " + l.ID
		for _, r := range l.Roads {
			place(where, r)
		}
		for _, f := range l.Foes {
			foe(where, f)
		}
		scene(where, l.EnterScene)
		for _, p := range l.People {
			scene(where, p.Scene)
		}
	}
	played := map[string]bool{ Story.Opening: true, Story.FirstEvent: true, Story.Tutorial: true, Story.ReturnScene: true }
	for _, q := range Quests {
		played[q.OfferScene], played[q.DoneScene] = true, true
	}
	for _, l := range Locations {
		played[l.EnterScene] = true
		for _, p := range l.People {
			played[p.Scene] = true
		}
	}
	for id, steps := range Scenes {
		where := "scene " + id
		walkSteps(steps, func(st *Step) {
			for _, s := range []string{ st.Goto, st.Run } {
				scene(where, s)
				played[s] = true
			}
			for _, c := range st.Choices {
				scene(where, c.Goto)
				played[c.Goto] = true
			}
			if d := st.Duel; d != nil {
				foe(where, d.Foe)
				for _, s := range []string{ d.Win, d.Lose } {
					scene(where, s)
					played[s] = true
				}
			}
			if pz := st.Puzzle; pz != nil {
				p := PuzzleByID(pz.ID)
				if p == nil {
					bad("%s: no puzzle %q", where, pz.ID)
				}
				for r, s := range pz.Routes {
					if p != nil && !slices.ContainsFunc(p.Routes, func(pr PuzzleRoute) bool { return pr.ID == r }) {
						bad("%s: puzzle %s has no route %q", where, pz.ID, r)
					}
					scene(where, s)
					played[s] = true
				}
				scene(where, pz.Lose)
				played[pz.Lose] = true
			}
			if st.AcceptQuest != "" && QuestByID(st.AcceptQuest) == nil {
				bad("%s: no quest %q", where, st.AcceptQuest)
			}
			for _, c := range []string{ st.CompanionJoins, st.CompanionLeaves } {
				if c != "" && CompanionByID(c) == nil {
					bad("%s: no companion %q", where, c)
				}
			}
			for c := range st.Favour {
				if CompanionByID(c) == nil {
					bad("%s: favour with no companion %q", where, c)
				}
			}
			for s := range st.Standing {
				if SocietyByID(s) == nil {
					bad("%s: standing with no society %q", where, s)
				}
			}
			if hs := st.SetHolder; hs != nil {
				place(where, hs.At)
			}
		})
	}
	for id := range Scenes {
		if !played[id] {
			bad("scene %s is never played", id)
		}
	}
}

// Story text has to fit a 420x860 phone: no scene line, choice or quest text runs past
// MAX_LINE characters, and choices stay short enough for one button.
func TestStoryTextFitsAPhone(t *testing.T) {
	const MAX_LINE, MAX_CHOICE = 200, 48
	check := func(where, s string, most int) {
		if n := len([]rune(Words(s))); n > most {
			t.Errorf("%s is %d characters, more than %d: %q", where, n, most, s)
		}
	}
	for id, steps := range Scenes {
		walkSteps(steps, func(st *Step) {
			check(fmt.Sprintf("scene %s", id), st.Text, MAX_LINE)
			for _, c := range st.Choices {
				check(fmt.Sprintf("scene %s choice", id), c.Text, MAX_CHOICE)
			}
		})
	}
	for _, q := range Quests {
		check("quest "+q.ID+" intro", q.Intro, MAX_LINE)
		check("quest "+q.ID+" outro", q.Outro, MAX_LINE)
	}
	for _, l := range Locations {
		check("location "+l.ID, l.Desc, MAX_LINE)
	}
	for _, p := range Puzzles {
		check("puzzle "+p.ID+" objective", p.Objective, MAX_LINE)
	}
}

// Story text may name the placeholders by key, and the player reads the names from the language
// file.
func TestStoryWordsFillPlaceholders(t *testing.T) {
	got := Words("{aether} and {entities}, paid in {currency}")
	if strings.Contains(got, "{") {
		t.Fatalf("placeholders left: %q", got)
	}
}
