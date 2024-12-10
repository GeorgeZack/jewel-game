package main

import (
	"image/color"
	"jewel-game/logic"
	"jewel-game/types"

	"github.com/gen2brain/raylib-go/raylib"
)

var cellHovered *types.Cell
var cellStart *types.Cell
var cellEnd *types.Cell
var isMouseButtonPressed bool
var strRecent []string

func unloadGemTextures(gemTypes []types.GemType) {
	for _, g := range gemTypes {
		rl.UnloadTexture(g.Texture)
	}
}

func main() {
	screenWidth := 1916
	screenHeight := 1058
	rl.InitWindow(int32(screenWidth), int32(screenHeight), "jewel game")
	rl.SetWindowSize(screenWidth, screenHeight)
	rl.SetWindowPosition(0, 0)
	rl.SetTargetFPS(60)
	bkg := rl.LoadTexture("res/bkg.jpg")
	defer rl.CloseWindow()

	gemTypes := logic.InitGemTypes()
	grid := logic.InitGrid(gemTypes)
	defer unloadGemTextures(gemTypes)

	globals := logic.InitGlobals()

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		rl.DrawTexture(bkg, 0, -1200, color.RGBA{ 255, 255, 255, 205 })
		logic.GameLoop(&grid, &globals)
		rl.EndDrawing()
	}
}

