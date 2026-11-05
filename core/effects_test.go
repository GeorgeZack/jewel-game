package core

import (
	"testing"

	"jewel-game/board"
)

func gemAt(x, y, color int) board.Cleared {
	return board.Cleared{Pos: board.Point{X: x, Y: y}, Gem: board.Gem{Color: color}}
}

func TestColorClusters(t *testing.T) {
	cleared := []board.Cleared{
		gemAt(0, 0, 0), gemAt(1, 0, 0), gemAt(2, 0, -1), gemAt(3, 0, 0),
		gemAt(0, 2, 0),
		gemAt(5, 5, 1), gemAt(6, 5, 1),
	}
	groups := colorClusters(cleared, 0)
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	if groups[0].weight != 3 || len(groups[0].cells) != 4 {
		t.Fatalf("wild should bridge the run: weight %d cells %d", groups[0].weight, len(groups[0].cells))
	}
	if groups[1].weight != 1 || len(groups[1].cells) != 1 {
		t.Fatalf("separated cell: weight %d cells %d", groups[1].weight, len(groups[1].cells))
	}
	diag := colorClusters([]board.Cleared{gemAt(0, 0, 0), gemAt(1, 1, 0)}, 0)
	if len(diag) != 2 {
		t.Fatalf("diagonal cells stay separate, got %d groups", len(diag))
	}
	if len(colorClusters(cleared, 2)) != 0 {
		t.Fatal("a color that was not cleared should have no clusters")
	}
	if len(colorClusters(cleared, -1)) != 0 {
		t.Fatal("wilds are not their own cluster")
	}
}

func TestSplitAmounts(t *testing.T) {
	sum := func(ns []int) int {
		s := 0
		for _, n := range ns {
			s += n
		}
		return s
	}
	got := splitAmounts(5, []int{1, 3})
	if sum(got) != 5 || got[0] != 1 || got[1] != 4 {
		t.Fatalf("5 across 1 and 3 = %v", got)
	}
	got = splitAmounts(10, []int{1, 1, 1})
	if sum(got) != 10 {
		t.Fatalf("10 across three = %v", got)
	}
	if len(splitAmounts(4, nil)) != 0 {
		t.Fatal("no weights")
	}
	if sum(splitAmounts(0, []int{2, 2})) != 0 {
		t.Fatal("nothing to share")
	}
	got = splitAmounts(3, []int{0, 0})
	if sum(got) != 3 || got[0] != 3 {
		t.Fatalf("zero weights = %v", got)
	}
}
