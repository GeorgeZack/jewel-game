package board

import (
	"strings"
	"testing"
)

// fromRows builds a board from strings: digits are colors, '.' is a random filler
// that can't match anything nearby, a trailing '*' marks the previous gem locked,
// and P/H/V/B place a Prism, horizontal Bolt, vertical Bolt, or Bomb (color 0).
func fromRows(t *testing.T, s Settings, rows ...string) *Board {
	t.Helper()
	s.Height = len(rows)
	s.Width = 0
	parsed := make([][]*Gem, len(rows))
	nextFiller := 3
	b := NewBoard(s, 1)
	for y, row := range rows {
		for i := 0; i < len(row); i++ {
			ch := row[i]
			if ch == '*' {
				parsed[y][len(parsed[y])-1].Locked = true
				continue
			}
			g := &Gem{}
			switch {
			case ch >= '0' && ch <= '9':
				g.Color = int(ch - '0')
			case ch == '.':
				// cycle through colors that the tests don't use so fillers never match
				g.Color = nextFiller
				nextFiller++
				if nextFiller >= s.NumColors {
					nextFiller = 3
				}
			case ch == 'P':
				g.Color, g.Special = -1, Prism
			case ch == 'H':
				g.Special = BoltH
			case ch == 'V':
				g.Special = BoltV
			case ch == 'B':
				g.Special = Bomb
			default:
				t.Fatalf("bad cell %q", ch)
			}
			parsed[y] = append(parsed[y], g)
		}
	}
	b.Settings.Width = len(parsed[0])
	b.Settings.Height = len(parsed)
	b.Cells = parsed
	for y := range parsed {
		if len(parsed[y]) != b.W() {
			t.Fatalf("row %d has %d cells, want %d", y, len(parsed[y]), b.W())
		}
		for _, g := range parsed[y] {
			b.nextID++
			g.ID = b.nextID
		}
	}
	return b
}

func testSettings() Settings {
	s := DefaultSettings()
	s.NumColors = 16
	s.MaxLocks = -1
	return s
}

func (b *Board) String() string {
	var sb strings.Builder
	for y := 0; y < b.H(); y++ {
		for x := 0; x < b.W(); x++ {
			g := b.Cells[y][x]
			switch {
			case g == nil:
				sb.WriteByte('_')
			case g.Special == Prism:
				sb.WriteByte('P')
			case g.Special == Bomb:
				sb.WriteByte('B')
			case g.Special.IsBolt():
				sb.WriteByte('|')
			default:
				sb.WriteByte(byte('a' + g.Color))
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func countSpecial(b *Board, s Special) int {
	n := 0
	b.each(func(_ Point, g *Gem) {
		if g.Special == s {
			n++
		}
	})
	return n
}

func TestNewBoardHasNoMatchesAndAMove(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		b := NewBoard(DefaultSettings(), seed)
		if len(b.findRuns()) != 0 {
			t.Fatalf("seed %d: new board starts with a match\n%s", seed, b)
		}
		if !b.HasMove() {
			t.Fatalf("seed %d: new board has no move\n%s", seed, b)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name	string
		rows	[]string
		want	Special
	}{
		{ "three", []string{ "000..", "....." }, None },
		{ "four", []string{ "0000.", "....." }, BoltH },
		{ "four vertical", []string{ "0.", "0.", "0.", "0.", ".." }, BoltV },
		{ "five", []string{ "00000", "....." }, Prism },
		{ "L shape", []string{ "0....", "0....", "000.." }, Bomb },
		{ "T shape", []string{ "000..", ".0...", ".0..." }, Bomb },
	}
	for _, c := range cases {
		b := fromRows(t, testSettings(), c.rows...)
		groups := b.findGroups()
		if len(groups) != 1 {
			t.Fatalf("%s: got %d groups, want 1", c.name, len(groups))
		}
		if got := groups[0].classify(); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestLockedGemsDontMatchOrBreak(t *testing.T) {
	b := fromRows(t, testSettings(),
		"00*0..",
		".....",
		"H....",
	)
	if b.HasMatches() {
		t.Fatal("a locked gem completed a match")
	}
	b.Cells[0][4].Special = BoltH
	b.force(Point{ 4, 0 })
	b.Step()
	if g := b.At(Point{ 1, 0 }); g == nil || !g.Locked {
		t.Fatalf("locked gem was destroyed by a bolt\n%s", b)
	}
}

func TestCollapseKeepsLockedGemsInPlace(t *testing.T) {
	for d := Down; d <= Right; d++ {
		b := fromRows(t, testSettings(),
			"1...",
			"22*..",
			"....",
			"....",
		)
		b.Settings.Gravity = d
		lockedID := b.At(Point{ 1, 1 }).ID
		b.set(Point{ 0, 0 }, nil)
		b.set(Point{ 1, 0 }, nil)
		b.set(Point{ 1, 2 }, nil)
		spawns := b.Collapse()
		if len(spawns) != 3 {
			t.Errorf("%v: got %d spawns, want 3", d, len(spawns))
		}
		for y := 0; y < b.H(); y++ {
			for x := 0; x < b.W(); x++ {
				if b.Cells[y][x] == nil {
					t.Fatalf("%v: hole left at %d,%d\n%s", d, x, y, b)
				}
			}
		}
		if g := b.At(Point{ 1, 1 }); g.ID != lockedID {
			t.Errorf("%v: locked gem moved\n%s", d, b)
		}
		for id, p := range spawns {
			if b.In(p) {
				t.Errorf("%v: gem %d spawned on the board at %v", d, id, p)
			}
		}
	}
}

// The motivating setup: lock most of two five-lines, finish them, then release both at once.
func TestReleasingLocksFiresTwoPrismsTogether(t *testing.T) {
	b := fromRows(t, testSettings(),
		"0*0*.0*0*",
		"..0..",
		"1*1*.1*1*",
		"..1..",
	)
	if b.HasMatches() {
		t.Fatalf("setup matched early\n%s", b)
	}
	// slot the last gem of each line into place; neither swap matches while the rest are locked
	b.Settings.FreeSwap = true
	if b.CanSwap(Point{ 1, 0 }, Point{ 1, 1 }) {
		t.Fatal("locked gems shouldn't swap")
	}
	b.Swap(Point{ 2, 1 }, Point{ 2, 0 })
	b.Swap(Point{ 2, 3 }, Point{ 2, 2 })
	if b.HasMatches() {
		t.Fatalf("matched before release\n%s", b)
	}
	if n := b.ReleaseAll(); n != 8 {
		t.Fatalf("released %d, want 8", n)
	}
	b.StartChain()
	res := b.Step()
	if res == nil || len(res.Created) != 2 {
		t.Fatalf("want two specials from one step, got %+v\n%s", res, b)
	}
	if countSpecial(b, Prism) != 2 {
		t.Fatalf("want 2 prisms\n%s", b)
	}
	if res.Points < 400 {
		t.Errorf("double special bonus missing: %d points", res.Points)
	}
}

func TestPrismSwapClearsColor(t *testing.T) {
	b := fromRows(t, testSettings(),
		"P0.0.",
		"..0..",
		"0...0",
	)
	if !b.IsValidSwap(Point{ 0, 0 }, Point{ 1, 0 }) {
		t.Fatal("prism swap should always be valid")
	}
	b.Swap(Point{ 0, 0 }, Point{ 1, 0 })
	b.BeginMove(Point{ 0, 0 }, Point{ 1, 0 })
	res := b.Step()
	if len(res.Cleared) != 6 {
		t.Fatalf("cleared %d, want 6 (5 gems + prism)\n%s", len(res.Cleared), b)
	}
	b.each(func(p Point, g *Gem) {
		if g.Color == 0 {
			t.Errorf("color 0 survived at %v", p)
		}
	})
}

func TestPrismPlusBombSpreadsBombs(t *testing.T) {
	s := testSettings()
	b := fromRows(t, s,
		"PB.......",
		".........",
		".........",
		"....0....",
		".........",
	)
	b.Swap(Point{ 0, 0 }, Point{ 1, 0 })
	b.BeginMove(Point{ 0, 0 }, Point{ 1, 0 })
	res := b.Step()
	if len(res.Blasts) < 2 {
		t.Fatalf("want the swapped bomb and the converted gem to blow, got %d blasts", len(res.Blasts))
	}
	if b.At(Point{ 3, 2 }) != nil {
		t.Errorf("converted bomb at 4,3 should clear its neighborhood\n%s", b)
	}
}

func TestBoltPlusBoltClearsCross(t *testing.T) {
	b := fromRows(t, testSettings(),
		".....",
		".....",
		".HV..",
		".....",
	)
	b.Swap(Point{ 1, 2 }, Point{ 2, 2 })
	b.BeginMove(Point{ 1, 2 }, Point{ 2, 2 })
	b.Step()
	for x := 0; x < b.W(); x++ {
		if b.At(Point{ x, 2 }) != nil {
			t.Fatalf("row not cleared\n%s", b)
		}
	}
	for y := 0; y < b.H(); y++ {
		if b.At(Point{ 2, y }) != nil {
			t.Fatalf("column not cleared\n%s", b)
		}
	}
}

func TestChainReactionThroughSpecials(t *testing.T) {
	b := fromRows(t, testSettings(),
		"000H.",
		"....V",
		".....",
	)
	res := b.Step()
	if len(res.Blasts) != 1 {
		t.Fatalf("want 1 blast, got %d", len(res.Blasts))
	}
	b = fromRows(t, testSettings(),
		"000HV",
		".....",
		".....",
	)
	res = b.Step()
	if len(res.Blasts) != 2 {
		t.Fatalf("want bolt to set off the other bolt, got %d blasts", len(res.Blasts))
	}
	for y := 0; y < b.H(); y++ {
		if b.At(Point{ 4, y }) != nil {
			t.Fatalf("chained column not cleared\n%s", b)
		}
	}
}

func TestShuffleFindsAMove(t *testing.T) {
	s := testSettings()
	s.NumColors = 4
	b := fromRows(t, s,
		"0123",
		"2301",
		"0123",
		"2301",
	)
	if b.HasMove() {
		t.Skip("fixture accidentally has a move")
	}
	if !b.Shuffle() {
		t.Fatal("shuffle said it wasn't needed")
	}
	if !b.HasMove() {
		t.Fatalf("still no move after shuffle\n%s", b)
	}
}

func TestUndoRestores(t *testing.T) {
	b := NewBoard(DefaultSettings(), 3)
	snap := b.Snapshot()
	before := b.String()
	a, c, _ := b.FindMove()
	b.Swap(a, c)
	b.BeginMove(a, c)
	for b.Step() != nil {
		b.Collapse()
	}
	if !b.Restore(snap) || b.String() != before || b.Moves != 0 || b.Score != 0 {
		t.Fatalf("restore mismatch")
	}
}

// Play lots of random moves under wild settings to shake out panics and stuck loops.
func TestRandomPlayIsStable(t *testing.T) {
	s := DefaultSettings()
	s.SpecialRate = 0.3
	s.Overcharge = true
	s.ChaosGravity = true
	s.DiagonalSwap = true
	s.MaxLocks = 4
	for seed := int64(0); seed < 20; seed++ {
		b := NewBoard(s, seed)
		for move := 0; move < 60; move++ {
			if move%7 == 0 {
				b.ToggleLock(Point{ b.rng.Intn(b.W()), b.rng.Intn(b.H()) })
			}
			a, c, ok := b.FindMove()
			if !ok {
				b.Shuffle()
				continue
			}
			b.Swap(a, c)
			b.BeginMove(a, c)
			for steps := 0; b.Step() != nil; steps++ {
				if steps > 500 {
					t.Fatalf("seed %d: cascade never ended", seed)
				}
				b.Collapse()
			}
			for y := range b.Cells {
				for x := range b.Cells[y] {
					if b.Cells[y][x] == nil {
						t.Fatalf("seed %d: hole at %d,%d", seed, x, y)
					}
				}
			}
		}
	}
}

func duelBoard(rows ...[]int) *Board {
	b := NewBoard(Settings{ Width: len(rows[0]), Height: len(rows), NumColors: 7, Duel: true, WildColors: 4, BombColor: 4 }, 1)
	for y, row := range rows {
		for x, c := range row {
			b.Cells[y][x] = b.newGem(c, None)
			if c == -1 {
				b.PlaceWild(Point{ x, y }, 3)
			}
		}
	}
	return b
}

func TestWildCompletesManaRunAndMultiplies(t *testing.T) {
	b := duelBoard(
		[]int{ 0, 0, -1, 6, 5 },
		[]int{ 1, 2, 3, 4, 5 },
		[]int{ 2, 3, 1, 5, 4 },
		[]int{ 3, 1, 2, 4, 6 },
	)
	res := b.Step()
	if res == nil || len(res.Matches) != 1 {
		t.Fatalf("expected one match, got %+v", res)
	}
	if m := res.Matches[0]; m.Color != 0 || m.Size != 3 || m.Mult != 3 || m.Wilds != 1 {
		t.Fatalf("bad match %+v", m)
	}
}

func TestWildDoesNotCompleteSkulls(t *testing.T) {
	b := duelBoard(
		[]int{ 4, 4, -1, 6, 5 },
		[]int{ 1, 2, 3, 4, 5 },
		[]int{ 2, 3, 1, 5, 4 },
		[]int{ 3, 1, 2, 4, 6 },
	)
	if b.HasMatches() {
		t.Fatalf("a Wild must not extend a skull run")
	}
}

func TestWildJoinsBothNeighbours(t *testing.T) {
	b := duelBoard(
		[]int{ 0, 0, -1, 1, 1 },
		[]int{ 1, 2, 3, 4, 5 },
		[]int{ 2, 3, 1, 5, 4 },
		[]int{ 3, 1, 2, 4, 6 },
	)
	if res := b.Step(); res == nil || len(res.Matches) != 2 || len(res.Cleared) != 5 {
		t.Fatalf("expected both sides to match: %+v", res)
	}
}

func TestDuelRunOfFiveEarnsWildAndFourNothing(t *testing.T) {
	b := duelBoard(
		[]int{ 0, 0, 0, 0, 0 },
		[]int{ 1, 2, 3, 4, 5 },
		[]int{ 2, 3, 1, 5, 4 },
	)
	res := b.Step()
	if len(res.Created) != 1 || b.At(res.Created[0]).Special != Wild || b.At(res.Created[0]).Mult != WILD_MULT {
		t.Fatalf("run of 5 should earn a Wild: %+v", res)
	}
	b = duelBoard(
		[]int{ 0, 0, 0, 0, 5 },
		[]int{ 1, 2, 3, 4, 6 },
		[]int{ 2, 3, 1, 5, 4 },
	)
	if res := b.Step(); len(res.Created) != 0 {
		t.Fatalf("run of 4 earns no special in a duel: %+v", res)
	}
}

func TestDuelSwapNeedsAMatchEvenWithWilds(t *testing.T) {
	b := duelBoard(
		[]int{ -1, 1, 2, 3 },
		[]int{ -1, 4, 5, 6 },
		[]int{ 6, 5, 4, 3 },
	)
	if b.IsValidSwap(Point{ 0, 0 }, Point{ 0, 1 }) {
		t.Fatalf("swapping two Wilds isn't a move in a duel")
	}
}

func TestBombClearsAreaInDuel(t *testing.T) {
	b := duelBoard(
		[]int{ 4, 4, 4, 1, 2 },
		[]int{ 1, 2, 3, 6, 5 },
		[]int{ 2, 3, 1, 5, 6 },
	)
	b.At(Point{ 1, 0 }).Special = Bomb
	if res := b.Step(); len(res.Blasts) != 1 || len(res.Cleared) != 6 {
		t.Fatalf("bomb should clear 3x2 at the edge: %+v", res)
	}
}

// A missing color never arrives: not on the first fill, a refill, a reroll or a shuffle.
func TestMissingColorNeverArrives(t *testing.T) {
	s := DefaultSettings()
	s.NumColors, s.Missing = 8, 1<<4
	for seed := range int64(20) {
		b := NewBoard(s, seed)
		for range 30 {
			if a, c, ok := b.FindMove(); ok {
				b.Swap(a, c)
				b.BeginMove(a, c)
			}
			for b.Step() != nil {
				b.Collapse()
			}
			b.Shuffle()
			b.each(func(p Point, g *Gem) {
				if g.Color == 4 {
					t.Fatalf("seed %d: color 4 at %v", seed, p)
				}
			})
		}
	}
}
