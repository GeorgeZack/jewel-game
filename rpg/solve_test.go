package rpg

import (
	"jewel-game/board"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestSolvePuzzle is a dev tool, skipped unless SOLVE names a puzzle:
//
//	SOLVE=vault SOLVE_DEPTH=4 go test ./rpg -run TestSolvePuzzle -v
//
// It tries every sequence of swaps and tool uses up to the depth, and prints how many end on each
// route or loss, with the shortest sequence for each.
func TestSolvePuzzle(t *testing.T) {
	pz := PuzzleByID(os.Getenv("SOLVE"))
	if pz == nil {
		t.Skip("set SOLVE to a puzzle id")
	}
	depth := 4
	if n, err := strconv.Atoi(os.Getenv("SOLVE_DEPTH")); err == nil {
		depth = n
	}
	counts := map[string]int{}
	shortest := map[string][]string{}
	var walk func(moves []string)
	walk = func(moves []string) {
		r, err := pz.Replay(moves)
		if err != nil {
			return
		}
		if r.State != PuzzlePlaying {
			key := "lost: " + r.Lost
			if r.State == PuzzleWon {
				key = "route: " + r.Route.ID
			}
			counts[key]++
			if old, ok := shortest[key]; !ok || len(moves) < len(old) {
				shortest[key] = append([]string{}, moves...)
			}
			return
		}
		if len(moves) >= depth {
			counts["unfinished"]++
			return
		}
		// only moves the board allows are tried, so whole subtrees aren't replayed for nothing
		b := r.Board
		for y := range b.H() {
			for x := range b.W() {
				a := board.Point{ X: x, Y: y }
				for _, c := range []board.Point{ { X: x + 1, Y: y }, { X: x, Y: y + 1 } } {
					if b.In(c) && b.IsValidSwap(a, c) {
						walk(append(moves, fmt.Sprintf("swap %d,%d %d,%d", x, y, c.X, c.Y)))
					}
				}
			}
		}
		for _, tl := range pz.Tools {
			if r.Uses[tl.ID] <= 0 {
				continue
			}
			if !tl.Targeted() {
				walk(append(moves, "tool "+tl.ID))
				continue
			}
			for y := range b.H() {
				for x := range b.W() {
					g := b.At(board.Point{ X: x, Y: y })
					if g == nil || g.Locked || (tl.Kind == "paint" && g.Color != puzzleColor(tl.From)) {
						continue
					}
					walk(append(moves, fmt.Sprintf("tool %s %d,%d", tl.ID, x, y)))
				}
			}
		}
	}
	walk(nil)
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("%s, up to %d moves:", pz.ID, depth)
	for _, k := range keys {
		t.Logf("  %-16s %6d  shortest: %s", k, counts[k], strings.Join(shortest[k], "; "))
	}
}
