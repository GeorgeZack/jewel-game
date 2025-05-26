package core

import "image/color"

type TileState struct {
	Name	string
	Color	color.RGBA
}

func NewTileState(name string, color color.RGBA) TileState {
	return TileState{ Name: name, Color: color }
}

