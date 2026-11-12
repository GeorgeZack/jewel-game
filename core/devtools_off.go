//go:build !dev

package core

import "jewel-game/rpg"

// devTools is off in the public build: no map editor, no hot reload.
const devTools = false

func (v *QuestView) devTour() {}

func (c *GameController) devTitle() {}

func devAutoplay() bool   { return false }
func devCard() string     { return "" }
func devPuzzle() string   { return "" }

func devPuzzleSolution(*rpg.Puzzle) []string { return nil }
func devHideEditor() bool { return false }
