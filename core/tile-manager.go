package core

import (
	"jewel-game/ui"
	"jewel-game/entities"
	"math/rand"
)

type TileManager struct {
	Grid				[][]*Tile
	Width, Height		int
	ScreenX, ScreenY	int
	DataManager			*DataManager
}

func NewTileManager(w, h, screenX, screenY int, dm *DataManager) TileManager {
	tm := TileManager{
		Width: w,
		Height: h,
		ScreenX: screenX,
		ScreenY: screenY,
		DataManager: dm,
	}
	grid := make([][]*Tile, h)
	for y := 0; y < h; y++ {
		grid[y] = make([]*Tile, w)
		for x := 0; x < w; x++ {
			grid[y][x] = NewTile(x, y, &tm)
		}
	}
	tm.Grid = grid
	ui.AddMessage("New TileManager created: %+v", tm)
	return tm
}

func (tm *TileManager) Draw() {
	for y := 0; y < tm.Height; y++ {
		for x := 0; x < tm.Width; x++ {
			tm.Grid[y][x].Draw()
		}
	}
}

func (tm *TileManager) PopulateRandomGems() {
	gemTypes := tm.DataManager.GetAllGemTypes()
	for _, grid := range tm.Grid {
		for _, t := range grid {
			t.RemoveAllEntities()
			randomGemTypeIndex := rand.Intn(len(gemTypes))
			selectedGemType := gemTypes[randomGemTypeIndex]
			newGemEntity := entities.NewGem(selectedGemType)
			t.AddEntity(newGemEntity)
		}
	}

}

func (tm *TileManager) Update() {
	for y := 0; y < tm.Height; y++ {
		for x := 0; x < tm.Width; x++ {
			tm.Grid[y][x].Update()
		}
	}
}

