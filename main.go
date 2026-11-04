package main

import (
	"embed"
	"jewel-game/core"
	"jewel-game/rl"
	"log"
)

// Bundled so the game runs from anywhere, including a browser; files on disk still win for hot reload.
//
//go:embed _res assets
var assets embed.FS

func main() {
	rl.SetAssets(assets)
	var gc *core.GameController
	err := rl.Run(1916, 1058, "Jewel Game",
		func() bool {
			// created on the first frame, once the graphics context exists
			if gc == nil {
				gc = core.NewGameController()
			}
			gc.Update()
			return !gc.Quit
		},
		func() { gc.Draw() },
	)
	if gc != nil {
		gc.Unload()
	}
	if err != nil {
		log.Fatal(err)
	}
}
