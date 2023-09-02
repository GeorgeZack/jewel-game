package main

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/gen2brain/raylib-go/raylib"
)

const CELL_WIDTH = 64
const GRID_LENGTH = 8
var cellHovered *Cell
var cellSelected *Cell

type Cell struct {
	gem		Gem
	cords	rl.Vector2
}


type Gem struct {
	gemType	*GemType
}

type GemType struct {
	color			color.RGBA
	rotation, scale	float32
	texture			rl.Texture2D
}

type Grid [][]Cell

func (cell *Cell) drawCell() {
	xCord := int(cell.cords.X)
	yCord := int(cell.cords.Y)
	gridColor := rl.LightGray
	if (xCord % 2 > 0 && yCord % 2 == 0) || (yCord % 2 > 0 && xCord % 2 == 0) {
		gridColor = rl.DarkGray
	}
	gridColor.A = uint8(122)
	rl.DrawRectangle(int32(xCord * CELL_WIDTH), int32(yCord * CELL_WIDTH), CELL_WIDTH, CELL_WIDTH, gridColor)
}

func (cell *Cell) drawCellBorder() {
	xCord := cell.cords.X
	yCord := cell.cords.Y
	rect := rl.Rectangle{ X: float32(xCord * CELL_WIDTH), Y: float32(yCord * CELL_WIDTH), Width: CELL_WIDTH, Height: CELL_WIDTH }
	rl.DrawRectangleLinesEx(rect, 2, rl.SkyBlue)
}

func drawDebugText() {
	strHover := fmt.Sprintf("%+v", cellHovered)
	strSelect := fmt.Sprintf("%+v", cellSelected)
	margin := int32(10)
	rl.DrawText(strHover, int32(CELL_WIDTH * GRID_LENGTH) + margin, margin, int32(20), rl.Black)
	rl.DrawText(strSelect, int32(CELL_WIDTH * GRID_LENGTH) + margin, margin * 2 + 20, int32(20), rl.Black)
}

func (cell *Cell) drawGem() {
	xCord := cell.cords.X
	yCord := cell.cords.Y
	gem := cell.gem
	rl.DrawTexture(gem.gemType.texture, int32(xCord * CELL_WIDTH), int32(yCord * CELL_WIDTH), gem.gemType.color)
}

func generateGrid(gemTypes []GemType) Grid {
	rows := make([][]Cell, GRID_LENGTH)
	for x := range rows {
		rows[x] = make([]Cell, GRID_LENGTH)
		for y := range rows[x] {
			var gemType GemType = gemTypes[rand.Intn(len(gemTypes) -1)]
			// var gemType GemType
			gem := Gem{ gemType: &gemType }
			cords := rl.Vector2{ X: float32(x), Y: float32(y) }
			rows[x][y] = Cell{ gem, cords }
		}
	}
	return rows
}

func getCell(grid Grid) *Cell {
	mouse := rl.GetMousePosition()
	for x := range grid {
		for y := range grid[x] {
			cell := grid[x][y]
			xx := cell.cords.X
			yy := cell.cords.Y
			if (mouse.X >= xx * CELL_WIDTH && mouse.X < xx * CELL_WIDTH + CELL_WIDTH &&
				mouse.Y >= yy * CELL_WIDTH && mouse.Y < yy * CELL_WIDTH + CELL_WIDTH) {
				return &cell
			}
		}
	}
	return nil
}

func unloadGemTextures(gemTypes []GemType) {
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

	amethyst 	:= GemType{ rl.Violet,	0, 1, rl.LoadTexture("res/amethyst.png") }
	crystal 	:= GemType{ rl.Yellow,	0, 1, rl.LoadTexture("res/crystalize.png") }
	diamond 	:= GemType{ rl.White,	0, 1, rl.LoadTexture("res/cut-diamond.png") }
	emerald 	:= GemType{ rl.Green,	0, 1, rl.LoadTexture("res/emerald.png") }
	rupee 		:= GemType{ rl.Red,		0, 1, rl.LoadTexture("res/rupee.png") }
	sapphire 	:= GemType{ rl.Blue,	0, 1, rl.LoadTexture("res/saphir.png") }
	topaz 		:= GemType{ rl.Orange,	0, 1, rl.LoadTexture("res/topaz.png") }
	gemTypes := []GemType { amethyst, crystal, diamond, emerald, rupee, sapphire, topaz };
	grid := generateGrid(gemTypes)
	defer unloadGemTextures(gemTypes)

	for !rl.WindowShouldClose() {
		mouse := rl.GetMousePosition()
		curCell := getCell(grid)
		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			cellSelected = curCell
		}
		cellHovered = curCell

		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		for x := range grid {
			for y := range grid[x] {
				cell := grid[x][y]
				cell.drawCell()
				cell.drawGem()
			}
		}
		if (cellHovered != nil) {
			cellHovered.drawCellBorder()
		}
		if (cellSelected != nil) {
			cellSelected.drawCellBorder()
		}
		if (mouse.X > CELL_WIDTH * GRID_LENGTH || mouse.Y > CELL_WIDTH * GRID_LENGTH) {
			cellHovered = nil
		}
		drawDebugText()
		rl.EndDrawing()
	}
}
