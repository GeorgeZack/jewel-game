package main

import (
	"image/color"
	"jewel-game/states"
	"jewel-game/utils"
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

	dm := utils.NewDataManager()
	defer rl.UnloadFont(dm.Consts.DebugFont)

	gemTypes := states.InitGemTypes()
	grid := states.InitGrid(gemTypes, dm)
	defer ui.UnloadGemTextures(gemTypes)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		rl.DrawTexture(bkg, 0, -1200, color.RGBA{ 255, 255, 255, 205 })
		// other game states go here, like Menu() or Score()
		states.GameLoop(&grid, dm)
		rl.EndDrawing()
	}
}

