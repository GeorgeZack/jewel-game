package core

import (
	"bytes"
	"image"
	"image/draw"
	"math"
	"path/filepath"

	"jewel-game/entities"
	"jewel-game/rpg"
	"jewel-game/ui"
	"jewel-game/utils"

	"gopkg.in/yaml.v3"
	"jewel-game/rl"
)

type Consts struct {
	DebugFont rl.Font
	UIFont    rl.Font
}

type DataManager struct {
	// AnimationQueue			[]*types.Cell
	// CellCur					*types.Cell
	// CellEnd					*types.Cell
	// CellHover				*types.Cell
	// CellPrev				*types.Cell
	// CellStart				*types.Cell
	Animating            bool
	AnimationTimeout     int
	Consts               *Consts
	GemTypes             map[string]*entities.GemType
	GemTypeOrder         []string // YAML order, so a gem's color index is stable
	skull, skullPlus     rl.Texture2D
	coin, star           rl.Texture2D
	IsMouseButtonPressed bool
	MoveStarted          bool
}

// NewDataManager must be called on the main thread after the window exists, since it loads textures.
// Hot reloading is driven by the GameController's watcher so textures are only touched on the main thread.
func NewDataManager() *DataManager {
	consts := &Consts{
		DebugFont: rl.LoadFontEx("_res/NotoMono-Regular.ttf", 16, nil),
		UIFont:    rl.LoadFontEx("_res/NotoMono-Regular.ttf", 64, nil),
	}
	rl.SetTextureFilter(consts.UIFont.Texture, rl.FilterBilinear)
	dm := &DataManager{
		// AnimationQueue:			make([]*types.Cell, 0),
		// CellCur:				nil,
		// CellEnd:				nil,
		// CellHover:				nil,
		// CellPrev:				nil,
		// CellStart:				nil,
		Animating:            false,
		AnimationTimeout:     0,
		Consts:               consts,
		GemTypes:             make(map[string]*entities.GemType),
		IsMouseButtonPressed: false,
		MoveStarted:          false,
	}
	dm.LoadGemTypes(utils.GEM_TYPES_FILE)
	dm.skull = loadFullColorGem("_res/rd-skull.png")
	dm.skullPlus = loadFullColorGem("_res/rd-skull-plus.png")
	dm.coin = loadFullColorGem("_res/rd-coin.png")
	dm.star = loadFullColorGem("_res/rd-star.png")

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

// DuelGemTypes is the fixed palette of a duel board, indexed by the rpg gem colors. The mana
// gems borrow textures from gemtypes.yaml. A bomb skull uses the cracked texture.
func (dm *DataManager) DuelGemTypes() []*entities.GemType {
	from := func(name string) (rl.Texture2D, bool) {
		gt, ok := dm.GemTypes[name]
		if !ok {
			return rl.Texture2D{}, false
		}
		return gt.Texture, gt.FullColor
	}
	piece := func(name string, col rl.Color, tex rl.Texture2D, full bool, bomb rl.Texture2D) *entities.GemType {
		return &entities.GemType{Name: name, Color: col, Texture: tex, FullColor: full, Bomb: bomb}
	}
	rupee, rupeeFull := from("Rupee")
	sapphire, sapphireFull := from("Sapphire")
	crystal, crystalFull := from("Crystal")
	emerald, emeraldFull := from("Emerald")
	return []*entities.GemType{
		rpg.Fire:  piece("Fire", rl.Color{R: 240, G: 60, B: 50, A: 255}, rupee, rupeeFull, rl.Texture2D{}),
		rpg.Water: piece("Water", rl.Color{R: 70, G: 130, B: 255, A: 255}, sapphire, sapphireFull, rl.Texture2D{}),
		rpg.Air:   piece("Air", rl.Color{R: 250, G: 225, B: 70, A: 255}, crystal, crystalFull, rl.Texture2D{}),
		rpg.Earth: piece("Earth", rl.Color{R: 70, G: 210, B: 90, A: 255}, emerald, emeraldFull, rl.Texture2D{}),
		rpg.Skull: piece("Skull", rl.Color{R: 235, G: 230, B: 215, A: 255}, dm.skull, true, dm.skullPlus),
		rpg.Gold:  piece("Gold", rl.Color{R: 255, G: 190, B: 40, A: 255}, dm.coin, true, rl.Texture2D{}),
		rpg.Star:  piece("Star", rl.Color{R: 190, G: 110, B: 255, A: 255}, dm.star, true, rl.Texture2D{}),
	}
}

func (dm *DataManager) Unload() {
	rl.UnloadFont(dm.Consts.DebugFont)
	rl.UnloadFont(dm.Consts.UIFont)
	for _, gt := range dm.GemTypes {
		if gt.Texture.ID != 0 {
			rl.UnloadTexture(gt.Texture)
		}
	}
	for _, t := range []rl.Texture2D{dm.skull, dm.skullPlus, dm.coin, dm.star} {
		if t.ID != 0 {
			rl.UnloadTexture(t)
		}
	}
}

func (dm *DataManager) LoadGemTypes(fileName string) {
	data, err := rl.ReadFile(fileName)
	if err != nil {
		ui.AddMessage("Error reading gem types file %s: %v", fileName, err)
		return
	}

	var tempGemTypes []struct {
		Name        string `yaml:"name"`
		ColorHex    string `yaml:"color"`
		TexturePath string `yaml:"texturePath"`
		FullColor   bool   `yaml:"fullColor"`
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
		color := rl.Color{R: r, G: g, B: b, A: a}
		var texture rl.Texture2D
		var texturePath string
		loaded := false
		if oldGem, ok := dm.GemTypes[temp.Name]; ok && filepath.Clean(temp.TexturePath) == filepath.Clean(oldGem.TexturePath) && oldGem.Texture.ID != 0 {
			texture = oldGem.Texture
			texturePath = oldGem.TexturePath
		} else if temp.FullColor {
			texture = loadFullColorGem(temp.TexturePath)
			texturePath = temp.TexturePath
			loaded = true
		} else {
			texture = rl.LoadTexture(temp.TexturePath)
			texturePath = temp.TexturePath
			loaded = true
			if texture.ID != 0 {
				rl.SetTextureFilter(texture, rl.FilterBilinear)
			}
		}
		if texture.ID == 0 {
			ui.AddMessage("Warning: Failed to load texture for gem %s at %s", temp.Name, temp.TexturePath)
		} else if loaded {
			ui.AddMessage("Loaded new texture for gem: %s from %s", temp.Name, temp.TexturePath)
		}

		newGemTypes[temp.Name] = &entities.GemType{
			Color:       color,
			Name:        temp.Name,
			Rotation:    0,
			Scale:       1,
			Texture:     texture,
			TexturePath: texturePath,
			FullColor:   temp.FullColor,
		}
		newOrder = append(newOrder, temp.Name)
	}

	dm.GemTypes = newGemTypes
	dm.GemTypeOrder = newOrder
	ui.AddMessage("Gem types loaded: %v", newOrder)
}

// loadFullColorGem trims the transparent margin so every sprite fills its cell by about the same amount.
func loadFullColorGem(path string) rl.Texture2D {
	raw, err := rl.ReadFile(path)
	if err != nil {
		return rl.Texture2D{}
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return rl.Texture2D{}
	}
	tex := rl.NewTexture(cropGem(img))
	rl.SetTextureFilter(tex, rl.FilterPoint)
	return tex
}

// cropGem centers the opaque pixels in a square where the longest side fills 82% of the texture.
func cropGem(img image.Image) image.Image {
	b := img.Bounds()
	minx, miny := b.Max.X, b.Max.Y
	maxx, maxy := b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0x1400 {
				if x < minx {
					minx = x
				}
				if y < miny {
					miny = y
				}
				if x > maxx {
					maxx = x
				}
				if y > maxy {
					maxy = y
				}
			}
		}
	}
	if maxx < minx {
		return img
	}
	bw, bh := maxx-minx+1, maxy-miny+1
	side := int(math.Ceil(float64(max(bw, bh)) / 0.82))
	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	ox, oy := (side-bw)/2, (side-bh)/2
	draw.Draw(dst, image.Rect(ox, oy, ox+bw, oy+bh), img, image.Pt(minx, miny), draw.Src)
	return dst
}
