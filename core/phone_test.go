package core

import "testing"

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
		r := phoneBoardRect(sz[0], sz[1])
		if r.X < 0 || r.Y < 0 || r.X+r.Width > sz[0]+1 || r.Y+r.Height > sz[1]+1 {
			t.Fatalf("board %v falls outside %v", r, sz)
		}
		if r.Width < 48 || r.Height != r.Width {
			t.Fatalf("board %v in %v is not a tappable square", r, sz)
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
