package ui

import (
	"fmt"
	"log"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const MAX_MESSAGES = 200

var (
	messages	[]string
	messagesMu	sync.Mutex
	ShowDebug	bool
)

func drawDebugText(font rl.Font) {
	fontSize := float32(16)
	spacing := float32(1)
	margin := float32(20)
	// strAnim :=		fmt.Sprintf("Anim.: %+v", dm.Animating)
	// strAnimTime :=	fmt.Sprintf("A.Tim: %+v", dm.AnimationTimeout)
	// strCurr :=		fmt.Sprintf("Curr:  %+v", dm.CellCur)
	// strEnd :=		fmt.Sprintf("End:   %+v", dm.CellEnd)
	// strHover :=		fmt.Sprintf("Hover: %+v", dm.CellHover)
	// strMoved :=		fmt.Sprintf("Moved: %+v", dm.MoveStarted)
	// strPrev :=		fmt.Sprintf("Prev:  %+v", dm.CellPrev)
	// strStart :=		fmt.Sprintf("Start: %+v", dm.CellStart)
	inc := float32(0)
	// hPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	// rl.DrawTextEx(font, strHover, hPos, fontSize, spacing, rl.Black)
	// inc++
	// cPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	// rl.DrawTextEx(font, strCurr, cPos, fontSize, spacing, rl.Black)
	// inc++
	// pPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	// rl.DrawTextEx(font, strPrev, pPos, fontSize, spacing, rl.Black)
	// inc++
	// sPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	// rl.DrawTextEx(font, strStart, sPos, fontSize, spacing, rl.Black)
	// inc++
	// ePos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	// rl.DrawTextEx(font, strEnd, ePos, fontSize, spacing, rl.Black)
	// inc++
	// mPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	// rl.DrawTextEx(font, strMoved, mPos, fontSize, spacing, rl.Black)
	// inc++
	// aPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	// rl.DrawTextEx(font, strAnim, aPos, fontSize, spacing, rl.Black)
	// inc++
	// tPos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * inc }
	// rl.DrawTextEx(font, strAnimTime, tPos, fontSize, spacing, rl.Black)
	// inc++
	rl.DrawRectangle(0, 0, int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()), rl.Fade(rl.RayWhite, 0.85))
	for i, a := range GetMessages() {
		pos := rl.Vector2{ X: margin, Y: float32(rl.GetScreenHeight()) - margin - fontSize * (inc + float32(i)) }
		// Use this for word wrapping
		// https://github.com/raysan5/raylib/blob/master/examples/text/text_rectangle_bounds.c
		if pos.Y > 0 {
			rl.DrawTextEx(font, a, pos, fontSize, spacing, rl.Black)
		}
	}
}

func AddMessage(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	log.Println(msg)
	messagesMu.Lock()
	defer messagesMu.Unlock()
	messages = append([]string{ msg }, messages...)
	if len(messages) > MAX_MESSAGES {
		messages = messages[:MAX_MESSAGES]
	}
}

func Draw(font rl.Font) {
	if ShowDebug {
		drawDebugText(font)
	}
}

func GetMessages() []string {
	messagesMu.Lock()
	defer messagesMu.Unlock()
	return append([]string(nil), messages...)
}

