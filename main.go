package main

import (
	"jewel-game/core"

	"github.com/gen2brain/raylib-go/raylib"
)

func main() {
	screenWidth := 1916
	screenHeight := 1058
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(int32(screenWidth), int32(screenHeight), "Jewel Game")
	rl.SetWindowSize(screenWidth, screenHeight)
	rl.SetWindowPosition(0, 0)
	rl.SetTargetFPS(60)
	rl.SetExitKey(0)	// Esc first drops the brush or selection, the controller decides when to quit
	defer rl.CloseWindow()

	gc := core.NewGameController()
	defer gc.Unload()

	for !rl.WindowShouldClose() && !gc.Quit {
		gc.Update()
		gc.Draw()
	}
}
