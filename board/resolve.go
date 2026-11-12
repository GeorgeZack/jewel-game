package board

const MIN_MATCH = 3
const WILD_MULT = 3	// what a Wild earned by a run of 5 multiplies by

type run struct {
	cells		[]Point
	horizontal	bool
	color		int
}

type group struct {
	cells	[]Point
	runs	[]run
	color	int
}

type Cleared struct {
	Pos		Point
	Gem		Gem
	Matched	bool	// part of a matched run, rather than blasted or forced
}

type Blast struct {
	Pos		Point
	Special	Special
	Wide	bool	// overcharged, or a combo
}

// Match is one run cleared in a step. A Wild counts toward Size and multiplies the run.
type Match struct {
	Color, Size, Mult	int
	Wilds, Bombs		int
}

// StepResult describes one cascade step so the caller can animate it.
type StepResult struct {
	Cleared	[]Cleared
	Created	[]Point		// cells that now hold a freshly made special
	Blasts	[]Blast
	Matches	[]Match
	Points	int
	Chain	int
}

func isWild(g *Gem) bool {
	return g != nil && !g.Locked && g.Special == Wild
}

// scanLine appends the runs along one row or column.
func (b *Board) scanLine(runs []run, start Point, step Point, horizontal bool) []run {
	var cur []Point
	color, trailing := -1, 0	// trailing: how many Wilds end cur, they also start the next run
	flush := func() {
		if color >= 0 && len(cur) >= MIN_MATCH {
			runs = append(runs, run{ cells: cur, horizontal: horizontal, color: color })
		}
		cur, color, trailing = nil, -1, 0
	}
	for p := start; b.In(p); p = p.Add(step) {
		g := b.At(p)
		if isWild(g) {
			if color >= b.Settings.WildColors {
				flush()
			}
			cur = append(cur, p)
			trailing++
			continue
		}
		if !matchable(g) {
			flush()
			continue
		}
		if g.Color != color {
			var carried []Point
			if g.Color < b.Settings.WildColors {
				carried = append(carried, cur[len(cur)-trailing:]...)
			}
			flush()
			cur, color = carried, g.Color
		}
		cur = append(cur, p)
		trailing = 0
	}
	flush()
	return runs
}

func (b *Board) findRuns() []run {
	var runs []run
	for y := 0; y < b.H(); y++ {
		runs = b.scanLine(runs, Point{ 0, y }, Point{ 1, 0 }, true)
	}
	for x := 0; x < b.W(); x++ {
		runs = b.scanLine(runs, Point{ x, 0 }, Point{ 0, 1 }, false)
	}
	return runs
}

// runsThrough lists runs on the lines of two cells, which is all a swap of them can change.
func (b *Board) runsThrough(a, c Point) []run {
	runs := b.scanLine(nil, Point{ 0, a.Y }, Point{ 1, 0 }, true)
	runs = b.scanLine(runs, Point{ a.X, 0 }, Point{ 0, 1 }, false)
	if c.Y != a.Y {
		runs = b.scanLine(runs, Point{ 0, c.Y }, Point{ 1, 0 }, true)
	}
	if c.X != a.X {
		runs = b.scanLine(runs, Point{ c.X, 0 }, Point{ 0, 1 }, false)
	}
	return runs
}

// PreviewSwap lists the matches a swap would make right away, without changing the board.
func (b *Board) PreviewSwap(a, c Point) []Match {
	b.Swap(a, c)
	defer b.Swap(a, c)
	var out []Match
	for _, r := range b.runsThrough(a, c) {
		out = append(out, b.describe(r))
	}
	return out
}

func (b *Board) describe(r run) Match {
	m := Match{ Color: r.color, Size: len(r.cells), Mult: 1 }
	for _, p := range r.cells {
		switch g := b.At(p); g.Special {
		case Wild:
			m.Wilds++
			m.Mult *= max(g.Mult, 1)
		case Bomb:
			m.Bombs++
		}
	}
	return m
}

// findGroups merges runs that share a cell, so an L or T shape counts as one match.
func (b *Board) findGroups() []group {
	runs := b.findRuns()
	parent := make([]int, len(runs))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	owner := map[Point]int{}
	for i, r := range runs {
		for _, c := range r.cells {
			if j, ok := owner[c]; ok {
				parent[find(i)] = find(j)
			} else {
				owner[c] = i
			}
		}
	}

	byRoot := map[int]*group{}
	var order []int
	for i, r := range runs {
		root := find(i)
		g, ok := byRoot[root]
		if !ok {
			g = &group{ color: b.At(r.cells[0]).Color }
			byRoot[root] = g
			order = append(order, root)
		}
		g.runs = append(g.runs, r)
	}
	groups := make([]group, 0, len(order))
	for _, root := range order {
		g := byRoot[root]
		seen := map[Point]bool{}
		for _, r := range g.runs {
			for _, c := range r.cells {
				if !seen[c] {
					seen[c] = true
					g.cells = append(g.cells, c)
				}
			}
		}
		groups = append(groups, *g)
	}
	return groups
}

// classify decides what special, if any, a match earns.
func (g group) classify() Special {
	longest := g.runs[0]
	hasH, hasV := false, false
	for _, r := range g.runs {
		if len(r.cells) > len(longest.cells) {
			longest = r
		}
		hasH = hasH || r.horizontal
		hasV = hasV || !r.horizontal
	}
	switch {
	case len(longest.cells) >= 5:
		return Prism
	case hasH && hasV:
		return Bomb
	case len(longest.cells) == 4 && longest.horizontal:
		return BoltH
	case len(longest.cells) == 4:
		return BoltV
	}
	return None
}

// placement picks where a new special appears: where the player swapped if possible,
// otherwise the crossing of an L/T, otherwise the middle of the longest run.
func (b *Board) placement(g group) Point {
	in := map[Point]int{}
	for _, r := range g.runs {
		for _, c := range r.cells {
			in[c]++
		}
	}
	for _, p := range b.swapped {
		if in[p] > 0 {
			return p
		}
	}
	for _, c := range g.cells {
		if in[c] > 1 {
			return c
		}
	}
	longest := g.runs[0]
	for _, r := range g.runs {
		if len(r.cells) > len(longest.cells) {
			longest = r
		}
	}
	return longest.cells[len(longest.cells)/2]
}

func (b *Board) force(p Point) {
	if g := b.At(p); g != nil && !g.Locked {
		b.pending = append(b.pending, p)
	}
}

func (b *Board) forceArea(cells []Point) {
	for _, p := range cells {
		b.force(p)
	}
}

// applyCombo handles swaps where a special meets another special or a prism.
func (b *Board) applyCombo(a, c Point) {
	ga, gc := b.At(a), b.At(c)
	if ga == nil || gc == nil {
		return
	}
	sa, sc := ga.Special, gc.Special
	switch {
	case sa == Prism && sc == Prism:
		ga.Special, gc.Special = None, None
		b.each(func(p Point, _ *Gem) { b.force(p) })

	case sa == Prism || sc == Prism:
		prism, other := ga, gc
		if sc == Prism {
			prism, other = gc, ga
		}
		prism.Special = None
		b.force(a)
		b.force(c)
		color, spread := other.Color, other.Special
		b.each(func(p Point, g *Gem) {
			if !matchable(g) || g.Color != color {
				return
			}
			// Prism + special turns every gem of that color into the special
			if spread != None && g.Special == None {
				g.Special = spread
				if spread.IsBolt() {
					g.Special = BoltH + Special(b.rng.Intn(2))
				}
			}
			b.force(p)
		})

	case sa.IsBolt() && sc.IsBolt():
		ga.Special, gc.Special = BoltH, BoltV
		b.force(a)
		b.force(c)

	case sa == Bomb && sc == Bomb:
		ga.Special, gc.Special = None, None
		b.forceArea(b.square(c, b.bombRadius()+1))

	case sa != None && sc != None:
		// Bolt + Bomb: a fat cross three (or five) lines wide
		ga.Special, gc.Special = None, None
		w := b.boltHalfWidth() + 1
		b.forceArea(b.rows(c.Y-w, c.Y+w))
		b.forceArea(b.cols(c.X-w, c.X+w))
	}
}

func (b *Board) bombRadius() int {
	if b.Settings.Overcharge {
		return 2
	}
	return 1
}

func (b *Board) boltHalfWidth() int {
	if b.Settings.Overcharge {
		return 1
	}
	return 0
}

func (b *Board) square(c Point, r int) []Point {
	var out []Point
	for y := c.Y - r; y <= c.Y+r; y++ {
		for x := c.X - r; x <= c.X+r; x++ {
			if p := (Point{ x, y }); b.In(p) {
				out = append(out, p)
			}
		}
	}
	return out
}

func (b *Board) rows(from, to int) []Point {
	var out []Point
	for y := max(0, from); y <= min(b.H()-1, to); y++ {
		for x := 0; x < b.W(); x++ {
			out = append(out, Point{ x, y })
		}
	}
	return out
}

func (b *Board) cols(from, to int) []Point {
	var out []Point
	for x := max(0, from); x <= min(b.W()-1, to); x++ {
		for y := 0; y < b.H(); y++ {
			out = append(out, Point{ x, y })
		}
	}
	return out
}

func (b *Board) mostCommonColor() int {
	counts := make([]int, b.Settings.NumColors)
	b.each(func(_ Point, g *Gem) {
		if matchable(g) && g.Color < len(counts) {
			counts[g.Color]++
		}
	})
	best := 0
	for c, n := range counts {
		if n > counts[best] {
			best = c
		}
	}
	return best
}

// blastArea is what a special destroys when caught in a match or another blast.
func (b *Board) blastArea(p Point, g *Gem) []Point {
	w := b.boltHalfWidth()
	switch g.Special {
	case BoltH:
		return b.rows(p.Y-w, p.Y+w)
	case BoltV:
		return b.cols(p.X-w, p.X+w)
	case Bomb:
		return b.square(p, b.bombRadius())
	case Prism:
		color := b.mostCommonColor()
		var out []Point
		b.each(func(q Point, o *Gem) {
			if matchable(o) && o.Color == color {
				out = append(out, q)
			}
		})
		return out
	}
	return nil
}

// Step resolves one cascade: clears every match and forced cell, sets off any specials
// caught in it, and places newly earned specials. Returns nil when the board is stable.
func (b *Board) Step() *StepResult {
	clear := map[Point]bool{}
	var order []Point
	add := func(p Point) {
		if g := b.At(p); g != nil && !g.Locked && !clear[p] {
			clear[p] = true
			order = append(order, p)
		}
	}
	for _, p := range b.pending {
		add(p)
	}

	type creation struct {
		pos		Point
		color	int
		special	Special
	}
	var creations []creation
	var matches []Match
	matched := map[Point]bool{}
	for _, g := range b.findGroups() {
		for _, c := range g.cells {
			add(c)
			matched[c] = true
		}
		longest := 0
		for _, r := range g.runs {
			matches = append(matches, b.describe(r))
			longest = max(longest, len(r.cells))
		}
		if b.Settings.Duel {
			if longest >= 5 {
				creations = append(creations, creation{ b.placement(g), -1, Wild })
			}
		} else if s := g.classify(); s != None {
			creations = append(creations, creation{ b.placement(g), g.color, s })
		}
	}

	res := &StepResult{ Matches: matches }
	// order grows while we walk it, so chain reactions keep going until they run out
	for i := 0; i < len(order); i++ {
		p := order[i]
		g := b.At(p)
		if g.Special == None || g.Special == Wild {
			continue
		}
		res.Blasts = append(res.Blasts, Blast{ Pos: p, Special: g.Special, Wide: b.Settings.Overcharge })
		for _, q := range b.blastArea(p, g) {
			add(q)
		}
	}

	b.pending, b.swapped = nil, nil
	if len(order) == 0 {
		return nil
	}

	b.Chain++
	b.BestChain = max(b.BestChain, b.Chain)
	res.Chain = b.Chain
	// Earning several specials in one step is the payoff for lock setups, so it scales hard
	res.Points = len(order)*10*b.Chain + len(creations)*len(creations)*100
	b.Score += res.Points
	b.Cleared += len(order)

	for _, p := range order {
		res.Cleared = append(res.Cleared, Cleared{ Pos: p, Gem: *b.At(p), Matched: matched[p] })
		b.set(p, nil)
	}
	for _, c := range creations {
		g := b.newGem(c.color, c.special)
		if c.special == Wild {
			g.Mult = WILD_MULT
		}
		b.set(c.pos, g)
		res.Created = append(res.Created, c.pos)
	}
	return res
}

// lanes returns each line gems fall along, ordered from the floor up.
func (b *Board) lanes() [][]Point {
	var out [][]Point
	d := b.Settings.Gravity
	vertical := d == Down || d == Up
	n, length := b.W(), b.H()
	if !vertical {
		n, length = b.H(), b.W()
	}
	for i := 0; i < n; i++ {
		lane := make([]Point, length)
		for j := 0; j < length; j++ {
			k := j
			if d == Down || d == Right {
				k = length - 1 - j
			}
			if vertical {
				lane[j] = Point{ i, k }
			} else {
				lane[j] = Point{ k, i }
			}
		}
		out = append(out, lane)
	}
	return out
}

// Collapse applies gravity and refills the gaps. Locked gems hang in place and others
// fall past them. It returns where each new gem should appear from, which may be off the board.
func (b *Board) Collapse() map[int]Point {
	spawns := map[int]Point{}
	up := b.Settings.Gravity.Vec()
	up = Point{ -up.X, -up.Y }
	for _, lane := range b.lanes() {
		var free []Point
		var gems []*Gem
		for _, p := range lane {
			if g := b.At(p); g != nil && g.Locked {
				continue
			}
			free = append(free, p)
			if g := b.At(p); g != nil {
				gems = append(gems, g)
			}
		}
		top := lane[len(lane)-1]
		for i, p := range free {
			if i < len(gems) {
				b.set(p, gems[i])
				continue
			}
			g := b.refillGem()
			if g == nil {
				b.set(p, nil)
				continue
			}
			k := i - len(gems) + 1
			spawns[g.ID] = Point{ top.X + up.X*k, top.Y + up.Y*k }
			b.set(p, g)
		}
	}
	return spawns
}

// Shuffle rearranges unlocked gems when no move is left. Returns false if it wasn't needed.
func (b *Board) Shuffle() bool {
	if b.HasMove() {
		return false
	}
	var spots []Point
	var gems []*Gem
	b.each(func(p Point, g *Gem) {
		if !g.Locked {
			spots = append(spots, p)
			gems = append(gems, g)
		}
	})
	for attempt := 0; attempt < 400; attempt++ {
		b.rng.Shuffle(len(gems), func(i, j int) { gems[i], gems[j] = gems[j], gems[i] })
		if attempt >= 200 {
			// the colors themselves are stuck, so reroll them too
			for _, g := range gems {
				if g.Special != Prism && g.Special != Wild {
					g.Color = b.rng.Intn(b.Settings.NumColors)
				}
			}
		}
		for i, p := range spots {
			b.set(p, gems[i])
		}
		if len(b.findRuns()) == 0 && b.HasMove() {
			break
		}
	}
	return true
}

func (b *Board) matchAt(p Point) bool {
	g := b.At(p)
	if !matchable(g) {
		return false
	}
	count := func(dx, dy int) int {
		n := 0
		for q := (Point{ p.X + dx, p.Y + dy }); ; q = q.Add(Point{ dx, dy }) {
			o := b.At(q)
			if !matchable(o) || o.Color != g.Color {
				return n
			}
			n++
		}
	}
	return count(-1, 0)+count(1, 0)+1 >= MIN_MATCH || count(0, -1)+count(0, 1)+1 >= MIN_MATCH
}
