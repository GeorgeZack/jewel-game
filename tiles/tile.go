package tiles

import (
	"jewel-game/debug"
	"jewel-game/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Tile struct {
	X, Y		int
	state		*TileState
	entities	[]TileEntity
	// dirty	bool
}

func NewTile(x, y int) *Tile {
	return &Tile{X: x, Y: y}
}

func (t *Tile) AddEntity(entity TileEntity) {
	t.entities = append(t.entities, entity)
}

func (t *Tile) Contains(p rl.Vector2) bool {
	x := float32(t.X * utils.CELL_WIDTH)
	y := float32(t.Y * utils.CELL_WIDTH)
	w := float32(utils.CELL_WIDTH)
	return p.X >= x && p.X <= x+w && p.Y >= y && p.Y <= y+w
}

func (t *Tile) Draw() {
	posX := int32(t.X * utils.CELL_WIDTH)
	posY := int32(t.Y * utils.CELL_WIDTH)
	rl.DrawRectangle(posX, posY, utils.CELL_WIDTH, utils.CELL_WIDTH, rl.LightGray)
	rl.DrawRectangleLines(posX, posY, utils.CELL_WIDTH, utils.CELL_WIDTH, rl.DarkGray)
	for _, a := range t.entities {
		a.Draw(posX, posY)
	}
}

func (t *Tile) RemoveAllEntities() {
	t.entities = nil
}

func (t *Tile) SetEntities(entities []TileEntity) {
	t.entities = entities
}

func (t *Tile) SetState(state *TileState) {
	t.state = state
}

func (t *Tile) Update() {
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mouse := rl.GetMousePosition()
		if t.Contains(mouse) {
			debug.AddMessage("Tile clicked: (%d, %d)", t.X, t.Y)
		}
	}
}

