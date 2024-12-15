package ui

import (
	"fmt"
	"jewel-game/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var messages []string

func drawDebugText(dm *utils.DataManager) {
	font := dm.Consts.DebugFont
	fontSize := float32(16)
	spacing := float32(1)
	margin := float32(20)
	strAnim :=		fmt.Sprintf("Anim.: %+v", dm.Animating)
	strAnimTime :=	fmt.Sprintf("A.Tim: %+v", dm.AnimationTimeout)
	strCurr :=		fmt.Sprintf("Curr:  %+v", dm.CellCur)
	strEnd :=		fmt.Sprintf("End:   %+v", dm.CellEnd)
	strHover :=		fmt.Sprintf("Hover: %+v", dm.CellHover)
	strMoved :=		fmt.Sprintf("Moved: %+v", dm.MoveStarted)
	strPrev :=		fmt.Sprintf("Prev:  %+v", dm.CellPrev)
	strStart :=		fmt.Sprintf("Start: %+v", dm.CellStart)
	inc := float32(0)
	hPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	rl.DrawTextEx(font, strHover, hPos, fontSize, spacing, rl.Black)
	inc++
	cPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	rl.DrawTextEx(font, strCurr, cPos, fontSize, spacing, rl.Black)
	inc++
	pPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	rl.DrawTextEx(font, strPrev, pPos, fontSize, spacing, rl.Black)
	inc++
	sPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	rl.DrawTextEx(font, strStart, sPos, fontSize, spacing, rl.Black)
	inc++
	ePos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	rl.DrawTextEx(font, strEnd, ePos, fontSize, spacing, rl.Black)
	inc++
	mPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	rl.DrawTextEx(font, strMoved, mPos, fontSize, spacing, rl.Black)
	inc++
	aPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	rl.DrawTextEx(font, strAnim, aPos, fontSize, spacing, rl.Black)
	inc++
	tPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	rl.DrawTextEx(font, strAnimTime, tPos, fontSize, spacing, rl.Black)
	inc++
	for i, a := range messages {
		pos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * (inc + float32(i)) }
		if pos.Y > 0 {
			rl.DrawTextEx(font, a, pos, fontSize, spacing, rl.Black)
		}
	}
}

func AddMessage(format string, a ...any) {
	messages = append(messages, fmt.Sprintf(format, a...))
}

func UnloadGemTextures(gemTypes []utils.GemType) {
	for _, g := range gemTypes {
		rl.UnloadTexture(g.Texture)
	}
}

func UpdateUI(dm *utils.DataManager) {
	drawDebugText(dm)
}

