package main

import (
	"fmt"
	"image/color"

	"github.com/gen2brain/raylib-go/raylib"
)

const CELL_WIDTH = 64
const GRID_LENGTH = 8
var GEM_TYPES []GemType
var cellHovered *Cell
var cellStart *Cell
var cellEnd *Cell
var isMouseButtonPressed bool
var strRecent []string

type Grid struct {
	Data		[][]Cell
	ScreenPos	rl.Vector2
}

type Cell struct {
	Cords	rl.Vector2
	Gem		Gem
	Grid	*Grid
}

type Gem struct {
	GemType	*GemType
	Valid	bool
}

type GemType struct {
	Color			color.RGBA
	Rotation, Scale	float32
	Texture			rl.Texture2D
}

func (cell *Cell) destroyGem() {
	if cell.Gem.Valid || cell.Gem.GemType != nil {
		cell.Gem.Valid = false
		cell.Gem.GemType = nil
	}
}

func (cell *Cell) drawCell() {
	xCord := int(cell.Cords.X)
	yCord := int(cell.Cords.Y)
	gridColor := rl.Black
	gridColor.A = uint8(200)
	if (xCord % 2 > 0 && yCord % 2 == 0) || (yCord % 2 > 0 && xCord % 2 == 0) {
		gridColor = color.RGBA{ 25, 25, 25, 200 }
	}
	rl.DrawRectangle(int32(xCord * CELL_WIDTH + int(cell.Grid.ScreenPos.X)), int32(yCord * CELL_WIDTH + int(cell.Grid.ScreenPos.Y)), CELL_WIDTH, CELL_WIDTH, gridColor)
}

func (cell *Cell) drawCellBorder() {
	xCord := cell.Cords.X
	yCord := cell.Cords.Y
	rect := rl.Rectangle{ X: float32(xCord * CELL_WIDTH + cell.Grid.ScreenPos.X), Y: float32(yCord * CELL_WIDTH + cell.Grid.ScreenPos.Y), Width: CELL_WIDTH, Height: CELL_WIDTH }
	rl.DrawRectangleLinesEx(rect, 2, rl.SkyBlue)
}

func (cell *Cell) drawGem() {
	xCord := cell.Cords.X
	yCord := cell.Cords.Y
	if cell.Gem.Valid {
		rl.DrawTexture(cell.Gem.GemType.Texture, int32(xCord * CELL_WIDTH + cell.Grid.ScreenPos.X), int32(yCord * CELL_WIDTH + cell.Grid.ScreenPos.Y), cell.Gem.GemType.Color)
	}
}

func drawDebugText() {
	strStart := fmt.Sprintf("Start: %+v", cellStart)
	strEnd := fmt.Sprintf("End: %+v", cellEnd)
	strHover := fmt.Sprintf("Hover: %+v", cellHovered)
	margin := int32(30)
	rl.DrawText(strHover, margin, int32(rl.GetScreenHeight()) - margin - 20, int32(20), rl.Black)
	rl.DrawText(strStart, margin, int32(rl.GetScreenHeight()) - margin - 20 * 2, int32(20), rl.Black)
	rl.DrawText(strEnd, margin, int32(rl.GetScreenHeight()) - margin - 20 * 3, int32(20), rl.Black)
	for i, a := range strRecent {
		newY := int32(rl.GetScreenHeight()) - margin - 20 * (4 + int32(i))
		if newY > 0 {
			rl.DrawText(a, margin, newY, int32(20), rl.Black)
		}
	}
}

func getRangeOfCells(start, end *Cell) int {
	x := start.Cords.X - end.Cords.X
	y := start.Cords.Y - end.Cords.Y
	xx := x
	yy := y
	if xx < 0 {
		xx = xx * -1
	}
	if yy < 0 {
		yy = yy * -1
	}
	if xx > 1 {
		return int(xx)
	}
	if yy > 1 {
		return int(yy)
	}
	return 1
}

func generateGrid() Grid {
	rows := make([][]Cell, GRID_LENGTH)
	var grid Grid
	xx := float32((rl.GetScreenWidth() / 2) - ((CELL_WIDTH * GRID_LENGTH) / 2))
	yy := float32((rl.GetScreenHeight() / 2) - ((CELL_WIDTH * GRID_LENGTH) / 2))
	grid.ScreenPos = rl.Vector2{ X: xx, Y: yy }
	for x := range rows {
		rows[x] = make([]Cell, GRID_LENGTH)
		for y := range rows[x] {
			gem := Gem{}
			cords := rl.Vector2{ X: float32(x), Y: float32(y) }
			rows[x][y] = Cell{ cords, gem, &grid }
		}
	}
	grid.Data = rows
	return grid
}

func (grid *Grid) getCell() *Cell {
	mouse := rl.GetMousePosition()
	for x := range grid.Data {
		for y := range grid.Data[x] {
			cell := &grid.Data[x][y]
			xx := cell.Cords.X
			yy := cell.Cords.Y
			if (mouse.X >= xx * CELL_WIDTH + cell.Grid.ScreenPos.X && mouse.X < xx * CELL_WIDTH + cell.Grid.ScreenPos.X + CELL_WIDTH) &&
				(mouse.Y >= yy * CELL_WIDTH + cell.Grid.ScreenPos.Y && mouse.Y < yy * CELL_WIDTH + cell.Grid.ScreenPos.Y + CELL_WIDTH) {
				return cell
			}
		}
	}
	return nil
}

func (grid *Grid) populateGrid() {
	strRecent = append([]string{"populateGrid"}, strRecent...)
	for x := range grid.Data {
		for y := range grid.Data[x] {
			newGem := Gem{
				GemType: &GEM_TYPES[rl.GetRandomValue(0, int32(len(GEM_TYPES) - 1))],
				Valid: true,
			}
			grid.Data[x][y].Gem = newGem
		}
	}
}

func unloadGemTextures() {
	for _, g := range GEM_TYPES {
		rl.UnloadTexture(g.Texture)
	}
}

func main() {
	// bunnymark()
	screenWidth := 1916
	screenHeight := 1058
	rl.InitWindow(int32(screenWidth), int32(screenHeight), "jewel game")
	rl.SetWindowSize(screenWidth, screenHeight)
	rl.SetWindowPosition(0, 0)
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)
	bkg := rl.LoadTexture("res/bkg.jpg")

	amethyst 	:= GemType{ rl.Violet,	0, 1, rl.LoadTexture("res/amethyst.png") }
	crystal 	:= GemType{ rl.Yellow,	0, 1, rl.LoadTexture("res/crystalize.png") }
	diamond 	:= GemType{ rl.White,	0, 1, rl.LoadTexture("res/cut-diamond.png") }
	emerald 	:= GemType{ rl.Green,	0, 1, rl.LoadTexture("res/emerald.png") }
	rupee 		:= GemType{ rl.Red,		0, 1, rl.LoadTexture("res/rupee.png") }
	sapphire 	:= GemType{ rl.Blue,	0, 1, rl.LoadTexture("res/saphir.png") }
	topaz 		:= GemType{ rl.Orange,	0, 1, rl.LoadTexture("res/topaz.png") }
	GEM_TYPES = []GemType { amethyst, crystal, diamond, emerald, rupee, sapphire, topaz };
	grid := generateGrid()
	grid.populateGrid()
	defer unloadGemTextures()

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		rl.DrawTexture(bkg, 0, -1200, color.RGBA{ 255, 255, 255, 205 })

		// Check if the left mouse button is pressed
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			isMouseButtonPressed = true
			cellStart = grid.getCell()
		}

		// Check if the left mouse button is released
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) && isMouseButtonPressed {
			isMouseButtonPressed = false
			cellEnd = grid.getCell()

			if cellStart != nil && cellEnd != nil {
				rangeOfCells := getRangeOfCells(cellStart, cellEnd)
				if cellStart != cellEnd && rangeOfCells == 1 {
					// "Empty" the gems in the selected cells
					tmpS := cellStart.Gem
					tmpE := cellEnd.Gem
					cellStart.destroyGem()
					cellEnd.destroyGem()
					cellStart.Gem = tmpE
					cellEnd.Gem = tmpS
				}
			}
		}
		cellHovered = grid.getCell()

		for x := range grid.Data {
			for y := range grid.Data[x] {
				cell := grid.Data[x][y]
				cell.drawCell()
				cell.drawGem()
			}
		}

		if cellHovered != nil {
			cellHovered.drawCellBorder()
		}
		if cellStart != nil {
			cellStart.drawCellBorder()
		}
		if cellEnd != nil {
			cellEnd.drawCellBorder()
		}

		drawDebugText()
		rl.EndDrawing()
	}
}

