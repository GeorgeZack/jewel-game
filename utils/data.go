package utils

import rl "github.com/gen2brain/raylib-go/raylib"

type Consts struct {
	DebugFont	rl.Font
}

type DataManager struct {
	Animating				bool
	AnimationTimeout		int
	CellCur					*Cell
	CellEnd					*Cell
	CellHover				*Cell
	CellPrev				*Cell
	CellStart				*Cell
	Consts					*Consts
	IsMouseButtonPressed	bool
	MoveStarted				bool
}

func NewDataManager() (*DataManager) {
	consts := &Consts{
		DebugFont: rl.LoadFontEx("res/NotoMono-Regular.ttf", 16, nil),
	}
	return &DataManager{
		Animating:				false,
		AnimationTimeout:		0,
		CellCur:				nil,
		CellEnd:				nil,
		CellHover:				nil,
		CellPrev:				nil,
		CellStart:				nil,
		Consts:					consts,
		IsMouseButtonPressed:	false,
		MoveStarted: 			false,
	}
}

func (dm *DataManager) BeginGemMovement() {
	dm.MoveStarted = true
	dm.Animating = true
	dm.AnimationTimeout = 30
}

