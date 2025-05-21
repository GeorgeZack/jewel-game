package entities

import (
	"image/color"
	"jewel-game/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type GemType struct {
	Color			color.RGBA
	Name			string
	Rotation, Scale	float32
	Texture			rl.Texture2D
	TexturePath		string
}

type Gem struct {
	GemType	*GemType
	Valid	bool
}

func NewGem(gemType *GemType) *Gem {
	return &Gem{
		GemType: gemType,
		Valid: true,
	}
}

func (g *Gem) Draw(realX, realY int32) {
	posX := float32(realX) + float32(utils.CELL_WIDTH/2)
	posY := float32(realY) + float32(utils.CELL_WIDTH/2)
	rl.DrawTexture(g.GemType.Texture, int32(posX-float32(g.GemType.Texture.Width)/2), int32(posY-float32(g.GemType.Texture.Height)/2), g.GemType.Color)
}

func (g *Gem) Update() {
}

