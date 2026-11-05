package core

import (
	"testing"

	"jewel-game/board"
	"jewel-game/entities"

	"jewel-game/rl"
)

func idleFixture(gems ...board.Gem) *TileManager {
	strip := rl.Texture2D{ID: 1}
	gt := &entities.GemType{Idle: strip, IdleFrames: 8, IdleFPS: 8}
	tm := &TileManager{
		Phase:   PhaseIdle,
		settled: true,
		Tiles:   map[int]*Tile{},
		Types:   func() []*entities.GemType { return []*entities.GemType{gt} },
		time:    10,
	}
	for _, g := range gems {
		tm.Tiles[g.ID] = &Tile{ID: g.ID, Gem: g}
	}
	return tm
}

func TestIdleWaitsForAQuietBoard(t *testing.T) {
	tm := idleFixture(board.Gem{ID: 1})
	tm.updateIdle(IDLE_QUIET - 0.1)
	if tm.idleID != 0 {
		t.Fatalf("played after %v seconds, want a wait of %v", IDLE_QUIET-0.1, IDLE_QUIET)
	}
	tm.updateIdle(0.1)
	if tm.idleID != 1 {
		t.Fatalf("idle gem %d, want 1", tm.idleID)
	}
	if tm.idleDur != 1 {
		t.Fatalf("duration %v, want 1 second for 8 frames at 8 fps", tm.idleDur)
	}
}

func TestIdlePlaysOnceThenWaitsAgain(t *testing.T) {
	tm := idleFixture(board.Gem{ID: 1})
	tm.updateIdle(IDLE_QUIET)
	tm.time += tm.idleDur - 0.01
	tm.updateIdle(0.01)
	if tm.idleID != 1 {
		t.Fatal("animation ended early")
	}
	tm.time += 0.02
	tm.updateIdle(0.02)
	if tm.idleID != 0 || tm.quiet != 0 {
		t.Fatalf("after one loop id=%d quiet=%v", tm.idleID, tm.quiet)
	}
	tm.updateIdle(IDLE_QUIET - 0.1)
	if tm.idleID != 0 {
		t.Fatal("next gem started before the quiet gap")
	}
}

func TestActivityCancelsIdle(t *testing.T) {
	tm := idleFixture(board.Gem{ID: 1})
	tm.updateIdle(IDLE_QUIET)
	tm.Phase = PhaseSwapping
	tm.updateIdle(0.1)
	if tm.idleID != 0 || tm.quiet != 0 {
		t.Fatalf("swap left id=%d quiet=%v", tm.idleID, tm.quiet)
	}
}

func TestIdleSkipsGemsWithoutAStrip(t *testing.T) {
	tm := idleFixture(board.Gem{ID: 1, Special: board.Prism})
	tm.updateIdle(IDLE_QUIET)
	if tm.idleID != 0 || tm.quiet != 0 {
		t.Fatalf("prism played id=%d quiet=%v", tm.idleID, tm.quiet)
	}
}
