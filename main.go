package main

import (
	"jewel-game/core"

	"github.com/gen2brain/raylib-go/raylib"
)

func main() {
	screenWidth := 1916
	screenHeight := 1058
	rl.InitWindow(int32(screenWidth), int32(screenHeight), "Jewel Game")
	rl.SetWindowSize(screenWidth, screenHeight)
	rl.SetWindowPosition(0, 0)
	rl.SetTargetFPS(60)
	defer rl.CloseWindow()

	gc := core.NewGameController()
	defer gc.Unload()

	for !rl.WindowShouldClose() {
		gc.Update()
		gc.Draw()
	}
}
