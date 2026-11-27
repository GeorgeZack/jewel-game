package core

import (
	"testing"

	"jewel-game/rpg"
)

// A missing foe must not start a duel or panic, including after a hot-reload leaves a dangling id.
func TestStartDuelUnknownFoeDoesNotPanic(t *testing.T) {
	if _, err := rpg.LoadStory(storyFiles{}, "../assets/story"); err != nil {
		t.Fatal(err)
	}
	v := &QuestView{TM: &TileManager{}, Hero: rpg.NewHero("Knight")}
	v.startDuel("no_such_foe", 1, rpg.Story.Start, "")
	if v.Duel != nil {
		t.Fatalf("unknown foe started a duel")
	}
	if v.Screen == ScreenDuel {
		t.Fatalf("unknown foe switched to the duel screen")
	}
	if foeName("no_such_foe") != "no_such_foe" {
		t.Fatalf("missing foe should fall back to its id")
	}
	if f := rpg.FoeByID("rat"); f == nil || foeName("rat") != f.Name {
		t.Fatalf("a known foe should keep its name")
	}
}
