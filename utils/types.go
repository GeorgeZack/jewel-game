package utils

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const CELL_WIDTH = 64
const GRID_LENGTH = 8

type Grid struct {
	Data		[][]Cell
	GemTypes	[]GemType
	ScreenPos	rl.Vector2
}

func (grid *Grid) GetCell() (*Cell) {
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

func (grid *Grid) Redraw() {
	for x := range grid.Data {
		for y := range grid.Data[x] {
			cell := grid.Data[x][y]
			cell.DrawCell()
			cell.DrawGem()
		}
	}
}

type Cell struct {
	AnimatedGem	Gem
	Cords		rl.Vector2
	Gem			Gem
	Grid		*Grid
}

func (cell *Cell) AnimateGem(otherCell *Cell) {
}

func (cell *Cell) DestroyGem() {
	if cell.Gem.Valid || cell.Gem.GemType != nil {
		cell.Gem.Valid = false
		cell.Gem.GemType = nil
	}
}

func (cell *Cell) DrawCell() {
	xCord := int(cell.Cords.X)
	yCord := int(cell.Cords.Y)
	gridColor := rl.Black
	gridColor.A = uint8(200)
	if (xCord % 2 > 0 && yCord % 2 == 0) || (yCord % 2 > 0 && xCord % 2 == 0) {
		gridColor = color.RGBA{ 25, 25, 25, 200 }
	}
	rl.DrawRectangle(int32(xCord * CELL_WIDTH + int(cell.Grid.ScreenPos.X)), int32(yCord * CELL_WIDTH + int(cell.Grid.ScreenPos.Y)), CELL_WIDTH, CELL_WIDTH, gridColor)
}

func (cell *Cell) DrawCellBorder() {
	xCord := cell.Cords.X
	yCord := cell.Cords.Y
	rect := rl.Rectangle{ X: float32(xCord * CELL_WIDTH + cell.Grid.ScreenPos.X), Y: float32(yCord * CELL_WIDTH + cell.Grid.ScreenPos.Y), Width: CELL_WIDTH, Height: CELL_WIDTH }
	rl.DrawRectangleLinesEx(rect, 2, rl.SkyBlue)
}

func (cell *Cell) DrawGem() {
	xCord := cell.Cords.X
	yCord := cell.Cords.Y
	if cell.Gem.Valid {
		rl.DrawTexture(cell.Gem.GemType.Texture, int32(xCord * CELL_WIDTH + cell.Grid.ScreenPos.X), int32(yCord * CELL_WIDTH + cell.Grid.ScreenPos.Y), cell.Gem.GemType.Color)
	}
}

func (cell *Cell) MoveGem(otherCell *Cell) {
	if cell != otherCell {
		tempCurrent := cell.Gem
		tempOther := otherCell.Gem
		cell.Gem = tempOther
		otherCell.Gem = tempCurrent
		if !cell.AnimatedGem.Valid {
			cell.AnimatedGem = cell.Gem
			cell.DestroyGem()
		}
	}
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

