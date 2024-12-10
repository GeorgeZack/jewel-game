package ui

import (
	"fmt"
	"jewel-game/types"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var messages []string

func drawDebugText(cs, ce, ch *types.Cell) {
	strStart := fmt.Sprintf("Start: %+v", cs)
	strEnd := fmt.Sprintf("End: %+v", ce)
	strHover := fmt.Sprintf("Hover: %+v", ch)
	margin := int32(30)
	rl.DrawText(strHover, margin, int32(rl.GetScreenHeight()) - margin - 20, int32(20), rl.Black)
	rl.DrawText(strStart, margin, int32(rl.GetScreenHeight()) - margin - 20 * 2, int32(20), rl.Black)
	rl.DrawText(strEnd, margin, int32(rl.GetScreenHeight()) - margin - 20 * 3, int32(20), rl.Black)
	for i, a := range messages {
		newY := int32(rl.GetScreenHeight()) - margin - 20 * (4 + int32(i))
		if newY > 0 {
			rl.DrawText(a, margin, newY, int32(20), rl.Black)
		}
	}
}

func AddMessage(format string, a ...any) {
	messages = append(messages, fmt.Sprintf(format, a...))
}

func UpdateUI(gl *types.Globals) {
	drawDebugText(gl.CS, gl.CE, gl.CH)
}

