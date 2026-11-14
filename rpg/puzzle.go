package rpg

import (
	"fmt"
	"jewel-game/board"
	"slices"
	"strings"
)

// A puzzle board is an authored board with an objective instead of a foe: puzzles/*.yaml. Gems
// marked on it are followed through the board's StepResults, and the route the player took is the
// first route whose conditions hold when the objective completes.

// PUZZLE_COLORS are the letters a puzzle board is drawn with, in gem color order.
const PUZZLE_COLORS = "rbygp"

type Puzzle struct {
	ID			string			`yaml:"id"`
	Name		string			`yaml:"name"`
	Objective	string			`yaml:"objective"`	// shown to the player
	Board		[]string		`yaml:"board"`		// rows, top first: a color letter per cell, see docs/story.md
	Marks		map[string][2]int	`yaml:"marks"`	// mark → [x, y]
	Gravity		string			`yaml:"gravity"`
	Moves		int				`yaml:"moves"`		// moves allowed: swaps and tool uses
	Refill		string			`yaml:"refill"`		// none (default), queue or seed
	Queue		string			`yaml:"queue"`		// refill colors in order, for refill: queue
	Seed		int64			`yaml:"seed"`
	Goal		PuzzleCond		`yaml:"goal"`
	Fail		[]PuzzleCond	`yaml:"fail"`		// any that holds after a move loses
	Tools		[]PuzzleTool	`yaml:"tools"`
	Routes		[]PuzzleRoute	`yaml:"routes"`
}

// PuzzleCond holds when all of its parts do. Parts left out don't matter.
type PuzzleCond struct {
	Cleared		[]string	`yaml:"cleared"`		// these marks are cleared
	Order		[]string	`yaml:"order"`		// these marks are cleared, each in a later step than the one before
	Blasted		*bool		`yaml:"blasted"`		// a mark was cleared by a blast or a Force (true), or none was (false)
	Tool		*bool		`yaml:"tool"`		// a tool was used, or none was
	MaxMatch	int			`yaml:"max_match"`	// the largest match was at most this
	MinMatch	int			`yaml:"min_match"`	// the largest match was at least this
	MaxMoves	int			`yaml:"max_moves"`	// at most this many moves (swaps and tool uses) were made
}

type PuzzleRoute struct {
	ID			string		`yaml:"id"`
	Name		string		`yaml:"name"`
	Flag		string		`yaml:"flag"`		// set on the hero when this route is taken
	Standing	map[string]int	`yaml:"standing,flow"`	// society → standing gained (or lost) by taking it
	When		PuzzleCond	`yaml:"when"`
	Solution	[]string	`yaml:"solution"`	// moves that take this route, replayed by validation
}

// PuzzleTool is a rite used on the board, a few times a puzzle instead of for mana.
type PuzzleTool struct {
	ID		string		`yaml:"id"`
	Name	string		`yaml:"name"`
	Kind	string		`yaml:"kind"`	// force: clear one gem; convert: repaint one color as another; paint: repaint one gem
	From	string		`yaml:"from"`	// convert and paint: color letters
	To		string		`yaml:"to"`
	Rites	[]string	`yaml:"rites"`	// the hero needs one of these rites
	Uses	int			`yaml:"uses"`
}

var Puzzles []Puzzle

func PuzzleByID(id string) *Puzzle {
	for i := range Puzzles {
		if Puzzles[i].ID == id {
			return &Puzzles[i]
		}
	}
	return nil
}

func puzzleColor(letter string) int {
	if len(letter) != 1 {
		return -1
	}
	return strings.Index(PUZZLE_COLORS, letter)
}

// cells parses the board. A cell is a color letter, "." for empty, and may end in "#" for a lock,
// "*" for a bomb, "-" or "|" for a bolt.
func (p *Puzzle) cells() ([][]*board.Gem, error) {
	var out [][]*board.Gem
	for y, row := range p.Board {
		var line []*board.Gem
		for x, tok := range strings.Fields(row) {
			if tok == "." {
				line = append(line, nil)
				continue
			}
			g := &board.Gem{ Color: puzzleColor(tok[:1]) }
			if g.Color < 0 {
				return nil, fmt.Errorf("cell %d,%d: unknown color %q", x, y, tok[:1])
			}
			for _, c := range tok[1:] {
				switch c {
				case '#':
					g.Locked = true
				case '*':
					g.Special = board.Bomb
				case '-':
					g.Special = board.BoltH
				case '|':
					g.Special = board.BoltV
				default:
					return nil, fmt.Errorf("cell %d,%d: unknown mark %q", x, y, c)
				}
			}
			line = append(line, g)
		}
		if len(out) > 0 && len(line) != len(out[0]) {
			return nil, fmt.Errorf("row %d has %d cells, not %d", y, len(line), len(out[0]))
		}
		out = append(out, line)
	}
	if len(out) < 4 || len(out) > MAX_CARD_SIZE || len(out[0]) < 4 || len(out[0]) > MAX_CARD_SIZE {
		return nil, fmt.Errorf("the board should be 4 to %d cells each way", MAX_CARD_SIZE)
	}
	return out, nil
}

// NewBoard builds the puzzle's board. The same puzzle always builds the same board.
func (p *Puzzle) NewBoard() (*board.Board, error) {
	cells, err := p.cells()
	if err != nil {
		return nil, err
	}
	s := board.DefaultSettings()
	s.NumColors = len(PUZZLE_COLORS)
	s.MaxLocks = 0
	s.Gravity, _ = gravityByName(p.Gravity)
	b := board.NewBoardFrom(s, p.Seed, cells)
	switch p.Refill {
	case "", "none":
		b.NoRefill = true
	case "queue":
		b.NoRefill = true
		for _, c := range p.Queue {
			b.Queue = append(b.Queue, puzzleColor(string(c)))
		}
	}
	return b, nil
}

type PuzzleState int

const (
	PuzzlePlaying PuzzleState = iota
	PuzzleWon
	PuzzleLost
)

type markClear struct {
	step	int
	blast	bool
}

// PuzzleRun is one attempt at a puzzle. Moves go through the board as usual; the run watches
// each StepResult (Observe) and judges the board when a move has resolved (Settle).
type PuzzleRun struct {
	P			*Puzzle
	Hero		*Hero		// nil: every tool is available
	Board		*board.Board
	Uses		map[string]int	// tool ID → uses left
	ToolUsed	bool
	toolMoves	int			// tool uses, which count as moves
	State		PuzzleState
	Route		*PuzzleRoute
	Lost		string		// why: moves, stuck, fail or jammed
	marks		map[int]string	// gem ID → mark
	cleared		map[string]markClear
	steps		int
	largest		int
}

func NewPuzzleRun(p *Puzzle, h *Hero) (*PuzzleRun, error) {
	b, err := p.NewBoard()
	if err != nil {
		return nil, err
	}
	r := &PuzzleRun{ P: p, Hero: h, Board: b, Uses: map[string]int{}, marks: map[int]string{}, cleared: map[string]markClear{} }
	for name, at := range p.Marks {
		if g := b.At(board.Point{ X: at[0], Y: at[1] }); g != nil {
			r.marks[g.ID] = name
		}
	}
	for _, t := range p.Tools {
		if r.HasTool(&t) {
			r.Uses[t.ID] = t.Uses
		}
	}
	return r, nil
}

// HasTool reports whether the hero knows a rite the tool needs.
func (r *PuzzleRun) HasTool(t *PuzzleTool) bool {
	if r.Hero == nil {
		return true
	}
	return slices.ContainsFunc(t.Rites, func(id string) bool {
		s := SpellByID(id)
		return s != nil && r.Hero.Knows(s)
	})
}

// Moves is how many moves have been made: swaps and tool uses.
func (r *PuzzleRun) Moves() int { return r.Board.Moves + r.toolMoves }

// Targeted reports whether a tool is aimed at one gem.
func (t *PuzzleTool) Targeted() bool { return t.Kind == "force" || t.Kind == "paint" }

// MarkAt names the mark on the gem at p, or "".
func (r *PuzzleRun) MarkAt(p board.Point) string {
	if g := r.Board.At(p); g != nil {
		return r.marks[g.ID]
	}
	return ""
}

// Observe notes what one cascade step cleared.
func (r *PuzzleRun) Observe(res *board.StepResult) {
	r.steps++
	for _, m := range res.Matches {
		r.largest = max(r.largest, m.Size)
	}
	for _, c := range res.Cleared {
		if name := r.marks[c.Gem.ID]; name != "" {
			if _, done := r.cleared[name]; !done {
				r.cleared[name] = markClear{ r.steps, !c.Matched }
			}
		}
	}
}

func (r *PuzzleRun) holds(c *PuzzleCond) bool {
	for _, m := range c.Cleared {
		if _, ok := r.cleared[m]; !ok {
			return false
		}
	}
	last := 0
	for _, m := range c.Order {
		mc, ok := r.cleared[m]
		if !ok || mc.step <= last {
			return false
		}
		last = mc.step
	}
	if c.Blasted != nil {
		blasted := false
		for _, mc := range r.cleared {
			blasted = blasted || mc.blast
		}
		if blasted != *c.Blasted {
			return false
		}
	}
	if c.Tool != nil && r.ToolUsed != *c.Tool {
		return false
	}
	if c.MaxMatch > 0 && r.largest > c.MaxMatch {
		return false
	}
	if c.MinMatch > 0 && r.largest < c.MinMatch {
		return false
	}
	if c.MaxMoves > 0 && r.Moves() > c.MaxMoves {
		return false
	}
	return true
}

// Settle judges the board once a move or tool has resolved. Finishing the objective by no route
// at all (say, the tumbler pins cleared out of order) jams the lock and loses.
func (r *PuzzleRun) Settle() {
	if r.State != PuzzlePlaying {
		return
	}
	if r.holds(&r.P.Goal) {
		for i := range r.P.Routes {
			if r.holds(&r.P.Routes[i].When) {
				r.State, r.Route = PuzzleWon, &r.P.Routes[i]
				return
			}
		}
		r.State, r.Lost = PuzzleLost, "jammed"
		return
	}
	for i := range r.P.Fail {
		if r.holds(&r.P.Fail[i]) {
			r.State, r.Lost = PuzzleLost, "fail"
			return
		}
	}
	tools := false
	for _, n := range r.Uses {
		tools = tools || n > 0
	}
	switch {
	case r.Moves() >= r.P.Moves:
		r.State, r.Lost = PuzzleLost, "moves"
	case !r.Board.HasMove() && !tools:
		r.State, r.Lost = PuzzleLost, "stuck"
	}
}

// ApplyTool spends a use of a tool on the board, at p for a Force. The caller then resolves the
// board as a move without a swap (board.StartChain) and calls Settle.
func (r *PuzzleRun) ApplyTool(id string, p board.Point) bool {
	var t *PuzzleTool
	for i := range r.P.Tools {
		if r.P.Tools[i].ID == id {
			t = &r.P.Tools[i]
		}
	}
	if t == nil || r.State != PuzzlePlaying || r.Uses[id] <= 0 {
		return false
	}
	switch t.Kind {
	case "force":
		if g := r.Board.At(p); g == nil || g.Locked {
			return false
		}
		r.Board.Force([]board.Point{ p })
	case "convert":
		if r.Board.Convert(puzzleColor(t.From), puzzleColor(t.To)) == 0 {
			return false
		}
	case "paint":
		g := r.Board.At(p)
		if g == nil || g.Locked || g.Special != board.None || g.Color != puzzleColor(t.From) {
			return false
		}
		g.Color = puzzleColor(t.To)
	default:
		return false
	}
	r.Uses[id]--
	r.ToolUsed = true
	r.toolMoves++
	r.Board.StartChain()
	return true
}

// resolve runs the board until it settles, as the view does with animation.
func (r *PuzzleRun) resolve() {
	for {
		res := r.Board.Step()
		if res == nil {
			break
		}
		r.Observe(res)
		r.Board.Collapse()
	}
	r.Settle()
}

// PuzzleMove is one move of a stored solution, written "swap x,y x,y", "tool id" or "tool id x,y".
type PuzzleMove struct {
	Tool	string		// "" for a swap
	A, C	board.Point	// the swap, or A is where a Force lands
}

func ParseMove(s string) (PuzzleMove, bool) {
	var m PuzzleMove
	f := strings.Fields(s)
	point := func(s string, p *board.Point) bool {
		_, err := fmt.Sscanf(s, "%d,%d", &p.X, &p.Y)
		return err == nil
	}
	switch {
	case len(f) == 3 && f[0] == "swap":
		return m, point(f[1], &m.A) && point(f[2], &m.C)
	case len(f) == 2 && f[0] == "tool":
		m.Tool = f[1]
		return m, true
	case len(f) == 3 && f[0] == "tool":
		m.Tool = f[1]
		return m, point(f[2], &m.A)
	}
	return m, false
}

// Play makes one move written as in a stored solution, and resolves the board. It is false if
// the move isn't allowed.
func (r *PuzzleRun) Play(move string) bool {
	m, ok := ParseMove(move)
	if !ok || r.State != PuzzlePlaying {
		return false
	}
	if m.Tool != "" {
		if !r.ApplyTool(m.Tool, m.A) {
			return false
		}
	} else {
		if !r.Board.In(m.A) || !r.Board.In(m.C) || !r.Board.Adjacent(m.A, m.C) || !r.Board.IsValidSwap(m.A, m.C) {
			return false
		}
		r.Board.Swap(m.A, m.C)
		r.Board.BeginMove(m.A, m.C)
	}
	r.resolve()
	return true
}

// Replay plays a stored solution from the start. It stops at the first move that can't be made.
func (p *Puzzle) Replay(moves []string) (*PuzzleRun, error) {
	r, err := NewPuzzleRun(p, nil)
	if err != nil {
		return nil, err
	}
	for i, m := range moves {
		if !r.Play(m) {
			return r, fmt.Errorf("move %d (%q) can't be made", i+1, m)
		}
	}
	return r, nil
}

// Finish banks a won puzzle: the route's flag is set on the hero.
func (r *PuzzleRun) Finish(h *Hero) {
	if r.State != PuzzleWon || h == nil {
		return
	}
	h.Apply(&Step{ SetFlag: r.Route.Flag, Standing: r.Route.Standing })
	h.earn("route:"+r.P.ID+"/"+r.Route.ID, Grimoire.Pay.Route)
}

func checkPuzzles() []string {
	var p []string
	bad := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	seen := map[string]bool{}
	for i := range Puzzles {
		pz := &Puzzles[i]
		where := "puzzle " + pz.ID
		if pz.ID == "" {
			bad("puzzles: a puzzle has no id")
		} else if seen[pz.ID] {
			bad("%s: defined twice", where)
		}
		seen[pz.ID] = true
		cells, err := pz.cells()
		if err != nil {
			bad("%s: %v", where, err)
			continue
		}
		if _, ok := gravityByName(pz.Gravity); !ok {
			bad("%s: unknown gravity %q", where, pz.Gravity)
		}
		if !slices.Contains([]string{ "", "none", "queue", "seed" }, pz.Refill) {
			bad("%s: unknown refill %q (none, queue or seed)", where, pz.Refill)
		}
		for _, c := range pz.Queue {
			if puzzleColor(string(c)) < 0 {
				bad("%s: unknown queue color %q", where, c)
			}
		}
		if pz.Moves <= 0 {
			bad("%s: moves should be more than 0", where)
		}
		for name, at := range pz.Marks {
			if at[1] < 0 || at[1] >= len(cells) || at[0] < 0 || at[0] >= len(cells[0]) || cells[at[1]][at[0]] == nil {
				bad("%s: mark %s isn't on a gem", where, name)
			}
		}
		conds := append([]PuzzleCond{ pz.Goal }, pz.Fail...)
		for _, r := range pz.Routes {
			conds = append(conds, r.When)
		}
		for _, c := range conds {
			for _, m := range append(slices.Clone(c.Cleared), c.Order...) {
				if _, ok := pz.Marks[m]; !ok {
					bad("%s: unknown mark %q", where, m)
				}
			}
		}
		if len(pz.Goal.Cleared) == 0 && len(pz.Goal.Order) == 0 {
			bad("%s: the goal clears no marks", where)
		}
		for _, t := range pz.Tools {
			switch t.Kind {
			case "force":
			case "convert", "paint":
				if puzzleColor(t.From) < 0 || puzzleColor(t.To) < 0 {
					bad("%s tool %s: convert needs from and to colors", where, t.ID)
				}
			default:
				bad("%s tool %s: unknown kind %q (force, convert or paint)", where, t.ID, t.Kind)
			}
			for _, id := range t.Rites {
				if s := SpellByID(id); s == nil || s.Class == "" {
					bad("%s tool %s: unknown rite %q", where, t.ID, id)
				}
			}
		}
		if len(pz.Routes) == 0 {
			bad("%s: no routes", where)
		}
		// every route's stored solution must still take that route
		for _, r := range pz.Routes {
			for soc := range r.Standing {
				if SocietyByID(soc) == nil {
					bad("%s route %s: standing with unknown society %q", where, r.ID, soc)
				}
			}
			if len(r.Solution) == 0 {
				bad("%s route %s: no stored solution", where, r.ID)
				continue
			}
			run, err := pz.Replay(r.Solution)
			switch {
			case err != nil:
				bad("%s route %s: stored solution fails: %v", where, r.ID, err)
			case run.State != PuzzleWon:
				bad("%s route %s: stored solution doesn't finish the puzzle (lost: %q)", where, r.ID, run.Lost)
			case run.Route.ID != r.ID:
				bad("%s route %s: stored solution takes route %s instead", where, r.ID, run.Route.ID)
			}
		}
	}
	return p
}
