package core

import (
	"jewel-game/board"
	"jewel-game/entities"
	"jewel-game/text"
	"jewel-game/ui"
	"jewel-game/utils"
	"math"
	"math/rand"
	"time"

	"jewel-game/rl"
)

type Phase int

const (
	PhaseIdle      Phase = iota
	PhaseSwapping        // gems sliding into a swap
	PhaseSwapBack        // a swap that didn't match, sliding back
	PhaseResolving       // ready to run the next cascade step
	PhaseClearing        // cleared gems vanishing
	PhaseFalling         // gems falling into gaps
)

const CLEAR_TIME = 0.22

// IDLE_QUIET is how long a settled board waits between one-shot gem animations.
const IDLE_QUIET = 4.0

// Brush is the sandbox paint tool; while one is set, clicking a gem repaints it.
type Brush struct {
	Color   int
	Special board.Special
}

// TileManager runs one board: it turns input into board moves and animates the results.
type TileManager struct {
	Board            *board.Board
	DataManager      *DataManager
	Tiles            map[int]*Tile
	Dying            []*Tile
	Effects          Effects
	Phase            Phase
	ScreenX, ScreenY float32
	CellSize         float32
	Speed            float32
	UndoEnabled      bool
	Brush            *Brush
	Paused           bool                       // ignore board input, e.g. during the enemy's turn
	holdInput        float32                    // seconds left ignoring board input, so a double click on a popup doesn't land on the board
	OnStep           func(*board.StepResult)    // called for every cascade step
	OnMoveDone       func()                     // called when a swap has fully resolved
	OnShuffle        func()                     // called when the board ran out of moves
	Types            func() []*entities.GemType // overrides the gem palette
	Duel             bool                       // no locks and no score popups
	moving           bool
	selected         *board.Point
	dragFrom         *board.Point
	swapA, swapB     board.Point
	swapValid        bool
	clearTimer       float32
	hint             []board.Point
	hintTimer        float32
	undo             []board.Snapshot
	spawns           map[int]board.Point
	shuffled         bool
	settled          bool
	time             float64
	quiet            float64 // seconds the board has been settled since the last idle
	idleID           int     // gem playing its strip, or 0
	idleStart        float64
	idleDur          float64
	lastIdle         int
}

func NewTileManager(settings board.Settings, dm *DataManager) *TileManager {
	tm := &TileManager{
		DataManager: dm,
		CellSize:    utils.CELL_WIDTH,
		Speed:       1,
	}
	tm.Reset(settings)
	return tm
}

// Reset starts a fresh board with the given settings.
func (tm *TileManager) Reset(settings board.Settings) {
	tm.Board = board.NewBoard(settings, time.Now().UnixNano())
	tm.Tiles = make(map[int]*Tile)
	tm.Dying = nil
	tm.Phase = PhaseIdle
	tm.selected, tm.dragFrom, tm.hint = nil, nil, nil
	tm.undo = nil
	tm.spawns = nil
	tm.moving = false
	tm.quiet, tm.idleID, tm.idleStart, tm.idleDur, tm.lastIdle = 0, 0, 0, 0, 0
	// new boards drop in from above
	for y, row := range tm.Board.Cells {
		for x, g := range row {
			tile := NewTile(g, rl.Vector2{X: float32(x), Y: float32(y - tm.Board.H() - 1)}, 1)
			tile.Target = rl.Vector2{X: float32(x), Y: float32(y)}
			tm.Tiles[g.ID] = tile
		}
	}
	ui.AddMessage("%s", text.Tf("log.new_board", "New board: %+v", settings))
}

// Layout fits the board inside r, as large as allowed, centered.
func (tm *TileManager) Layout(r rl.Rectangle) {
	w, h := float32(tm.Board.W()), float32(tm.Board.H())
	tm.CellSize = float32(math.Floor(float64(min(r.Width/w, r.Height/h, utils.MAX_CELL_WIDTH))))
	tm.ScreenX = r.X + (r.Width-tm.CellSize*w)/2
	tm.ScreenY = r.Y + (r.Height-tm.CellSize*h)/2
}

func (tm *TileManager) Rect() rl.Rectangle {
	return rl.Rectangle{
		X:      tm.ScreenX,
		Y:      tm.ScreenY,
		Width:  tm.CellSize * float32(tm.Board.W()),
		Height: tm.CellSize * float32(tm.Board.H()),
	}
}

func (tm *TileManager) CellCenter(p rl.Vector2) rl.Vector2 {
	return rl.Vector2{X: tm.ScreenX + (p.X+0.5)*tm.CellSize, Y: tm.ScreenY + (p.Y+0.5)*tm.CellSize}
}

func (tm *TileManager) pointCenter(p board.Point) rl.Vector2 {
	return tm.CellCenter(rl.Vector2{X: float32(p.X), Y: float32(p.Y)})
}

func (tm *TileManager) CellAt(pos rl.Vector2) (board.Point, bool) {
	x := int(math.Floor(float64((pos.X - tm.ScreenX) / tm.CellSize)))
	y := int(math.Floor(float64((pos.Y - tm.ScreenY) / tm.CellSize)))
	p := board.Point{X: x, Y: y}
	return p, tm.Board.In(p)
}

func (tm *TileManager) Busy() bool { return tm.Phase != PhaseIdle }

// HoldInput ignores clicks on the board for a moment, after a popup closes.
func (tm *TileManager) HoldInput() { tm.holdInput = 0.35 }

func (tm *TileManager) Update() {
	dt := rl.GetFrameTime() * tm.Speed
	tm.time += float64(rl.GetFrameTime())
	tm.holdInput = max(0, tm.holdInput-rl.GetFrameTime())
	if tm.Phase == PhaseIdle && !tm.Paused && tm.holdInput == 0 {
		tm.handleInput()
	}
	tm.syncTiles()
	tm.animate(dt)
	tm.step(dt)
	tm.Effects.Update(dt)
	tm.updateIdle(float64(rl.GetFrameTime()))
	if tm.hintTimer > 0 {
		tm.hintTimer -= rl.GetFrameTime()
	}
}

// syncTiles makes sure every gem on the board has a sprite heading to the right cell.
func (tm *TileManager) syncTiles() {
	seen := make(map[int]bool, len(tm.Tiles))
	for y, row := range tm.Board.Cells {
		for x, g := range row {
			if g == nil {
				continue
			}
			at := rl.Vector2{X: float32(x), Y: float32(y)}
			t, ok := tm.Tiles[g.ID]
			if !ok {
				if sp, ok := tm.spawns[g.ID]; ok {
					t = NewTile(g, rl.Vector2{X: float32(sp.X), Y: float32(sp.Y)}, 1)
					delete(tm.spawns, g.ID)
				} else {
					t = NewTile(g, at, 0)
				}
				tm.Tiles[g.ID] = t
			}
			t.Gem = *g
			t.Target = at
			seen[g.ID] = true
		}
	}
	for id := range tm.Tiles {
		if !seen[id] {
			delete(tm.Tiles, id)
		}
	}
}

func (tm *TileManager) animate(dt float32) {
	tm.settled = true
	for _, t := range tm.Tiles {
		if !t.Update(dt) {
			tm.settled = false
		}
	}
	alive := tm.Dying[:0]
	for _, t := range tm.Dying {
		if !t.Update(dt) {
			alive = append(alive, t)
		}
	}
	tm.Dying = alive
}

// step advances the swap → clear → fall → clear... cycle once the animation for the current phase is done.
func (tm *TileManager) step(dt float32) {
	switch tm.Phase {
	case PhaseSwapping:
		if !tm.settled {
			return
		}
		if tm.swapValid {
			tm.Board.BeginMove(tm.swapA, tm.swapB)
			tm.moving = true
			tm.Phase = PhaseResolving
		} else {
			tm.Board.Swap(tm.swapA, tm.swapB)
			tm.dropUndo()
			tm.Phase = PhaseSwapBack
		}
	case PhaseSwapBack:
		if tm.settled {
			tm.Phase = PhaseIdle
		}
	case PhaseResolving:
		res := tm.Board.Step()
		if res == nil {
			tm.finishResolve()
			return
		}
		tm.showStep(res)
		if tm.OnStep != nil {
			tm.OnStep(res)
		}
		tm.clearTimer = CLEAR_TIME
		tm.Phase = PhaseClearing
	case PhaseClearing:
		tm.clearTimer -= dt
		if tm.clearTimer <= 0 && len(tm.Dying) == 0 {
			tm.spawns = tm.Board.Collapse()
			tm.Phase = PhaseFalling
		}
	case PhaseFalling:
		if tm.settled {
			tm.Phase = PhaseResolving
		}
	}
}

func (tm *TileManager) finishResolve() {
	if !tm.Board.Settings.FreeSwap && !tm.shuffled && tm.Board.Shuffle() {
		tm.shuffled = true
		if tm.OnShuffle != nil {
			tm.OnShuffle()
		}
		r := tm.Rect()
		tm.Effects.Popup(text.T("ui.no_moves", "No moves left - shuffling"), rl.Vector2{X: r.X + r.Width/2, Y: r.Y + r.Height/2}, 34, ui.Accent)
		tm.Phase = PhaseFalling
		return
	}
	tm.shuffled = false
	tm.Phase = PhaseIdle
	if tm.moving {
		tm.moving = false
		if tm.OnMoveDone != nil {
			tm.OnMoveDone()
		}
	}
}

// showStep turns a cascade step into vanishing sprites, flashes, and score popups.
func (tm *TileManager) showStep(res *board.StepResult) {
	types := tm.gemTypes()
	var sum rl.Vector2
	for _, c := range res.Cleared {
		center := tm.pointCenter(c.Pos)
		sum.X += center.X
		sum.Y += center.Y
		if t, ok := tm.Tiles[c.Gem.ID]; ok {
			delete(tm.Tiles, c.Gem.ID)
			t.Dying = true
			tm.Dying = append(tm.Dying, t)
		}
		col := rl.White
		if c.Gem.Color >= 0 && len(types) > 0 {
			col = types[c.Gem.Color%len(types)].Color
		}
		tm.Effects.Burst(center, col, 6, tm.CellSize*4)
	}

	r := tm.Rect()
	for _, b := range res.Blasts {
		c := tm.pointCenter(b.Pos)
		f := flash{kind: b.Special, center: c, life: 0.4}
		half := tm.CellSize / 2
		if b.Wide {
			half *= 3
		}
		switch b.Special {
		case board.BoltH:
			f.rect = rl.Rectangle{X: r.X, Y: c.Y - half, Width: r.Width, Height: half * 2}
		case board.BoltV:
			f.rect = rl.Rectangle{X: c.X - half, Y: r.Y, Width: half * 2, Height: r.Height}
		case board.Bomb:
			f.radius = tm.CellSize * 1.6
			if b.Wide {
				f.radius *= 1.7
			}
		case board.Prism:
			f.radius = max(r.Width, r.Height) * 0.7
			f.life = 0.7
		}
		tm.Effects.Flash(f)
	}

	n := float32(len(res.Cleared))
	mid := rl.Vector2{X: sum.X / n, Y: sum.Y / n}
	if !tm.Duel {
		tm.Effects.Popup(text.Tf("ui.plus", "+%d", res.Points), mid, 26, rl.White)
	}
	if res.Chain >= 2 {
		tm.Effects.Popup(text.Tf("ui.chain_popup", "Chain x%d!", res.Chain), rl.Vector2{X: r.X + r.Width/2, Y: r.Y - 10}, 30+float32(min(res.Chain, 8))*3, ui.Accent)
	}
	for _, p := range res.Created {
		if g := tm.Board.At(p); g != nil {
			tm.Effects.Popup(text.Tf("ui.special_popup", "%s!", trSpecial(g.Special)), tm.pointCenter(p), 24, ui.Accent)
		}
	}
	if len(res.Created) >= 2 {
		tm.Effects.Popup(text.Tf("ui.synchro_popup", "Synchro x%d!", len(res.Created)), rl.Vector2{X: r.X + r.Width/2, Y: r.Y + r.Height/2}, 44, rl.SkyBlue)
	}
}

func (tm *TileManager) handleInput() {
	mouse := rl.GetMousePosition()
	p, inside := tm.CellAt(mouse)

	if rl.IsMouseButtonPressed(rl.MouseRightButton) && inside && !tm.Duel {
		tm.ToggleLock(p)
		return
	}

	if tm.Brush != nil {
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && inside {
			tm.pushUndo()
		}
		if rl.IsMouseButtonDown(rl.MouseLeftButton) && inside {
			tm.Board.Paint(p, tm.Brush.Color, tm.Brush.Special)
		}
		return
	}

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		if !inside {
			tm.selected = nil
			return
		}
		if tm.selected != nil && *tm.selected != p && tm.Board.Adjacent(*tm.selected, p) {
			tm.trySwap(*tm.selected, p)
			return
		}
		tm.selected = &p
		tm.dragFrom = &p
	}

	if tm.dragFrom != nil && rl.IsMouseButtonDown(rl.MouseLeftButton) {
		from := *tm.dragFrom
		c := tm.pointCenter(from)
		dx, dy := (mouse.X-c.X)/tm.CellSize, (mouse.Y-c.Y)/tm.CellSize
		if math.Abs(float64(dx)) > 0.55 || math.Abs(float64(dy)) > 0.55 {
			step := board.Point{}
			diagonal := tm.Board.Settings.DiagonalSwap && math.Abs(float64(dx)) > 0.4 && math.Abs(float64(dy)) > 0.4
			if diagonal || math.Abs(float64(dx)) >= math.Abs(float64(dy)) {
				step.X = sign(dx)
			}
			if diagonal || math.Abs(float64(dy)) > math.Abs(float64(dx)) {
				step.Y = sign(dy)
			}
			tm.dragFrom = nil
			if to := from.Add(step); tm.Board.In(to) {
				tm.trySwap(from, to)
			}
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		tm.dragFrom = nil
	}
}

func (tm *TileManager) trySwap(a, c board.Point) {
	tm.selected, tm.dragFrom = nil, nil
	if !tm.Board.CanSwap(a, c) {
		for _, p := range []board.Point{a, c} {
			if g := tm.Board.At(p); g != nil && g.Locked {
				tm.Effects.Popup(text.T("ui.locked", "Locked"), tm.pointCenter(p), 22, rl.SkyBlue)
			}
		}
		return
	}
	tm.pushUndo()
	tm.hint = nil
	tm.swapA, tm.swapB = a, c
	tm.swapValid = tm.Board.IsValidSwap(a, c)
	tm.Board.Swap(a, c)
	tm.Phase = PhaseSwapping
}

// ToggleLock locks or releases one gem. Releasing can set off matches it was holding back.
func (tm *TileManager) ToggleLock(p board.Point) {
	if tm.Busy() {
		return
	}
	tm.pushUndo()
	if !tm.Board.ToggleLock(p) {
		tm.dropUndo()
		tm.Effects.Popup(text.T("ui.no_locks", "No locks left"), tm.pointCenter(p), 22, rl.SkyBlue)
		return
	}
	if g := tm.Board.At(p); g != nil && !g.Locked && tm.Brush == nil && tm.Board.HasMatches() {
		tm.Board.StartChain()
		tm.Phase = PhaseResolving
	}
}

// ReleaseLocks unlocks everything at once so every held-back match fires in the same step.
func (tm *TileManager) ReleaseLocks() {
	if tm.Busy() || tm.Board.LockCount() == 0 {
		return
	}
	tm.pushUndo()
	n := tm.Board.ReleaseAll()
	r := tm.Rect()
	tm.Effects.Popup(text.Tf("ui.released", "Released %d", n), rl.Vector2{X: r.X + r.Width/2, Y: r.Y + r.Height + 8}, 26, rl.SkyBlue)
	tm.Board.StartChain()
	tm.Phase = PhaseResolving
}

// Resolve clears any matches sitting on the board, e.g. after painting in the sandbox.
func (tm *TileManager) Resolve() {
	if tm.Busy() || !tm.Board.HasMatches() {
		return
	}
	tm.pushUndo()
	tm.Board.StartChain()
	tm.Phase = PhaseResolving
}

func (tm *TileManager) ShowHint() {
	if tm.Busy() || tm.Paused {
		return
	}
	if a, c, ok := tm.Board.FindMove(); ok {
		tm.hint = []board.Point{a, c}
		tm.hintTimer = 3
	}
}

// RunMove resolves the board as a full move, for changes made outside a swap such as a spell.
// OnMoveDone fires when it settles, even if nothing ended up clearing.
func (tm *TileManager) RunMove() {
	tm.Board.StartChain()
	tm.moving = true
	tm.Phase = PhaseResolving
}

// ShowMove highlights a swap about to be played, so an opponent's move can be followed.
func (tm *TileManager) ShowMove(a, c board.Point) {
	tm.hint = []board.Point{a, c}
	tm.hintTimer = 0.6
}

// PlaySwap makes a swap without player input.
func (tm *TileManager) PlaySwap(a, c board.Point) {
	tm.trySwap(a, c)
}

// updateIdle plays one gem's strip after the board has been still, then waits again.
func (tm *TileManager) updateIdle(dt float64) {
	active := tm.Phase != PhaseIdle || !tm.settled || len(tm.Dying) > 0
	if active {
		tm.quiet = 0
		tm.idleID = 0
		return
	}
	if tm.idleID != 0 {
		if _, ok := tm.Tiles[tm.idleID]; !ok || tm.time-tm.idleStart >= tm.idleDur {
			tm.lastIdle = tm.idleID
			tm.idleID = 0
			tm.quiet = 0
		}
		return
	}
	tm.quiet += dt
	if tm.quiet < IDLE_QUIET {
		return
	}
	if !tm.beginIdle() {
		tm.quiet = 0
	}
}

// idleIDs are the tiles whose type has an idle strip to play.
func (tm *TileManager) idleIDs() []int {
	var ids []int
	for id, t := range tm.Tiles {
		if t.Gem.Special == board.Prism || t.Gem.Special == board.Wild {
			continue
		}
		if _, _, ok := idleClip(tm.gemType(t.Gem), t.Gem.Special == board.Bomb); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// beginIdle starts one gem that has a strip, preferring a gem in a move the player can make so
// the idle doubles as a hint once the board has been quiet. It reports whether one started.
func (tm *TileManager) beginIdle() bool {
	ids := tm.idleIDs()
	if tm.Board != nil {
		if a, c, ok := tm.Board.FindMove(); ok {
			var moves []int
			for _, p := range []board.Point{a, c} {
				g := tm.Board.At(p)
				if g == nil {
					continue
				}
				for _, id := range ids {
					if id == g.ID {
						moves = append(moves, id)
						break
					}
				}
			}
			if len(moves) > 0 {
				ids = moves
			}
		}
	}
	if len(ids) == 0 {
		return false
	}
	pool := ids
	if len(ids) > 1 {
		var rest []int
		for _, id := range ids {
			if id != tm.lastIdle {
				rest = append(rest, id)
			}
		}
		if len(rest) > 0 {
			pool = rest
		}
	}
	id := pool[rand.Intn(len(pool))]
	t := tm.Tiles[id]
	frames, fps, _ := idleClip(tm.gemType(t.Gem), t.Gem.Special == board.Bomb)
	tm.idleID = id
	tm.idleStart = tm.time
	tm.idleDur = float64(frames) / float64(fps)
	return true
}

// gemType is the palette entry for a gem, or nil for a colorless one.
func (tm *TileManager) gemType(g board.Gem) *entities.GemType {
	types := tm.gemTypes()
	if g.Color < 0 || len(types) == 0 {
		return nil
	}
	return types[g.Color%len(types)]
}

// idleClip is the strip a one-shot idle plays. A bomb uses the cracked strip when it has one.
func idleClip(gt *entities.GemType, bomb bool) (frames int, fps float32, ok bool) {
	if gt == nil {
		return 0, 0, false
	}
	frames, fps = gt.IdleFrames, gt.IdleFPS
	id := gt.Idle.ID
	if bomb && gt.BombIdle.ID != 0 {
		frames, fps, id = gt.BombIdleFrames, gt.BombIdleFPS, gt.BombIdle.ID
	}
	if id == 0 || frames < 2 {
		return 0, 0, false
	}
	if fps <= 0 {
		fps = 8
	}
	return frames, fps, true
}

func (tm *TileManager) gemTypes() []*entities.GemType {
	if tm.Types != nil {
		return tm.Types()
	}
	return tm.DataManager.GetAllGemTypes()
}

func (tm *TileManager) pushUndo() {
	if !tm.UndoEnabled {
		return
	}
	tm.undo = append(tm.undo, tm.Board.Snapshot())
	if len(tm.undo) > utils.UNDO_LIMIT {
		tm.undo = tm.undo[1:]
	}
}

func (tm *TileManager) dropUndo() {
	if len(tm.undo) > 0 {
		tm.undo = tm.undo[:len(tm.undo)-1]
	}
}

func (tm *TileManager) CanUndo() bool { return tm.UndoEnabled && len(tm.undo) > 0 && !tm.Busy() }

func (tm *TileManager) Undo() {
	if !tm.CanUndo() {
		return
	}
	snap := tm.undo[len(tm.undo)-1]
	tm.undo = tm.undo[:len(tm.undo)-1]
	tm.Board.Restore(snap)
	tm.selected, tm.hint = nil, nil
}

func (tm *TileManager) ClearSelection() bool {
	had := tm.selected != nil
	tm.selected, tm.dragFrom = nil, nil
	return had
}

func (tm *TileManager) Draw() {
	off := tm.Effects.Offset()
	r := tm.Rect()
	r.X += off.X
	r.Y += off.Y

	frame := rl.Rectangle{X: r.X - 10, Y: r.Y - 10, Width: r.Width + 20, Height: r.Height + 20}
	rl.DrawRectangleRounded(frame, 0.03, 8, rl.Color{R: 10, G: 8, B: 22, A: 200})
	rl.DrawRectangleRoundedLines(frame, 0.03, 8, 2, ui.PanelEdge)
	for y := 0; y < tm.Board.H(); y++ {
		for x := 0; x < tm.Board.W(); x++ {
			col := rl.Color{R: 255, G: 255, B: 255, A: 10}
			if (x+y)%2 == 0 {
				col.A = 22
			}
			rl.DrawRectangleRec(tm.cellRect(board.Point{X: x, Y: y}, off), col)
		}
	}

	if tm.Phase == PhaseIdle && !tm.Paused {
		if p, ok := tm.CellAt(rl.GetMousePosition()); ok {
			rl.DrawRectangleRec(tm.cellRect(p, off), rl.Fade(rl.White, 0.08))
		}
	}
	if tm.selected != nil {
		pulse := float32(0.6 + 0.4*math.Sin(tm.time*8))
		rl.DrawRectangleLinesEx(tm.cellRect(*tm.selected, off), 3, rl.Fade(ui.Accent, pulse))
	}
	if tm.hintTimer > 0 {
		pulse := float32(0.5 + 0.5*math.Sin(tm.time*10))
		for _, p := range tm.hint {
			rl.DrawRectangleLinesEx(tm.cellRect(p, off), 4, rl.Fade(rl.Lime, pulse))
		}
	}

	// gems spawn off the top (or whichever side gravity pulls from), so clip to the board
	rl.BeginScissorMode(int32(r.X), int32(r.Y), int32(r.Width), int32(r.Height))
	for _, t := range tm.Tiles {
		t.Draw(tm, off)
	}
	for _, t := range tm.Dying {
		t.Draw(tm, off)
	}
	tm.Effects.DrawFlashes(off)
	rl.EndScissorMode()

	tm.drawGravity(r)
	tm.Effects.Draw()
}

func (tm *TileManager) cellRect(p board.Point, off rl.Vector2) rl.Rectangle {
	return rl.Rectangle{
		X:      tm.ScreenX + float32(p.X)*tm.CellSize + off.X,
		Y:      tm.ScreenY + float32(p.Y)*tm.CellSize + off.Y,
		Width:  tm.CellSize,
		Height: tm.CellSize,
	}
}

// drawGravity puts a small arrow outside the board showing which way gems fall, when it isn't the usual down.
func (tm *TileManager) drawGravity(r rl.Rectangle) {
	d := tm.Board.Settings.Gravity
	if d == board.Down && !tm.Board.Settings.ChaosGravity {
		return
	}
	v := d.Vec()
	c := rl.Vector2{X: r.X + r.Width/2 + float32(v.X)*(r.Width/2+30), Y: r.Y + r.Height/2 + float32(v.Y)*(r.Height/2+30)}
	tip := rl.Vector2{X: c.X + float32(v.X)*12, Y: c.Y + float32(v.Y)*12}
	side := rl.Vector2{X: float32(-v.Y) * 12, Y: float32(v.X) * 12}
	back := rl.Vector2{X: c.X - float32(v.X)*12, Y: c.Y - float32(v.Y)*12}
	a := rl.Vector2{X: back.X + side.X, Y: back.Y + side.Y}
	b := rl.Vector2{X: back.X - side.X, Y: back.Y - side.Y}
	rl.DrawTriangle(tip, a, b, ui.Accent)
	rl.DrawTriangle(tip, b, a, ui.Accent)
}

func sign(f float32) int {
	if f < 0 {
		return -1
	}
	return 1
}
