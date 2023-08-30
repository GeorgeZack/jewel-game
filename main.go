package main

import (
	"image/color"
	"math/rand"

	"github.com/gen2brain/raylib-go/raylib"
)

const CELL_WIDTH = 64
const GRID_LENGTH = 8

//TODO:
// Add grid with Cells
// Cells draw background and position self
// Cells fill self with gem given to them

type Cell struct {
	gem		Gem
	x		int
	y		int
	draw	func(int, int)
}

type GemType struct {
	color		color.RGBA
	rotation	float32
	scale		float32
	texture		rl.Texture2D
}

type Gem struct {
	gemType	GemType
	x		int
	y		int
}

func drawGrid() {
	for i := 0; i < GRID_LENGTH; i++ {
		for j := 0; j < GRID_LENGTH; j++ {
			drawCell(i, j)
		}
	}
}

func drawCell(x int, y int) {
	gridColor := rl.LightGray
	if (x % 2 > 0 && y % 2 == 0) || (y % 2 > 0 && x % 2 == 0) {
		gridColor = rl.DarkGray
	}
	gridColor.A = uint8(122)
	// rect := rl.Rectangle{ X: float32(x * int(CELL_WIDTH)), Y: float32(y * int(CELL_WIDTH)), Width: CELL_WIDTH, Height: CELL_WIDTH }
	rl.DrawRectangle(int32(x * CELL_WIDTH), int32(y * CELL_WIDTH), CELL_WIDTH, CELL_WIDTH, gridColor)
	// rl.DrawRectangleLinesEx(rect, 2, gridColor)
}

func fillGrid(gemTypes []GemType) {
	for i := 0; i < GRID_LENGTH; i++ {
		for j := 0; j < GRID_LENGTH; j++ {
			gem := gemTypes[rand.Intn(GRID_LENGTH - 1)]
			// rl.DrawTextureEx(gem.texture, rl.Vector2{ X: (float32(i) * (float32(gem.texture.Width) * gem.scale)), Y: (float32(j) * (float32(gem.texture.Width) * gem.scale)) }, gem.rotation, gem.scale, gem.color)
		}
	}
}

func generateGrid() []Cell {
	cells := [GRID_LENGTH * GRID_LENGTH]Cell
	return cells
}

func unload(gemTypes []GemType) {
	for _, g := range gemTypes {
		rl.UnloadTexture(g.texture)
	}
}

func main() {
	// bunnymark()
	screenWidth := int32(1280)
	screenHeight := int32(960)
	rl.InitWindow(screenWidth, screenHeight, "jewel game")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	amethyst := GemType{ rl.Violet, 0, 1, rl.LoadTexture("res/amethyst.png") }
	crystal := GemType{ rl.Yellow, 0, 1, rl.LoadTexture("res/crystalize.png") }
	diamond := GemType{ rl.White, 0, 1, rl.LoadTexture("res/cut-diamond.png") }
	emerald := GemType{ rl.Green, 0, 1, rl.LoadTexture("res/emerald.png") }
	rupee := GemType{ rl.Red, 0, 1, rl.LoadTexture("res/rupee.png") }
	sapphire := GemType{ rl.Blue, 0, 1, rl.LoadTexture("res/saphir.png") }
	topaz := GemType{ rl.Orange, 0, 1, rl.LoadTexture("res/topaz.png") }
	gemTypes := []GemType { amethyst, crystal, diamond, emerald, rupee, sapphire, topaz };
	// fillGrid(gemTypes)
	grid := generateGrid()
	defer unload(gemTypes)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		drawGrid()
		// for i, g := range gemTypes {
		// 	rl.DrawTextureEx(g.texture, rl.Vector2{ X: 0 + (float32(i) * (float32(g.texture.Width) * g.scale)), Y: 0 }, g.rotation, g.scale, g.color)
		// }
		rl.EndDrawing()
	}
}
