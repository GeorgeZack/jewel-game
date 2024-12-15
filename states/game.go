package states

import (
	"jewel-game/utils"
	"jewel-game/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func GetRangeOfCells(start, end *utils.Cell) int {
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

func InitGemTypes() []utils.GemType {
	amethyst := utils.GemType{
		Color: rl.Violet,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/amethyst.png"),
	}
	crystal := utils.GemType{
		Color: rl.Yellow,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/crystalize.png"),
	}
	diamond := utils.GemType{
		Color: rl.White,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/cut-diamond.png"),
	}
	emerald := utils.GemType{
		Color: rl.Green,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/emerald.png"),
	}
	rupee := utils.GemType{
		Color: rl.Red,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/rupee.png"),
	}
	sapphire := utils.GemType{
		Color: rl.Blue,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/saphir.png"),
	}
	topaz := utils.GemType{
		Color: rl.Orange,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/topaz.png"),
	}
	return []utils.GemType { amethyst, crystal, diamond, emerald, rupee, sapphire, topaz }
}

func InitGrid(gemTypes []utils.GemType, dm *utils.DataManager) (grid utils.Grid) {
	rows := make([][]utils.Cell, utils.GRID_LENGTH)
	xx := float32((rl.GetScreenWidth() / 2) - ((utils.CELL_WIDTH * utils.GRID_LENGTH) / 2))
	yy := float32((rl.GetScreenHeight() / 2) - ((utils.CELL_WIDTH * utils.GRID_LENGTH) / 2))
	grid.ScreenPos = rl.Vector2{ X: xx, Y: yy }
	for x := range rows {
		rows[x] = make([]utils.Cell, utils.GRID_LENGTH)
		for y := range rows[x] {
			gem := utils.Gem{}
			cords := rl.Vector2{ X: float32(x), Y: float32(y) }
			cell := utils.Cell{
				Cords: cords,
				Gem: gem,
				Grid: &grid,
			}
			rows[x][y] = cell
			// utils.SubscribeToData(dm, "Animating", func(newVal bool) {
			// })
		}
	}
	grid.GemTypes = gemTypes
	grid.Data = rows

	ui.AddMessage("Populating grid...")
	for x := range grid.Data {
		for y := range grid.Data[x] {
			newGem := utils.Gem{
				GemType: &grid.GemTypes[rl.GetRandomValue(0, int32(len(grid.GemTypes) - 1))],
				Valid: true,
			}
			grid.Data[x][y].Gem = newGem
		}
	}

	return grid
}

func GameLoop(grid *utils.Grid, dm *utils.DataManager) {
	if dm.Animating && dm.AnimationTimeout > 0 {
		dm.AnimationTimeout--
		// for i, a := range dm.AnimatingQueue {
		// }
	} else if dm.AnimationTimeout == 0 {
		dm.Animating = false
	}
	// Only accept interactions when no move is occurring
	if !dm.Animating {
		// Check if the left mouse button is pressed
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			dm.IsMouseButtonPressed = true
			if dm.CellStart == nil {
				if dm.CellPrev != nil {
					dm.CellPrev = nil
				}
				dm.CellStart = grid.GetCell()
			}
			if dm.CellCur != nil || dm.CellStart != dm.CellPrev {
				dm.CellPrev = dm.CellCur
			}
		}

		// Check if the left mouse button is released
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) && dm.IsMouseButtonPressed {
			dm.IsMouseButtonPressed = false
			dm.CellEnd = grid.GetCell()

			if dm.CellStart != nil && dm.CellEnd != nil {
				dm.CellCur = grid.GetCell()

				if dm.CellStart != dm.CellEnd {
					dragRange := GetRangeOfCells(dm.CellStart, dm.CellEnd)
					if dragRange == 1 {
						dm.CellStart.MoveGem(dm.CellEnd)
						dm.BeginGemMovement()
					}
				} else if dm.CellCur != nil && dm.CellPrev != nil {
					clickRange := GetRangeOfCells(dm.CellCur, dm.CellPrev)
					if clickRange == 1 {
						dm.CellCur.MoveGem(dm.CellPrev)
						dm.BeginGemMovement()
					}
				}
			}
		}
	}
	dm.CellHover = grid.GetCell()
	grid.Redraw()

	if dm.CellHover != nil {
		dm.CellHover.DrawCellBorder()
	}
	if dm.CellStart != nil {
		dm.CellStart.DrawCellBorder()
	}
	if dm.CellEnd != nil {
		dm.CellEnd.DrawCellBorder()
	}

	if dm.MoveStarted && !dm.Animating {
		dm.MoveStarted = false
		dm.CellCur = nil
		dm.CellStart = nil
		dm.CellEnd = nil
	}

	ui.UpdateUI(dm)
}

