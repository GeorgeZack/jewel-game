package main

import (
	"image/color"
	"math/rand"

	"github.com/gen2brain/raylib-go/raylib"
)

const CELL_WIDTH = 64
const GRID_LENGTH = 8

type Cell struct {
	gem		Gem
	x		int
	y		int
	drawBkg	func(int, int)
	drawGem	func(int, int, Gem)
}

type GemType struct {
	color		color.RGBA
	rotation	float32
	scale		float32
	texture		rl.Texture2D
}

type Gem struct {
	gemType	GemType
}

func drawCell(xCord int, yCord int) {
	gridColor := rl.LightGray
	if (xCord % 2 > 0 && yCord % 2 == 0) || (yCord % 2 > 0 && xCord % 2 == 0) {
		gridColor = rl.DarkGray
	}
	gridColor.A = uint8(122)
	rl.DrawRectangle(int32(xCord * CELL_WIDTH), int32(yCord * CELL_WIDTH), CELL_WIDTH, CELL_WIDTH, gridColor)
}

func drawCellBorder(xCord int, yCord int) {
	rect := rl.Rectangle{ X: float32(xCord * int(CELL_WIDTH)), Y: float32(yCord * int(CELL_WIDTH)), Width: CELL_WIDTH, Height: CELL_WIDTH }
	rl.DrawRectangleLinesEx(rect, 2, rl.SkyBlue)
}

func drawGem(xCord int, yCord int, gem Gem) {
	rl.DrawTexture(gem.gemType.texture, int32(xCord * CELL_WIDTH), int32(yCord * CELL_WIDTH), gem.gemType.color)
}

func generateGrid(gemTypes []GemType) [][]Cell {
	rows := make([][]Cell, GRID_LENGTH)
	for x := range rows {
		rows[x] = make([]Cell, GRID_LENGTH)
		col := rows[x]
		for y := range col {
			gemType := gemTypes[rand.Intn(len(gemTypes) -1)]
			gem := Gem{ gemType }
			rows[x][y] = Cell{ gem, x, y, drawCell, drawGem }
		}
	}
	return rows
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
	grid := generateGrid(gemTypes)
	defer unload(gemTypes)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		for x := range grid {
			for y := range grid[x] {
				grid[x][y].drawBkg(x, y)
				grid[x][y].drawGem(x, y, grid[x][y].gem)
			}
		}
		// for i, g := range gemTypes {
		// 	rl.DrawTextureEx(g.texture, rl.Vector2{ X: 0 + (float32(i) * (float32(g.texture.Width) * g.scale)), Y: 0 }, g.rotation, g.scale, g.color)
		// }
		rl.EndDrawing()
	}
}
