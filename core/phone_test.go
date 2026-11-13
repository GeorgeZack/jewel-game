package core

import (
	"jewel-game/board"
	"testing"
)

// PHONE_TAP_PX is the smallest gem worth tapping, in screen pixels.
const PHONE_TAP_PX = 36

// On a 420x860 phone every board a card may ask for, up to 10x10, keeps gems a finger wide.
func TestPhoneGemsStayTappable(t *testing.T) {
	for _, b := range [][2]int{{8, 8}, {9, 9}, {10, 8}, {10, 10}} {
		tm := &TileManager{Board: board.NewBoard(board.Settings{Width: b[0], Height: b[1], NumColors: 7}, 1)}
		tm.Layout(phoneBoardRect(420, 860, b[0], b[1]))
		t.Logf("%dx%d board: %.0f px gems", b[0], b[1], tm.CellSize)
		if tm.CellSize < PHONE_TAP_PX {
			t.Errorf("a %dx%d board has %.0f px gems on a 420x860 phone", b[0], b[1], tm.CellSize)
		}
	}
}

func TestNarrowSize(t *testing.T) {
	if !narrowSize(390, 844) || !narrowSize(844, 390) {
		t.Fatal("a phone, upright or sideways, is narrow")
	}
	if narrowSize(1400, 900) {
		t.Fatal("a desktop window keeps the three columns")
	}
}

func TestPhoneBoardFits(t *testing.T) {
	for _, sz := range [][2]float32{{390, 844}, {844, 390}, {360, 640}, {700, 400}} {
		r := phoneBoardRect(sz[0], sz[1], 8, 8)
		if r.X < 0 || r.Y < 0 || r.X+r.Width > sz[0]+1 || r.Y+r.Height > sz[1]+1 {
			t.Fatalf("board %v falls outside %v", r, sz)
		}
		if r.Width < 48 || r.Height != r.Width {
			t.Fatalf("board %v in %v is not a tappable square", r, sz)
		}
		// other boards keep their shape inside the same space
		for _, b := range [][2]int{{10, 10}, {10, 6}, {6, 10}, {9, 9}} {
			o := phoneBoardRect(sz[0], sz[1], b[0], b[1])
			if o.X < 0 || o.Y < 0 || o.X+o.Width > sz[0]+1 || o.Y+o.Height > sz[1]+1 {
				t.Fatalf("a %v board %v falls outside %v", b, o, sz)
			}
			if cw, ch := o.Width/float32(b[0]), o.Height/float32(b[1]); cw-ch > 0.01 || ch-cw > 0.01 {
				t.Fatalf("a %v board %v has cells %.1f by %.1f", b, o, cw, ch)
			}
		}
	}
}

func TestUIScalePhone(t *testing.T) {
	if uiScaleFor(390, 844) != 1 || uiScaleFor(844, 390) != 1 {
		t.Fatal("phone text is laid out in screen pixels, so the scale stays 1")
	}
	if uiScaleFor(1600, 1000) < 1 {
		t.Fatal("a large desktop window should not shrink the panels")
	}
}
