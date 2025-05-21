package game

import (
	"image/color"
	"jewel-game/data"
	"jewel-game/debug"
	"jewel-game/entities"
	"jewel-game/tiles"
	"jewel-game/ui"
	"jewel-game/utils"
	"math/rand"

	"github.com/fsnotify/fsnotify"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type GameController struct {
	AvailalbleGemTypes	[]*entities.GemType
	DataManager			*data.DataManager
	ReloadChannel		chan struct{}
	TileManager			tiles.TileManager
	TileStates			[]tiles.TileState
	bkg					rl.Texture2D
}

func NewGameController() GameController {
	reload := make(chan struct{}, 100)
	dm := data.NewDataManager(reload)
	tm := tiles.NewTileManager(utils.GRID_LENGTH, utils.GRID_LENGTH)
	states := InitTileStates()
	gemTypes := dm.GetAllGemTypes()

	for _, grid := range tm.Grid {
		for _, t := range grid {
			randomGemTypeIndex := rand.Intn(len(gemTypes))
			selectedGemType := gemTypes[randomGemTypeIndex]
			newGemEntity := entities.NewGem(selectedGemType)
			t.AddEntity(newGemEntity)
		}
	}

	bkg := rl.LoadTexture("_res/bkg.jpg")

	return GameController{
		AvailalbleGemTypes:	dm.GetAllGemTypes(),
		DataManager:		&dm,
		ReloadChannel:		reload,
		TileManager:		tm,
		TileStates:			states,
		bkg:				bkg,
	}
}

func (c *GameController) Unload() {
	rl.UnloadTexture(c.bkg)
	rl.UnloadFont(c.DataManager.Consts.DebugFont)
	for _, g := range c.AvailalbleGemTypes {
		defer rl.UnloadTexture(g.Texture)
	}
}

func (c *GameController) Update() {
	select {
	case event, ok := <-c.watcherEvents:
		if !ok {
			debug.AddMessage("Watcher events channel closed.")
			c.water = nil
			return
		}
		if event.Name == "assets/gemtypes.yaml" && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
			debug.AddMessage("Detected change in %s, reloading gemt types...", event.Name)
			// TODO: Implement these changes:https://gemini.google.com/app/62bd52abc01c2077
		}
	}
	c.TileManager.Update()
}

func (c *GameController) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)
	rl.DrawTexture(c.bkg, 0, -1200, color.RGBA{ 255, 255, 255, 205 })
	c.TileManager.Draw()
	ui.Draw(c.DataManager)
	rl.EndDrawing()
}

func InitTileStates() []tiles.TileState {
	blank := tiles.NewTileState("_blank", color.RGBA{ 55, 55, 55, 128 })
	grass := tiles.NewTileState("Grass", color.RGBA{ 0, 102, 0, 255 })
	forest := tiles.NewTileState("Forest", color.RGBA{ 0, 51, 0, 255 })
	water := tiles.NewTileState("Water", color.RGBA{ 0, 0, 185, 255 })
	return []tiles.TileState{ blank, grass, forest, water }
}

