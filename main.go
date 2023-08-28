package main

import (
	"image/color"

	"github.com/gen2brain/raylib-go/raylib"
)

type GemType struct {
	color		color.RGBA
	rotation	float32
	scale		float32
	texture		rl.Texture2D
}

func unload(gemTypes []GemType) {
	for _, g := range gemTypes {
		rl.UnloadTexture(g.texture)
	}
}

func main() {
	screenWidth := int32(1280)
	screenHeight := int32(960)
	rl.InitWindow(screenWidth, screenHeight, "raylib [core] example - basic window")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	amethyst := GemType{ rl.Violet, float32(0), float32(1), rl.LoadTexture("res/amethyst.png") }
	crystal := GemType{ rl.Yellow, float32(0), float32(1), rl.LoadTexture("res/crystalize.png") }
	diamond := GemType{ rl.White, float32(0), float32(1), rl.LoadTexture("res/cut-diamond.png") }
	emerald := GemType{ rl.Green, float32(0), float32(1), rl.LoadTexture("res/emerald.png") }
	rupee := GemType{ rl.Red, float32(0), float32(1), rl.LoadTexture("res/rupee.png") }
	sapphire := GemType{ rl.Blue, float32(0), float32(1), rl.LoadTexture("res/saphir.png") }
	topaz := GemType{ rl.Orange, float32(0), float32(1), rl.LoadTexture("res/topaz.png") }
	gemTypes := []GemType { amethyst, crystal, diamond, emerald, rupee, sapphire, topaz };
	defer unload(gemTypes)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		rl.DrawText("Congrats! You created your first window!", 190, 200, 20, rl.LightGray)
		for i, g := range gemTypes {
			rl.DrawTextureEx(g.texture, rl.Vector2{ X: 0 + (float32(i) * (float32(g.texture.Width) * g.scale)), Y: 0 }, g.rotation, g.scale, g.color)
		}
		rl.EndDrawing()
	}
}
