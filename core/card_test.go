package core

import (
	"testing"

	"jewel-game/board"
	"jewel-game/rpg"
)

// A foe's rule card has to reach the board the duel is played on, through startDuel.
func TestFoeCardReachesTheBoard(t *testing.T) {
	if _, err := rpg.LoadStory(storyFiles{}, "../assets/story"); err != nil {
		t.Fatal(err)
	}
	defer rpg.LoadStory(storyFiles{}, "../assets/story")
	rpg.RuleCards = append(rpg.RuleCards, rpg.RuleCard{ID: "test", Width: 6, Height: 10, Gravity: "up", Overcharge: true})
	rpg.FoeByID("rat").Card = "test"

	v := &QuestView{TM: &TileManager{}, Hero: rpg.NewHero("Knight")}
	v.startDuel("rat", 1, rpg.Story.Start)
	s := v.TM.Board.Settings
	if v.TM.Board.W() != 6 || v.TM.Board.H() != 10 || s.Gravity != board.Up || !s.Overcharge || !s.Duel {
		t.Fatalf("card didn't reach the board: %+v", s)
	}
	if v.Duel.Board != v.TM.Board || v.Duel.Card == nil || v.Duel.Card.ID != "test" {
		t.Fatalf("duel isn't on the card's board")
	}

	v.startDuel("goblin", 1, rpg.Story.Start)
	if got, want := v.TM.Board.Settings, rpg.BoardSettings(nil, false); got != want || got.Width != 8 || got.Height != 8 {
		t.Fatalf("a foe with no card should get the standard 8x8 duel, got %+v", got)
	}
}
