package ui

import (
	"fmt"
	"jewel-game/types"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var messages []string

func drawDebugText(states *types.States) {
	font := states.MonoSpaceFont
	strHover := fmt.Sprintf("Hover: %+v", states.CellHover)
	strCurr := fmt.Sprintf("Curr:  %+v", states.CurrCell)
	strPrev := fmt.Sprintf("Prev:  %+v", states.PrevCell)
	strStart := fmt.Sprintf("Start: %+v", states.CellStart)
	strEnd := fmt.Sprintf("End:   %+v", states.CellEnd)
	strMoved := fmt.Sprintf("Moved: %+v", states.MoveStarted)
	strAnim := fmt.Sprintf("Anim.: %+v", states.Animating)
	strAnimTime := fmt.Sprintf("A.Tim: %+v", states.AnimationTimeout)
	fontSize := float32(16)
	spacing := float32(1)
	margin := float32(20)
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

func UnloadGemTextures(gemTypes []types.GemType) {
	for _, g := range gemTypes {
		rl.UnloadTexture(g.Texture)
	}
}

func UpdateUI(states *types.States) {
	drawDebugText(states)
}

