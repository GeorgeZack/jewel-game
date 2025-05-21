package tiles

import (
	"jewel-game/debug"
)

type TileManager struct {
	Grid			[][]*Tile
	Width, Height	int
}

func NewTileManager(w, h int) TileManager {
	grid := make([][]*Tile, h)
	for y := 0; y < h; y++ {
		grid[y] = make([]*Tile, w)
		for x := 0; x < w; x++ {
			grid[y][x] = NewTile(x, y)
		}
	}
	debug.AddMessage("NewTileManager created")
	return TileManager{ Grid: grid, Width: w, Height: h}
}

func (tm *TileManager) Draw() {
	for y := 0; y < tm.Height; y++ {
		for x := 0; x < tm.Width; x++ {
			tm.Grid[y][x].Draw()
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

