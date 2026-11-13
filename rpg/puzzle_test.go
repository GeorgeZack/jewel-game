package rpg

import (
	"fmt"
	"jewel-game/board"
	"strings"
	"testing"
)

// dumpBoard draws a puzzle board as text: color letters, upper case for marks, # for locks,
// * for specials and . for empty cells.
func dumpBoard(r *PuzzleRun) string {
	var sb strings.Builder
	for y := range r.Board.H() {
		for x := range r.Board.W() {
			p := board.Point{ X: x, Y: y }
			s := "."
			if g := r.Board.At(p); g != nil {
				s = string(PUZZLE_COLORS[g.Color])
				if r.MarkAt(p) != "" {
					s = strings.ToUpper(s)
				}
				if g.Locked {
					s += "#"
				} else if g.Special != board.None {
					s += "*"
				}
			}
			fmt.Fprintf(&sb, "%-3s", s)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// Every route of every puzzle is taken by its stored solution, and sets its flag.
func TestPuzzleRoutesReplay(t *testing.T) {
	if PuzzleByID("vault") == nil {
		t.Fatal("no vault puzzle")
	}
	for i := range Puzzles {
		pz := &Puzzles[i]
		for _, route := range pz.Routes {
			r, err := pz.Replay(route.Solution)
			if err != nil || r.State != PuzzleWon || r.Route.ID != route.ID {
				t.Fatalf("%s %s: %v, state %d, lost %q, route %+v\n%s", pz.ID, route.ID, err, r.State, r.Lost, r.Route, dumpBoard(r))
			}
			h := NewHero("Knight")
			r.Finish(h)
			if !h.Flags[route.Flag] {
				t.Fatalf("%s %s: flag %s not set", pz.ID, route.ID, route.Flag)
			}
			t.Logf("%s %-9s %d moves: %s", pz.ID, route.ID, len(route.Solution), strings.Join(route.Solution, "; "))
		}
	}
}

func TestPuzzleLosses(t *testing.T) {
	pz := *PuzzleByID("vault")
	quiet := []string{ "swap 1,2 1,3", "swap 2,3 3,3", "swap 5,2 5,3" }
	// finishing the goal by no route jams the lock (the vault's board leaves no way to clear its
	// pins out of order without a tool, so take the quiet route away)
	pz.Routes = pz.Routes[:2]
	r, err := pz.Replay(quiet)
	if err != nil || r.State != PuzzleLost || r.Lost != "jammed" {
		t.Fatalf("no route: %v %d %q", err, r.State, r.Lost)
	}
	// running out of moves, and a tool use counts as one
	pz = *PuzzleByID("vault")
	pz.Moves = 2
	r, err = pz.Replay([]string{ "tool lockpick 3,3", "swap 1,2 1,3" })
	if err != nil || r.State != PuzzleLost || r.Lost != "moves" || r.Moves() != 2 {
		t.Fatalf("out of moves: %v %d %q", err, r.State, r.Lost)
	}
	if r.Play("swap 5,2 5,3") {
		t.Fatalf("played on after losing")
	}
	// a fail condition
	pz = *PuzzleByID("vault")
	pz.Fail = []PuzzleCond{ { Cleared: []string{ "a" } } }
	if r, _ := pz.Replay([]string{ "swap 1,2 1,3" }); r.State != PuzzleLost || r.Lost != "fail" {
		t.Fatalf("fail condition: %d %q", r.State, r.Lost)
	}
	// retrying is free: a new run starts from the authored board
	again, _ := NewPuzzleRun(PuzzleByID("vault"), nil)
	fresh, _ := PuzzleByID("vault").Replay(nil)
	if dumpBoard(again) != dumpBoard(fresh) || again.State != PuzzlePlaying {
		t.Fatalf("a retry isn't the authored board")
	}
}

// The same moves always leave the same board, refilled or not.
func TestPuzzleIsDeterministic(t *testing.T) {
	for _, refill := range []string{ "none", "queue", "seed" } {
		pz := *PuzzleByID("vault")
		pz.Refill, pz.Queue, pz.Seed = refill, "rbygpybgrpgybrpyg", 7
		moves := pz.Routes[len(pz.Routes)-1].Solution[:1]	// later moves may need the board unrefilled
		a, err1 := pz.Replay(moves)
		b, err2 := pz.Replay(moves)
		if err1 != nil || err2 != nil || dumpBoard(a) != dumpBoard(b) {
			t.Fatalf("%s refill: two replays differ (%v %v)\n%s\n%s", refill, err1, err2, dumpBoard(a), dumpBoard(b))
		}
		empty := strings.Count(dumpBoard(a), ".")
		if (refill == "none") != (empty > 0) {
			t.Fatalf("%s refill left %d empty cells:\n%s", refill, empty, dumpBoard(a))
		}
	}
}

func TestPuzzleToolsNeedRites(t *testing.T) {
	pz := PuzzleByID("vault")
	knight, _ := NewPuzzleRun(pz, NewHero("Knight"))
	if knight.Uses["force"] != pz.Tools[0].Uses || knight.Uses["lockpick"] != 0 {
		t.Fatalf("knight tools: %v", knight.Uses)
	}
	if knight.Play("tool lockpick 3,3") {
		t.Fatalf("the knight picked the lock without a rite for it")
	}
	druid, _ := NewPuzzleRun(pz, NewHero("Druid"))
	for druid.Hero.Level < 4 {
		druid.Hero.AddXP(XPToNext(druid.Hero.Level))
	}
	druid, _ = NewPuzzleRun(pz, druid.Hero)
	// the lockpick paints one gem: it only sets one pin, and only on a gem of its color
	if druid.Play("tool lockpick 0,0") || !druid.Play("tool lockpick 3,3") || druid.State != PuzzlePlaying || druid.Uses["lockpick"] != 0 || druid.Moves() != 1 {
		t.Fatalf("druid lockpick: %+v", druid)
	}
	for _, m := range []string{ "swap 1,2 1,3", "swap 5,2 5,3" } {
		druid.Play(m)
	}
	if druid.Route == nil || druid.Route.ID != "lockpick" {
		t.Fatalf("druid lockpick route: %+v", druid)
	}
	// Forcing the bomb that blasts two pins is forcing the lock, for a knight too
	r, _ := NewPuzzleRun(pz, NewHero("Knight"))
	for _, m := range []string{ "tool force 2,4", "swap 5,2 5,3" } {
		r.Play(m)
	}
	if r.State != PuzzleWon || r.Route.ID != "force" {
		t.Fatalf("force tool route: %d %+v %q\n%s", r.State, r.Route, r.Lost, dumpBoard(r))
	}
}

func TestPuzzleValidation(t *testing.T) {
	defer LoadStory(diskFiles{}, "../assets/story")
	board := `["r g b y", "g b y r", "b y r g", "y r g b"]`
	files := memFiles{
		"s/story.yaml":     "start: a",
		"s/foes.yaml":      "- { id: rat, name: Rat }",
		"s/locations.yaml": "- { id: a, name: A, open: true }",
		"s/quests.yaml":    "[]",
		"s/scenes/a.yaml":  "intro:\n  - puzzle: nowhere\n",
		"s/puzzles/a.yaml": "id: stale\nmoves: 3\nboard: " + board + "\nmarks: { a: [0, 0] }\ngoal: { cleared: [a] }\n" +
			"routes:\n  - { id: only, solution: [\"swap 0,0 1,0\"] }\n",
		"s/puzzles/b.yaml": "id: odd\nmoves: 0\nrefill: sometimes\nboard: [\"r g\", \"g\"]\nmarks: { z: [9, 9] }\ngoal: { cleared: [q] }\n" +
			"tools: [{ id: t, kind: shove, rites: [nope] }]\n",
	}
	problems, err := LoadStory(files, "s")
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(problems, "\n")
	for _, w := range []string{ "unknown puzzle \"nowhere\"", "route only: stored solution fails", "puzzle odd: row 1 has" } {
		if !strings.Contains(all, w) {
			t.Errorf("no problem mentioning %q in:\n%s", w, all)
		}
	}
}
