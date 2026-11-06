package core

import (
	"os"
	"testing"

	"jewel-game/rpg"

	"jewel-game/rl"
)

func TestHexLocationsKeepTheirOwnCell(t *testing.T) {
	if _, err := rpg.LoadStory(storyFiles{}, "../assets/story"); err != nil {
		t.Fatal(err)
	}
	m := seedMap()
	seen := map[[2]int]string{}
	for i := range rpg.Locations {
		l := &rpg.Locations[i]
		c, r := locationCell(m, l)
		if other, ok := seen[[2]int{c, r}]; ok {
			t.Fatalf("%s and %s share hex %d,%d", l.ID, other, c, r)
		}
		seen[[2]int{c, r}] = l.ID
		ref, ok := m.at(c, r)
		if !ok {
			t.Fatalf("%s cell missing", l.ID)
		}
		if ref != hexBiomes[l.ID].mark {
			t.Fatalf("%s mark %+v, got %+v", l.ID, hexBiomes[l.ID].mark, ref)
		}
	}
}

func TestHexMapRoundTrip(t *testing.T) {
	m := seedMap()
	m.set(1, 2, hx(hexSheetLava, 4, 0))
	path := t.TempDir() + "/world.yaml"
	if err := writeHexMap(path, m); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseHexMap(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.cols != m.cols || got.rows != m.rows || len(got.tiles) != len(m.tiles) {
		t.Fatalf("size %+v vs %+v", got.cols, m.cols)
	}
	ref, _ := got.at(1, 2)
	if ref != hx(hexSheetLava, 4, 0) {
		t.Fatalf("paint %+v", ref)
	}
}

func TestRoadBendRoundTrip(t *testing.T) {
	m := seedMap()
	m.setNode("mossgate", norm{0.3, 0.5})
	m.insertBend("emberfield", "mossgate", norm{0.2, 0.7}, 0)
	m.insertBend("mossgate", "emberfield", norm{0.25, 0.6}, 0)
	// second insert is from mossgate toward emberfield, at the start of that direction,
	// so it sits nearest mossgate: emberfield, 0.2, 0.25, mossgate
	got := m.roadBetween("emberfield", "mossgate")
	if len(got) != 2 || got[0] != (norm{0.2, 0.7}) || got[1] != (norm{0.25, 0.6}) {
		t.Fatalf("via %+v", got)
	}
	m.moveBend("mossgate", "emberfield", 0, norm{0.28, 0.55})
	got = m.roadBetween("emberfield", "mossgate")
	if got[1] != (norm{0.28, 0.55}) {
		t.Fatalf("moved %+v", got)
	}
	m.deleteBend("emberfield", "mossgate", 0)
	if via := m.roadBetween("emberfield", "mossgate"); len(via) != 1 || via[0] != (norm{0.28, 0.55}) {
		t.Fatalf("deleted %+v", via)
	}
	path := t.TempDir() + "/world.yaml"
	if err := writeHexMap(path, m); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	back, err := parseHexMap(b)
	if err != nil {
		t.Fatal(err)
	}
	theMap = back
	mapLoaded = true
	if info, err := os.Stat(hexMapFile); err == nil {
		mapMod = info.ModTime()
	}
	if nodeNorm("mossgate") != (norm{0.3, 0.5}) {
		t.Fatalf("node %+v", nodeNorm("mossgate"))
	}
	if via := back.roadBetween("emberfield", "mossgate"); len(via) != 1 || via[0].x != 0.28 {
		t.Fatalf("saved via %+v", via)
	}
	mid := pointAlong([]rl.Vector2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}}, 0.5)
	if mid.X < 9.9 || mid.Y > 0.2 {
		t.Fatalf("midpoint %+v", mid)
	}
	theMap = nil
	mapLoaded = false
}

func TestFitRectKeepsShape(t *testing.T) {
	// A wide picture in a tall frame is limited by the width and stays centered.
	got := fitRect(rl.Rectangle{X: 10, Y: 20, Width: 200, Height: 400}, 2)
	if got.Width != 200 || got.Height != 100 || got.X != 10 || got.Y != 170 {
		t.Fatalf("wide picture %+v", got)
	}
	// A tall picture in a wide frame is limited by the height.
	got = fitRect(rl.Rectangle{Width: 400, Height: 100}, 0.5)
	if got.Width != 50 || got.Height != 100 || got.X != 175 {
		t.Fatalf("tall picture %+v", got)
	}
}

func TestShortMapFile(t *testing.T) {
	m, err := parseHexMap([]byte("background: assets/maps/background.png\nnodes:\n  mossgate: [0.4, 0.5]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m.background != "assets/maps/background.png" {
		t.Fatalf("background %q", m.background)
	}
	if m.nodes["mossgate"] != (norm{0.4, 0.5}) {
		t.Fatalf("node %+v", m.nodes["mossgate"])
	}
	if _, err := parseHexMap([]byte("cols: 2\nrows: 2\ntiles:\n  - [0, 0, 0]\n")); err == nil {
		t.Fatal("partial tile grid was accepted")
	}
	saved, err := os.ReadFile("../assets/maps/world.yaml")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseHexMap(saved)
	if err != nil {
		t.Fatal(err)
	}
	if got.background != "assets/maps/background.png" {
		t.Fatalf("saved background %q", got.background)
	}
	if _, ok := got.nodes["emberfield"]; !ok {
		t.Fatal("saved places were dropped")
	}
}

func TestHexRefsStayOnSheet(t *testing.T) {
	if len(hexBiomes) != 12 {
		t.Fatalf("biomes %d", len(hexBiomes))
	}
	for id, bio := range hexBiomes {
		refs := append([]hexRef{}, bio.ground...)
		refs = append(refs, bio.mark)
		if len(bio.ground) == 0 {
			t.Fatalf("%s has no ground", id)
		}
		for _, ref := range refs {
			if ref.sheet < 0 || ref.sheet >= len(hexSheetPaths) || ref.c < 0 || ref.c > 6 || ref.r < 0 || ref.r > 6 {
				t.Fatalf("%s ref %+v", id, ref)
			}
		}
	}
}
