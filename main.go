package main

import (
	"image/color"
	"jewel-game/logic"
	"jewel-game/ui"

	"github.com/gen2brain/raylib-go/raylib"
)

func main() {
	screenWidth := 1916
	screenHeight := 1058
	rl.InitWindow(int32(screenWidth), int32(screenHeight), "Jewel Game")
	rl.SetWindowSize(screenWidth, screenHeight)
	rl.SetWindowPosition(0, 0)
	rl.SetTargetFPS(60)
	bkg := rl.LoadTexture("res/bkg.jpg")
	defer rl.CloseWindow()

	gemTypes := logic.InitGemTypes()
	grid := logic.InitGrid(gemTypes)
	defer ui.UnloadGemTextures(gemTypes)

	states := logic.InitStates()
	states.MonoSpaceFont = rl.LoadFontEx("res/NotoMono-Regular.ttf", 32, nil)
	defer rl.UnloadFont(states.MonoSpaceFont)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		rl.DrawTexture(bkg, 0, -1200, color.RGBA{ 255, 255, 255, 205 })
		logic.GameLoop(&grid, &states)
		rl.EndDrawing()
	}
}

