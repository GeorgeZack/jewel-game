package core

import (
	"math"

	"jewel-game/board"
	"jewel-game/rl"
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
)

// startPuzzle opens a puzzle board. Leaving it, won or lost, goes back to the world map.
func (v *QuestView) startPuzzle(id string) {
	pz := rpg.PuzzleByID(id)
	if pz == nil {
		ui.AddMessage("%s", text.Tf("log.no_puzzle", "Story: no puzzle %q", id))
		return
	}
	run, err := rpg.NewPuzzleRun(pz, v.Hero)
	if err != nil {
		ui.AddMessage("%s", text.Tf("log.puzzle_broken", "Puzzle %s is broken: %v", id, err))
		return
	}
	v.Puzzle, v.puzzleTool, v.puzzleShown = run, "", 0
	v.TM.Types = v.DM.PuzzleGemTypes
	v.TM.NoShuffle = true
	v.TM.Load(run.Board)
	v.Screen = ScreenPuzzle
}

func (v *QuestView) leavePuzzle() {
	v.Puzzle = nil
	v.TM.Types = v.DM.DuelGemTypes
	v.TM.NoShuffle = false
	v.Screen = ScreenWorld
}

func (v *QuestView) updatePuzzle() {
	r := v.Puzzle
	v.TM.Paused = v.puzzleTool != "" || r.State != rpg.PuzzlePlaying
	if v.TM.Busy() || r.State != rpg.PuzzlePlaying {
		return
	}
	// a Force waits for the gem to land on
	if v.puzzleTool != "" && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		if p, ok := v.TM.CellAt(rl.GetMousePosition()); ok && r.ApplyTool(v.puzzleTool, p) {
			v.TM.RunMove()
		}
		v.puzzleTool = ""
		return
	}
	// dev: replay a stored solution in the running game
	if moves := devPuzzleSolution(r.P); v.puzzleShown < len(moves) {
		m, _ := rpg.ParseMove(moves[v.puzzleShown])
		v.puzzleShown++
		if m.Tool == "" {
			v.TM.PlaySwap(m.A, m.C)
		} else if r.ApplyTool(m.Tool, m.A) {
			v.TM.RunMove()
		}
	}
}

// useTool runs a tool from its button: a convert at once, a Force once a gem is picked.
func (v *QuestView) useTool(t *rpg.PuzzleTool) {
	r := v.Puzzle
	if v.TM.Busy() || r.Uses[t.ID] <= 0 {
		return
	}
	if t.Kind == "force" {
		v.puzzleTool = t.ID
		return
	}
	if r.ApplyTool(t.ID, board.Point{}) {
		v.TM.RunMove()
	}
}

// puzzleDone settles the run once a move has resolved, and shows the result when it's over.
func (v *QuestView) puzzleDone() {
	r := v.Puzzle
	r.Settle()
	switch r.State {
	case rpg.PuzzleWon:
		r.Finish(v.Hero)
		v.save()
		v.show(Modal{
			Title:   text.T("ui.puzzle_won", "Solved!"),
			Body:    text.Tf("ui.puzzle_route", "%s\n\nRoute: %s, in %d moves.", r.P.Name, r.Route.Name, r.Moves()),
			Buttons: []ModalButton{{text.T("ui.continue", "Continue"), v.leavePuzzle}},
		})
	case rpg.PuzzleLost:
		why := map[string]string{
			"moves":  text.T("ui.puzzle_lost_moves", "You ran out of moves."),
			"stuck":  text.T("ui.puzzle_lost_stuck", "No moves are left on the board."),
			"fail":   text.T("ui.puzzle_lost_fail", "That set off what it shouldn't have."),
			"jammed": text.T("ui.puzzle_lost_jammed", "It's done, but not in any way that works: the lock jammed."),
		}[r.Lost]
		id := r.P.ID
		v.show(Modal{
			Title: text.T("ui.puzzle_lost", "Not this time"),
			Body:  why + "\n\n" + text.T("ui.puzzle_retry", "Trying again costs nothing."),
			Buttons: []ModalButton{
				{text.T("ui.try_again", "Try again"), func() { v.startPuzzle(id) }},
				{text.T("ui.leave", "Leave"), v.leavePuzzle},
			},
		})
	}
}

// drawPuzzleInfo is the objective and moves left: the right panel, or the phone's top panel.
func (v *QuestView) drawPuzzleInfo(col *ui.Column) {
	r := v.Puzzle
	col.Heading(r.P.Name)
	col.Paragraph(r.P.Objective, 17, ui.TextColor)
	col.Line(text.Tf("ui.puzzle_moves", "Moves left  %d / %d", max(0, r.P.Moves-r.Moves()), r.P.Moves), 20, ui.Accent)
}

// drawPuzzleTools is the tools with their uses: the left panel, or the phone's dock.
func (v *QuestView) drawPuzzleTools(col *ui.Column) {
	r := v.Puzzle
	col.Heading(text.T("ui.tools", "Tools"))
	any := false
	for i := range r.P.Tools {
		t := &r.P.Tools[i]
		if !r.HasTool(t) {
			continue
		}
		any = true
		label := text.Tf("ui.tool_uses", "%s  (%d)", t.Name, r.Uses[t.ID])
		ok := r.Uses[t.ID] > 0 && !v.TM.Busy() && r.State == rpg.PuzzlePlaying
		if ui.ButtonIf(col.Next(38), label, ok) {
			v.useTool(t)
		}
	}
	if !any {
		col.Paragraph(text.T("ui.no_tools", "You know no rite that helps here."), 15, ui.DimText)
	}
	if v.puzzleTool != "" {
		col.Paragraph(text.T("ui.pick_gem", "Pick the gem to Force."), 16, ui.Accent)
	}
	if ui.ButtonIf(col.Next(34), text.T("ui.leave", "Leave"), !v.TM.Busy()) {
		v.leavePuzzle()
	}
}

// drawPuzzleMarks rings the marked gems, so the objective's pieces stand out.
func (v *QuestView) drawPuzzleMarks() {
	r := v.Puzzle
	if r == nil || v.Screen != ScreenPuzzle {
		return
	}
	pulse := float32(0.6 + 0.4*math.Sin(v.time*4))
	for y := range r.Board.H() {
		for x := range r.Board.W() {
			p := board.Point{X: x, Y: y}
			if r.MarkAt(p) == "" {
				continue
			}
			t, ok := v.TM.Tiles[r.Board.At(p).ID]
			c := v.TM.pointCenter(p)
			if ok {
				c = v.TM.CellCenter(t.Pos)
			}
			s := v.TM.CellSize
			rl.DrawRing(c, s*0.44, s*0.5, 0, 360, 32, rl.Fade(ui.Accent, pulse))
			ui.TextCenteredOutlined(r.MarkAt(p), c.X+s*0.32, c.Y-s*0.5, 16, ui.Accent)
		}
	}
}
