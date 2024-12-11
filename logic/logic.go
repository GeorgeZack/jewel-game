package logic

import (
	"jewel-game/types"
	"jewel-game/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func GetRangeOfCells(start, end *types.Cell) int {
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

func InitGemTypes() []types.GemType {
	amethyst := types.GemType{
		Color: rl.Violet,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/amethyst.png"),
	}
	crystal := types.GemType{
		Color: rl.Yellow,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/crystalize.png"),
	}
	diamond := types.GemType{
		Color: rl.White,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/cut-diamond.png"),
	}
	emerald := types.GemType{
		Color: rl.Green,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/emerald.png"),
	}
	rupee := types.GemType{
		Color: rl.Red,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/rupee.png"),
	}
	sapphire := types.GemType{
		Color: rl.Blue,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/saphir.png"),
	}
	topaz := types.GemType{
		Color: rl.Orange,
		Rotation: 0,
		Scale: 1,
		Texture: rl.LoadTexture("res/topaz.png"),
	}
	return []types.GemType { amethyst, crystal, diamond, emerald, rupee, sapphire, topaz }
}

func InitStates() (states types.States) {
	return states
}

func InitGrid(gemTypes []types.GemType) (grid types.Grid) {
	rows := make([][]types.Cell, types.GRID_LENGTH)
	xx := float32((rl.GetScreenWidth() / 2) - ((types.CELL_WIDTH * types.GRID_LENGTH) / 2))
	yy := float32((rl.GetScreenHeight() / 2) - ((types.CELL_WIDTH * types.GRID_LENGTH) / 2))
	grid.ScreenPos = rl.Vector2{ X: xx, Y: yy }
	for x := range rows {
		rows[x] = make([]types.Cell, types.GRID_LENGTH)
		for y := range rows[x] {
			gem := types.Gem{}
			cords := rl.Vector2{ X: float32(x), Y: float32(y) }
			rows[x][y] = types.Cell{
				Cords: cords,
				Gem: gem,
				Grid: &grid,
			}
		}
	}
	grid.GemTypes = gemTypes
	grid.Data = rows

	ui.AddMessage("Populating grid...")
	for x := range grid.Data {
		for y := range grid.Data[x] {
			newGem := types.Gem{
				GemType: &grid.GemTypes[rl.GetRandomValue(0, int32(len(grid.GemTypes) - 1))],
				Valid: true,
			}
			grid.Data[x][y].Gem = newGem
		}
	}

	return grid
}

func GameLoop(grid *types.Grid, states *types.States) {
	if states.AnimationStarted && states.AnimationTimeout > 0 {
		states.AnimationTimeout--
	} else if states.AnimationTimeout == 0 {
		states.AnimationStarted = false
	}
	// Only accept interactions when no move is occurring
	if !states.MoveStarted {
		// Check if the left mouse button is pressed
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			states.IsMouseButtonPressed = true
			states.CellStart = grid.GetCell()
			if states.CurrCell != nil || states.CellStart != states.PrevCell {
				states.PrevCell = states.CurrCell
			}
		}

		// Check if the left mouse button is released
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) && states.IsMouseButtonPressed {
			states.IsMouseButtonPressed = false
			states.CellEnd = grid.GetCell()

			if states.CellStart != nil && states.CellEnd != nil {
				states.CurrCell = grid.GetCell()

				if states.CellStart != states.CellEnd {
					dragRange := GetRangeOfCells(states.CellStart, states.CellEnd)
					if dragRange == 1 {
						states.CellStart.MoveGem(states.CellEnd)
						// TODO move to func
						states.MoveStarted = true
						states.AnimationStarted = true
						states.AnimationTimeout = 100
					}
				} else if states.CurrCell != nil && states.PrevCell != nil {
					clickRange := GetRangeOfCells(states.CurrCell, states.PrevCell)
					if clickRange == 1 {
						states.CurrCell.MoveGem(states.PrevCell)
						// TODO move to func
						states.MoveStarted = true
						states.AnimationStarted = true
						states.AnimationTimeout = 100
					}
				}
			}
		}
	}
	states.CellHover = grid.GetCell()
	grid.Redraw()

	if states.CellHover != nil {
		states.CellHover.DrawCellBorder()
	}
	if states.CellStart != nil {
		states.CellStart.DrawCellBorder()
	}
	if states.CellEnd != nil {
		states.CellEnd.DrawCellBorder()
	}

	ui.UpdateUI(states)
	if states.MoveStarted && !states.AnimationStarted {
		states.MoveStarted = false
		states.CurrCell = nil
		states.CellStart = nil
		states.CellEnd = nil
	}
}

