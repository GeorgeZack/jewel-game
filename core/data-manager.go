package core

import (
	"jewel-game/ui"
	"jewel-game/entities"
	"jewel-game/utils"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
	"gopkg.in/yaml.v3"
)

type Consts struct {
	DebugFont	rl.Font
	UIFont		rl.Font
}

type DataManager struct {
	// AnimationQueue			[]*types.Cell
	// CellCur					*types.Cell
	// CellEnd					*types.Cell
	// CellHover				*types.Cell
	// CellPrev				*types.Cell
	// CellStart				*types.Cell
	Animating				bool
	AnimationTimeout		int
	Consts					*Consts
	GemTypes				map[string]*entities.GemType
	GemTypeOrder			[]string	// YAML order, so a gem's color index is stable
	IsMouseButtonPressed	bool
	MoveStarted				bool
}

// NewDataManager must be called on the main thread after the window exists, since it loads textures.
// Hot reloading is driven by the GameController's watcher so textures are only touched on the main thread.
func NewDataManager() *DataManager {
	consts := &Consts{
		DebugFont: rl.LoadFontEx("_res/NotoMono-Regular.ttf", 16, nil),
		UIFont: rl.LoadFontEx("_res/NotoMono-Regular.ttf", 64, nil),
	}
	rl.SetTextureFilter(consts.UIFont.Texture, rl.FilterBilinear)
	dm := &DataManager{
		// AnimationQueue:			make([]*types.Cell, 0),
		// CellCur:				nil,
		// CellEnd:				nil,
		// CellHover:				nil,
		// CellPrev:				nil,
		// CellStart:				nil,
		Animating:				false,
		AnimationTimeout:		0,
		Consts:					consts,
		GemTypes: 				make(map[string]*entities.GemType),
		IsMouseButtonPressed:	false,
		MoveStarted: 			false,
	}
	dm.LoadGemTypes(utils.GEM_TYPES_FILE)

	return dm
}

func (dm *DataManager) GetAllGemTypes() []*entities.GemType {
	types := make([]*entities.GemType, 0, len(dm.GemTypeOrder))
	for _, name := range dm.GemTypeOrder {
		types = append(types, dm.GemTypes[name])
	}
	return types
}

// PrismTexture is drawn rainbow-tinted for colorless prism gems.
func (dm *DataManager) PrismTexture() rl.Texture2D {
	if gt, ok := dm.GemTypes[utils.PRISM_GEM_TYPE]; ok {
		return gt.Texture
	}
	if len(dm.GemTypeOrder) > 0 {
		return dm.GemTypes[dm.GemTypeOrder[0]].Texture
	}
	return rl.Texture2D{}
}

func (dm *DataManager) Unload() {
	rl.UnloadFont(dm.Consts.DebugFont)
	rl.UnloadFont(dm.Consts.UIFont)
	for _, gt := range dm.GemTypes {
		if gt.Texture.ID != 0 {
			rl.UnloadTexture(gt.Texture)
		}
	}
}

func (dm *DataManager) LoadGemTypes(fileName string) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		ui.AddMessage("Error reading gem types file %s: %v", fileName, err)
		return
	}

	var tempGemTypes []struct {
		Name		string	`yaml:"name"`
		ColorHex	string	`yaml:"color"`
		TexturePath	string	`yaml:"texturePath"`
		Role		string	`yaml:"role"`
	}
	if err := yaml.Unmarshal(data, &tempGemTypes); err != nil {
		ui.AddMessage("Error unmarshaling gem types: %v", err)
		return
	}

	if len(tempGemTypes) < 3 {
		ui.AddMessage("Need at least 3 gem types in %s, found %d; keeping the old ones", fileName, len(tempGemTypes))
		return
	}

	newGemTypes := make(map[string]*entities.GemType)
	newOrder := make([]string, 0, len(tempGemTypes))
	for name, oldGem := range dm.GemTypes {
		found := false
		for _, newDef := range tempGemTypes {
			if newDef.Name == name && filepath.Clean(newDef.TexturePath) == filepath.Clean(oldGem.TexturePath) && oldGem.Texture.ID != 0 {
				ui.AddMessage("Found difference in %s type", name)
				found = true
				break
			}
		}
		if !found && oldGem.Texture.ID != 0 {
			rl.UnloadTexture(oldGem.Texture)
			ui.AddMessage("Unloaded old texture for gem: %s", name)
		}
	}

	for _, temp := range tempGemTypes {
		r, g, b, a := utils.ParseHexColor(temp.ColorHex)
		color := rl.Color{ R: r, G: g, B: b, A: a }
		var texture rl.Texture2D
		var texturePath string
		if oldGem, ok := dm.GemTypes[temp.Name]; ok && filepath.Clean(temp.TexturePath) == filepath.Clean(oldGem.TexturePath) && oldGem.Texture.ID != 0 {
			texture = oldGem.Texture
			texturePath = oldGem.TexturePath
		} else {
			texture = rl.LoadTexture(temp.TexturePath)
			texturePath = temp.TexturePath
			if texture.ID == 0 {
				ui.AddMessage("Warning: Failed to load texture for gem %s at %s", temp.Name, temp.TexturePath)
			} else {
				rl.SetTextureFilter(texture, rl.FilterBilinear)
				ui.AddMessage("Loaded new texture for gem: %s from %s", temp.Name, temp.TexturePath)
			}
		}

		newGemTypes[temp.Name] = &entities.GemType{
			Color: color,
			Name: temp.Name,
			Rotation: 0,
			Scale: 1,
			Texture: texture,
			TexturePath: texturePath,
			Role: temp.Role,
		}
		newOrder = append(newOrder, temp.Name)
	}

	dm.GemTypes = newGemTypes
	dm.GemTypeOrder = newOrder
	ui.AddMessage("Gem types loaded: %v", newOrder)
}
