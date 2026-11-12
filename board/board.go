// Package board holds the pure game rules: no rendering, no input.
// The core package drives it one step at a time so it can animate between steps.
package board

import "math/rand"

type Point struct {
	X, Y int
}

func (p Point) Add(q Point) Point { return Point{ p.X + q.X, p.Y + q.Y } }

type Special uint8

const (
	None Special = iota
	BoltH		// clears its row when detonated
	BoltV		// clears its column when detonated
	Bomb		// clears the surrounding square
	Prism		// colorless; swap with a gem to clear that whole color
	Wild		// duel rules: colorless, completes a run of any mana color and multiplies it
)

func (s Special) IsBolt() bool { return s == BoltH || s == BoltV }

func (s Special) String() string {
	switch s {
	case BoltH, BoltV:
		return "Bolt"
	case Bomb:
		return "Bomb"
	case Prism:
		return "Prism"
	case Wild:
		return "Wild"
	}
	return ""
}

// Direction is the way gems fall.
type Direction uint8

const (
	Down Direction = iota
	Left
	Up
	Right
)

func (d Direction) Vec() Point {
	switch d {
	case Left:
		return Point{ -1, 0 }
	case Up:
		return Point{ 0, -1 }
	case Right:
		return Point{ 1, 0 }
	}
	return Point{ 0, 1 }
}

func (d Direction) String() string {
	return [...]string{ "Down", "Left", "Up", "Right" }[d%4]
}

type Gem struct {
	ID		int
	Color	int		// -1 for Prism
	Special	Special
	Locked	bool	// locked gems never move, match, or get destroyed
	Mult	int		// a Wild's multiplier
}

type Settings struct {
	Width, Height	int
	NumColors		int
	MaxLocks		int		// simultaneous locks allowed, -1 = unlimited
	FreeSwap		bool	// swaps don't need to make a match
	DiagonalSwap	bool
	Gravity			Direction
	ChaosGravity	bool	// gravity turns randomly after every move
	SpecialRate		float64	// chance a refilled gem arrives as a special
	Overcharge		bool	// bigger bombs, wider bolts
	Duel			bool	// no earned specials or combos; a run of 5 earns a Wild instead
	WildColors		int		// duel rules: colors below this can be completed by a Wild
	BombColor		int		// duel rules: refills of this color may arrive as a Bomb
	BombRate		float64
}

func DefaultSettings() Settings {
	return Settings{
		Width:		8,
		Height:		8,
		NumColors:	7,
		MaxLocks:	3,
	}
}

func (s *Settings) clamp() {
	s.Width = clampInt(s.Width, 4, 16)
	s.Height = clampInt(s.Height, 4, 16)
	s.NumColors = clampInt(s.NumColors, 3, 16)
	if s.MaxLocks < -1 {
		s.MaxLocks = -1
	}
	if s.SpecialRate < 0 {
		s.SpecialRate = 0
	}
}

type Board struct {
	Settings	Settings
	Cells		[][]*Gem	// [y][x], nil only mid-resolve
	Score		int
	Chain		int
	BestChain	int
	Moves		int
	Cleared		int
	rng			*rand.Rand
	nextID		int
	swapped		[]Point		// where the last swap landed, preferred spot for new specials
	pending		[]Point		// cells forced to clear by a special combo

	// authored boards: refills come from Queue (colors, first first) while it lasts, then
	// from the seeded random source unless NoRefill leaves the cells empty
	Queue		[]int
	NoRefill	bool
}

func NewBoard(s Settings, seed int64) *Board {
	s.clamp()
	b := &Board{
		Settings:	s,
		rng:		rand.New(rand.NewSource(seed)),
	}
	b.Cells = make([][]*Gem, s.Height)
	for y := range b.Cells {
		b.Cells[y] = make([]*Gem, s.Width)
	}
	b.fill()
	return b
}

// NewBoardFrom builds an authored board: cells[y][x] gives each gem's color, special and lock;
// nil leaves a cell empty. The seed drives everything random afterwards, so the same cells and
// seed play out the same way.
func NewBoardFrom(s Settings, seed int64, cells [][]*Gem) *Board {
	s.Width, s.Height = len(cells[0]), len(cells)
	s.clamp()
	b := &Board{
		Settings:	s,
		rng:		rand.New(rand.NewSource(seed)),
	}
	b.Cells = make([][]*Gem, s.Height)
	for y := range b.Cells {
		b.Cells[y] = make([]*Gem, s.Width)
		for x, g := range cells[y] {
			if g != nil {
				n := b.newGem(g.Color, g.Special)
				n.Locked, n.Mult = g.Locked, g.Mult
				b.Cells[y][x] = n
			}
		}
	}
	return b
}

// refillGem is the next gem to drop into an emptied cell, or nil to leave it empty.
func (b *Board) refillGem() *Gem {
	if len(b.Queue) > 0 {
		c := b.Queue[0]
		b.Queue = b.Queue[1:]
		return b.newGem(c, None)
	}
	if b.NoRefill {
		return nil
	}
	return b.randomGem()
}

func (b *Board) W() int { return b.Settings.Width }
func (b *Board) H() int { return b.Settings.Height }

func (b *Board) In(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < b.W() && p.Y < b.H()
}

func (b *Board) At(p Point) *Gem {
	if !b.In(p) {
		return nil
	}
	return b.Cells[p.Y][p.X]
}

func (b *Board) set(p Point, g *Gem) { b.Cells[p.Y][p.X] = g }

func (b *Board) newGem(color int, special Special) *Gem {
	b.nextID++
	if special == Prism || special == Wild {
		color = -1
	}
	return &Gem{ ID: b.nextID, Color: color, Special: special }
}

func (b *Board) randomGem() *Gem {
	special := None
	if b.rng.Float64() < b.Settings.SpecialRate {
		switch r := b.rng.Intn(10); {
		case r < 3:
			special = BoltH
		case r < 6:
			special = BoltV
		case r < 9:
			special = Bomb
		default:
			special = Prism
		}
	}
	color := b.rng.Intn(b.Settings.NumColors)
	if b.Settings.Duel && color == b.Settings.BombColor && b.rng.Float64() < b.Settings.BombRate {
		special = Bomb
	}
	return b.newGem(color, special)
}

// fill builds a fresh board with no ready-made matches and at least one move.
func (b *Board) fill() {
	for attempt := 0; attempt < 100; attempt++ {
		for y := 0; y < b.H(); y++ {
			for x := 0; x < b.W(); x++ {
				b.Cells[y][x] = b.newGem(b.safeColor(x, y), None)
			}
		}
		if b.HasMove() {
			return
		}
	}
}

// safeColor picks a color that doesn't complete a run with the two gems left or above.
func (b *Board) safeColor(x, y int) int {
	n := b.Settings.NumColors
	start := b.rng.Intn(n)
	for i := 0; i < n; i++ {
		c := (start + i) % n
		if x >= 2 && b.colorAt(x-1, y) == c && b.colorAt(x-2, y) == c {
			continue
		}
		if y >= 2 && b.colorAt(x, y-1) == c && b.colorAt(x, y-2) == c {
			continue
		}
		return c
	}
	return start
}

func (b *Board) colorAt(x, y int) int {
	if g := b.Cells[y][x]; g != nil {
		return g.Color
	}
	return -1
}

// matchable gems have a real color (so not prisms) and aren't locked.
func matchable(g *Gem) bool {
	return g != nil && !g.Locked && g.Color >= 0
}

func (b *Board) Adjacent(a, c Point) bool {
	dx, dy := abs(a.X-c.X), abs(a.Y-c.Y)
	if b.Settings.DiagonalSwap {
		return max(dx, dy) == 1
	}
	return dx+dy == 1
}

// CanSwap reports whether the two cells may be physically swapped, ignoring whether it scores.
func (b *Board) CanSwap(a, c Point) bool {
	ga, gc := b.At(a), b.At(c)
	return ga != nil && gc != nil && !ga.Locked && !gc.Locked && b.Adjacent(a, c)
}

// IsValidSwap reports whether swapping a and c would be accepted as a move.
func (b *Board) IsValidSwap(a, c Point) bool {
	if !b.CanSwap(a, c) {
		return false
	}
	if b.Settings.FreeSwap {
		return true
	}
	ga, gc := b.At(a), b.At(c)
	if b.Settings.Duel {
		b.Swap(a, c)
		ok := len(b.runsThrough(a, c)) > 0
		b.Swap(a, c)
		return ok
	}
	if ga.Special == Prism || gc.Special == Prism {
		return true
	}
	if ga.Special != None && gc.Special != None {
		return true
	}
	b.Swap(a, c)
	ok := b.matchAt(a) || b.matchAt(c)
	b.Swap(a, c)
	return ok
}

func (b *Board) Swap(a, c Point) {
	ga, gc := b.At(a), b.At(c)
	b.set(a, gc)
	b.set(c, ga)
}

// FindMove returns any valid swap, used for hints and dead-board detection.
func (b *Board) FindMove() (Point, Point, bool) {
	dirs := []Point{ { 1, 0 }, { 0, 1 } }
	if b.Settings.DiagonalSwap {
		dirs = append(dirs, Point{ 1, 1 }, Point{ 1, -1 })
	}
	for y := 0; y < b.H(); y++ {
		for x := 0; x < b.W(); x++ {
			p := Point{ x, y }
			for _, d := range dirs {
				if q := p.Add(d); b.In(q) && b.IsValidSwap(p, q) {
					return p, q, true
				}
			}
		}
	}
	return Point{}, Point{}, false
}

func (b *Board) HasMove() bool {
	_, _, ok := b.FindMove()
	return ok
}

func (b *Board) HasMatches() bool {
	return len(b.findRuns()) > 0 || len(b.pending) > 0
}

func (b *Board) LockCount() int {
	n := 0
	b.each(func(_ Point, g *Gem) {
		if g.Locked {
			n++
		}
	})
	return n
}

// LocksLeft returns -1 when locks are unlimited.
func (b *Board) LocksLeft() int {
	if b.Settings.MaxLocks < 0 {
		return -1
	}
	return max(0, b.Settings.MaxLocks-b.LockCount())
}

// ToggleLock locks or unlocks the gem at p. It returns false if nothing changed.
func (b *Board) ToggleLock(p Point) bool {
	g := b.At(p)
	if g == nil {
		return false
	}
	if !g.Locked && b.LocksLeft() == 0 {
		return false
	}
	g.Locked = !g.Locked
	return true
}

// ReleaseAll unlocks every gem at once, so setups resolve together. Returns how many were released.
func (b *Board) ReleaseAll() int {
	n := 0
	b.each(func(_ Point, g *Gem) {
		if g.Locked {
			g.Locked = false
			n++
		}
	})
	return n
}

// Paint overwrites a gem in place (sandbox brush). Lock state and identity are kept.
func (b *Board) Paint(p Point, color int, special Special) {
	g := b.At(p)
	if g == nil {
		return
	}
	g.Special = special
	g.Color = color
	if special == Prism {
		g.Color = -1
	} else if g.Color < 0 {
		g.Color = b.rng.Intn(b.Settings.NumColors)
	}
}

// BeginMove commits a swap the player already made: it arms special combos and starts a new chain.
func (b *Board) BeginMove(a, c Point) {
	b.Moves++
	b.Chain = 0
	b.swapped = []Point{ c, a }
	if !b.Settings.Duel {
		b.applyCombo(a, c)
	}
	if b.Settings.ChaosGravity {
		b.Settings.Gravity = Direction(b.rng.Intn(4))
	}
}

// StartChain resets the chain counter for resolves not caused by a swap, like releasing locks.
func (b *Board) StartChain() {
	b.Chain = 0
	b.swapped = nil
}

// Force queues cells to be cleared on the next Step, for spells.
func (b *Board) Force(cells []Point) {
	b.forceArea(cells)
}

// CellsOfColor lists every gem of a color, Wilds excluded.
func (b *Board) CellsOfColor(color int) []Point {
	var out []Point
	b.each(func(p Point, g *Gem) {
		if g.Color == color {
			out = append(out, p)
		}
	})
	return out
}

// Convert repaints every gem of one color as another. It returns how many changed.
func (b *Board) Convert(from, to int) int {
	cells := b.CellsOfColor(from)
	for _, p := range cells {
		b.At(p).Color = to
	}
	return len(cells)
}

// RandomCells picks up to n distinct cells that aren't Wilds.
func (b *Board) RandomCells(n int) []Point {
	var all []Point
	b.each(func(p Point, g *Gem) {
		if g.Special != Wild {
			all = append(all, p)
		}
	})
	b.rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	return all[:min(n, len(all))]
}

// PlaceWild turns the gem at p into a Wild with the given multiplier.
func (b *Board) PlaceWild(p Point, mult int) {
	if g := b.At(p); g != nil {
		g.Color, g.Special, g.Mult = -1, Wild, mult
	}
}

// Rows and Cols list every cell in an inclusive range of lines, for spells.
func (b *Board) Rows(from, to int) []Point { return b.rows(from, to) }
func (b *Board) Cols(from, to int) []Point { return b.cols(from, to) }

// Square lists the cells within r of c.
func (b *Board) Square(c Point, r int) []Point { return b.square(c, r) }

func (b *Board) each(f func(Point, *Gem)) {
	for y := 0; y < b.H(); y++ {
		for x := 0; x < b.W(); x++ {
			if g := b.Cells[y][x]; g != nil {
				f(Point{ x, y }, g)
			}
		}
	}
}

// Snapshot is a copy of the board state for undo. Settings are not included.
type Snapshot struct {
	cells								[][]Gem
	score, chain, best, moves, cleared	int
}

func (b *Board) Snapshot() Snapshot {
	s := Snapshot{ score: b.Score, chain: b.Chain, best: b.BestChain, moves: b.Moves, cleared: b.Cleared }
	s.cells = make([][]Gem, b.H())
	for y := range s.cells {
		s.cells[y] = make([]Gem, b.W())
		for x := range s.cells[y] {
			if g := b.Cells[y][x]; g != nil {
				s.cells[y][x] = *g
			}
		}
	}
	return s
}

// Restore rolls back to a snapshot. It returns false if the board size has changed since.
func (b *Board) Restore(s Snapshot) bool {
	if len(s.cells) != b.H() || len(s.cells[0]) != b.W() {
		return false
	}
	for y := range s.cells {
		for x := range s.cells[y] {
			g := s.cells[y][x]
			b.Cells[y][x] = &g
		}
	}
	b.Score, b.Chain, b.BestChain, b.Moves, b.Cleared = s.score, s.chain, s.best, s.moves, s.cleared
	b.pending, b.swapped = nil, nil
	return true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func clampInt(n, lo, hi int) int {
	return max(lo, min(hi, n))
}
