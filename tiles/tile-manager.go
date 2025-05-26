package tiles

import (
	"jewel-game/data"
	"jewel-game/debug"
	"jewel-game/entities"
	"math/rand"
)

type TileManager struct {
	Grid			[][]*Tile
	Width, Height	int
	DataManager		*data.DataManager
}

func NewTileManager(w, h int, dm *data.DataManager) TileManager {
	grid := make([][]*Tile, h)
	for y := 0; y < h; y++ {
		grid[y] = make([]*Tile, w)
		for x := 0; x < w; x++ {
			grid[y][x] = NewTile(x, y)
		}
	}
	debug.AddMessage("NewTileManager created")
	return TileManager{ Grid: grid, Width: w, Height: h, DataManager: dm }
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

func (tm *TileManager) Reset() {

}

func (tm *TileManager) Update() {
	for y := 0; y < tm.Height; y++ {
		for x := 0; x < tm.Width; x++ {
			tm.Grid[y][x].Update()
		}
	}
}

