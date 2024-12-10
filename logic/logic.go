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

func InitGlobals() (globals types.Globals) {
	return globals
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

func GameLoop(grid *types.Grid, gl *types.Globals) {
	// Check if the left mouse button is pressed
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		gl.IsMouseButtonPressed = true
		gl.CS = grid.GetCell()
	}

	// Check if the left mouse button is released
	if rl.IsMouseButtonReleased(rl.MouseLeftButton) && gl.IsMouseButtonPressed {
		gl.IsMouseButtonPressed = false
		gl.CE = grid.GetCell()

		if gl.CS != nil && gl.CE != nil {
			rangeOfCells := GetRangeOfCells(gl.CS, gl.CE)
			if gl.CS != gl.CE && rangeOfCells == 1 {
				// "Empty" the gems in the selected cells
				tmpS := gl.CS.Gem
				tmpE := gl.CE.Gem
				gl.CS.DestroyGem()
				gl.CE.DestroyGem()
				gl.CS.Gem = tmpE
				gl.CE.Gem = tmpS
			}
		}
	}
	gl.CH = grid.GetCell()
	grid.Redraw()

	if gl.CH != nil {
		gl.CH.DrawCellBorder()
	}
	if gl.CS != nil {
		gl.CS.DrawCellBorder()
	}
	if gl.CE != nil {
		gl.CE.DrawCellBorder()
	}

	ui.UpdateUI(gl)
}

