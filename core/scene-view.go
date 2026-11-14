package core

import (
	"jewel-game/rl"
	"jewel-game/rpg"
	"jewel-game/text"
	"jewel-game/ui"
	"math"
	"path"
	"strings"
)

const (
	TEXT_SPEED	= 50	// characters per second as dialogue types out
	TITLE_TIME	= 3.5	// seconds a title card stays up unless clicked
)

type sceneFrame struct {
	steps	[]rpg.Step
	pc		int
}

// SceneRunner plays scenes, the scripted parts of the story: dialogue, choices, title cards,
// fades, and the actions that change the hero. See docs/story.md for the step format.
type SceneRunner struct {
	v		*QuestView
	queue	[]string
	stack	[]sceneFrame

	line		*rpg.Step		// dialogue on screen
	choices		[]rpg.Choice	// the line's choices that are available
	shown		float32			// characters typed out so far
	title		*rpg.Step
	titleT		float32
	wait		float32
	fade		float32			// black over everything, 0..1
	fadeFrom	float32
	fadeTo		float32
	fadeT		float32
	fadeLen		float32
	bg			string
	skipping	bool
	duel		*rpg.DuelStep	// waiting for this duel to end
	puzzle		*rpg.PuzzleStep	// waiting for this puzzle board to be left
	lineScroll	ui.Scroll
}

// Play queues a scene; scenes play one after another.
func (s *SceneRunner) Play(id string) {
	if id == "" {
		return
	}
	if rpg.Scenes[id] == nil {
		ui.AddMessage("%s", text.Tf("log.no_scene", "Story: no scene %q", id))
		return
	}
	s.queue = append(s.queue, id)
}

// Active is true while scenes are playing or queued.
func (s *SceneRunner) Active() bool { return len(s.stack) > 0 || len(s.queue) > 0 }

// Showing is true when the scene owns the screen (not while it waits for a duel).
func (s *SceneRunner) Showing() bool { return s.Active() && s.duel == nil && s.puzzle == nil }

func (s *SceneRunner) WaitingForDuel() bool { return s.duel != nil }

// Stop drops everything, for when the story is reloaded or the hero deleted.
func (s *SceneRunner) Stop() {
	*s = SceneRunner{ v: s.v }
}

func (s *SceneRunner) Update(dt float32) {
	if s.duel != nil || s.puzzle != nil {
		return
	}
	if len(s.stack) == 0 {
		if len(s.queue) == 0 {
			return
		}
		id := s.queue[0]
		s.queue = s.queue[1:]
		s.push(id)
		s.bg, s.fade = "", 0
	}
	// run steps until one needs time or input
	for range 1000 {
		if s.blocked(dt) {
			return
		}
		dt = 0
		if !s.next() {
			return
		}
	}
	ui.AddMessage("%s", text.T("log.scene_loop", "Story: a scene ran 1000 steps without stopping, check it for a loop"))
	s.Stop()
}

// blocked advances whatever is on screen and reports whether it still needs time or input.
func (s *SceneRunner) blocked(dt float32) bool {
	if s.fadeLen > 0 {
		s.fadeT += dt
		if s.skipping {
			s.fadeT = s.fadeLen
		}
		t := min(1, s.fadeT/s.fadeLen)
		s.fade = s.fadeFrom + (s.fadeTo-s.fadeFrom)*t
		if t < 1 {
			return true
		}
		s.fadeLen = 0
	}
	if s.wait > 0 {
		s.wait -= dt
		if !s.skipping && s.wait > 0 {
			return true
		}
		s.wait = 0
	}
	if s.title != nil {
		s.titleT += dt
		if !s.skipping && s.titleT < TITLE_TIME {
			return true
		}
		s.title = nil
	}
	if s.line != nil {
		s.shown += dt * TEXT_SPEED
		if len(s.choices) > 0 {
			s.skipping = false
			return true
		}
		if !s.skipping {
			return true
		}
		s.line = nil
	}
	return s.duel != nil || s.puzzle != nil
}

func (s *SceneRunner) push(id string) {
	s.stack = append(s.stack, sceneFrame{ steps: rpg.Scenes[id] })
	if h := s.v.Hero; h != nil {
		h.Seen[id] = true
	}
}

// next runs one step; false when the scenes are over.
func (s *SceneRunner) next() bool {
	for len(s.stack) > 0 {
		top := &s.stack[len(s.stack)-1]
		if top.pc < len(top.steps) {
			step := top.steps[top.pc]
			top.pc++
			s.exec(step)
			return true
		}
		s.stack = s.stack[:len(s.stack)-1]
	}
	// all done: clear the stage, then start the next queued scene if any
	s.skipping, s.bg, s.fade, s.fadeLen = false, "", 0, 0
	s.v.save()
	if len(s.queue) == 0 {
		return false
	}
	id := s.queue[0]
	s.queue = s.queue[1:]
	s.push(id)
	return true
}

func (s *SceneRunner) exec(st rpg.Step) {
	h := s.v.Hero
	if h == nil {
		s.Stop()
		return
	}
	if len(st.If) > 0 {
		ok := h.Meets(st.If)
		if st.Then != nil || st.Else != nil {
			if ok {
				s.stack = append(s.stack, sceneFrame{ steps: st.Then })
			} else {
				s.stack = append(s.stack, sceneFrame{ steps: st.Else })
			}
			return
		}
		if !ok {
			return	// "if" alone guards the rest of the step
		}
	}

	// changes to the hero
	travelled := h.Location
	a := h.Apply(&st)
	if h.Location != travelled {
		s.v.route = nil
	}
	if a.Levels > 0 {
		s.v.show(Modal{ Title: text.Tf("ui.level_up", "Level %d.", h.Level), Body: text.T("ui.level_up_short", "Your HP rises. Spend residue in the Orrery.") })
	}
	if a.Ending != "" || a.Banked {
		s.v.save()	// the profile has the ending, or the banked meta currency, now
	}

	// presentation
	if st.Background != "" {
		s.bg = st.Background
		if st.Background == "none" {
			s.bg = ""
		}
	}
	if st.FadeIn > 0 {
		s.startFade(1, 0, st.FadeIn)
	}
	if st.FadeOut > 0 {
		s.startFade(s.fade, 1, st.FadeOut)
	}
	s.wait = st.Wait
	if st.Title != "" {
		s.title, s.titleT = &st, 0
	}
	if a.Act != nil && s.title == nil {
		// a new act's title card; a title on the same step wins
		s.title, s.titleT = &rpg.Step{ Title: a.Act.Title, Subtitle: a.Act.Subtitle }, 0
	}
	if st.Text != "" || len(st.Choices) > 0 {
		line := st
		line.Text = rpg.Words(st.Text)
		s.line, s.shown, s.choices = &line, 0, nil
		for _, c := range st.Choices {
			if h.Meets(c.If) {
				c.Text = rpg.Words(c.Text)
				s.choices = append(s.choices, c)
			}
		}
	}

	// flow
	if st.Run != "" {
		s.push(st.Run)
	}
	if st.Goto != "" {
		s.stack = s.stack[:0]
		s.push(st.Goto)
	}
	if d := st.Duel; d != nil && rpg.FoeByID(d.Foe) != nil {
		level := d.Level
		if level == 0 {
			level = h.Level
		}
		s.duel = d
		s.line, s.title = nil, nil
		s.v.startDuel(d.Foe, level, h.Location, d.Card)
	}
	if pz := st.Puzzle; pz != nil && rpg.PuzzleByID(pz.ID) != nil {
		s.puzzle = pz
		s.line, s.title = nil, nil
		s.v.startPuzzle(pz.ID)
	}
}

func (s *SceneRunner) startFade(from, to, secs float32) {
	s.fadeFrom, s.fadeTo, s.fadeT, s.fadeLen = from, to, 0, secs
	s.fade = from
}

// DuelDone resumes the scene after its duel.
func (s *SceneRunner) DuelDone(won bool) {
	d := s.duel
	s.duel = nil
	switch {
	case won && d.Win != "":
		s.stack = s.stack[:0]
		s.push(d.Win)
	case !won && d.Lose != "":
		s.stack = s.stack[:0]
		s.push(d.Lose)
	case !won:
		s.stack = s.stack[:0]
	}
}

// PuzzleDone resumes the scene after its puzzle board: route is the route taken, "" when the
// board was left unsolved. A route or a loss with a scene of its own goes there; the rest carry on.
func (s *SceneRunner) PuzzleDone(route string) {
	pz := s.puzzle
	s.puzzle = nil
	next := pz.Lose
	if route != "" {
		next = pz.Routes[route]
	}
	if next != "" {
		s.stack = s.stack[:0]
		s.push(next)
	}
}

// advance finishes typing the line, or moves past it.
func (s *SceneRunner) advance() {
	switch {
	case s.title != nil:
		s.title = nil
	case s.line != nil && int(s.shown) < len([]rune(s.line.Text)):
		s.shown = float32(len([]rune(s.line.Text)))
	case s.line != nil && len(s.choices) == 0:
		s.line = nil
	}
}

func (s *SceneRunner) choose(c rpg.Choice) {
	s.line, s.choices = nil, nil
	if c.Goto != "" {
		s.stack = s.stack[:0]
		s.push(c.Goto)
	}
	if len(c.Do) > 0 {
		s.stack = append(s.stack, sceneFrame{ steps: c.Do })
	}
}

// Draw covers area (the whole window, in panel space) with the scene.
func (s *SceneRunner) Draw(area rl.Rectangle, t float64) {
	if bg := storyImage("backgrounds", s.bg); bg.ID != 0 {
		drawCover(bg, area, 255)
	} else if s.line != nil || s.title != nil {
		rl.DrawRectangleRec(area, rl.Fade(rl.Black, 0.55))
	}
	if s.title != nil {
		a := min(1, s.titleT/0.6)
		rl.DrawRectangleRec(area, rl.Fade(rl.Black, 0.8*a))
		cy := area.Y + area.Height*0.42
		fade := func(c rl.Color) rl.Color { return rl.Fade(c, a) }
		ui.TextCenteredFit(s.title.Title, area.X+area.Width/2, cy-40, 64, area.Width-32, fade(ui.Accent))
		if s.title.Subtitle != "" {
			ui.TextCenteredFit(s.title.Subtitle, area.X+area.Width/2, cy+36, 26, area.Width-32, fade(ui.TextColor))
		}
	}
	if s.line != nil {
		s.drawLine(area, t)
	}

	if s.fade > 0 {
		rl.DrawRectangleRec(rl.Rectangle{ X: area.X - 10, Y: area.Y - 10, Width: area.Width + 20, Height: area.Height + 20 }, rl.Fade(rl.Black, s.fade))
	}
	// Skip runs through to the next choice (or the end), fades and pauses included
	skipW := float32(130)
	if narrow() {
		skipW = 108
	}
	skip := rl.Rectangle{ X: area.X + area.Width - skipW - 12, Y: area.Y + 12, Width: skipW, Height: 40 }
	if len(s.choices) == 0 && ui.Button(skip, text.T("ui.skip", "Skip  >>"), false) {
		s.skipping = true
		s.title, s.line = nil, nil
		return
	}
	// a click anywhere else, or Space / Enter, moves on
	if ui.Clicked(area) || rl.IsKeyPressed(rl.KeySpace) || rl.IsKeyPressed(rl.KeyEnter) {
		s.advance()
	}
}

func (s *SceneRunner) drawLine(area rl.Rectangle, t float64) {
	st := s.line
	phone := narrow()
	margin := float32(40)
	boxH := float32(240)
	textSize := float32(24)
	if phone {
		margin = 10
		textSize = 20
		boxH = min(float32(300), max(float32(188), area.Height*0.40))
	}
	choiceH := float32(46)
	choiceGap := float32(8)
	choiceBlock := float32(len(s.choices)) * (choiceH + choiceGap)
	if room := area.Height - 64 - choiceBlock - 12; boxH > room && room > 140 {
		boxH = room
	}
	w := min(float32(1200), area.Width-margin*2)
	if w < 64 {
		w = max(64, area.Width-8)
	}
	y := area.Y + area.Height - boxH - 10
	if y < area.Y+52 {
		y = area.Y + 52
	}
	box := rl.Rectangle{ X: area.X + (area.Width-w)/2, Y: y, Width: w, Height: boxH }

	// choices stack up above the box, as wide as the box when the screen is narrow
	choiceW := min(float32(560), box.Width-8)
	for i := len(s.choices) - 1; i >= 0; i-- {
		n := len(s.choices) - i
		r := rl.Rectangle{ X: box.X + box.Width - choiceW, Y: box.Y - float32(n)*(choiceH+choiceGap), Width: choiceW, Height: choiceH }
		if int(s.shown) >= len([]rune(st.Text)) && ui.Button(r, s.choices[i].Text, false) {
			s.choose(s.choices[i])
			return
		}
	}

	ui.Panel(box)
	textX := box.X + 20
	name, col := s.speaker(st.Who)
	// a 200px portrait only fits when the line still has a readable column beside it
	if st.Who != "" && box.Width >= 460 {
		pr := rl.Rectangle{ X: box.X + 16, Y: box.Y + 16, Width: 200, Height: min(200, box.Height-32) }
		if img := s.portrait(st.Who, st.Mood); img.ID != 0 {
			drawContain(img, pr)
		} else {
			drawFoe("hero", col, rl.Vector2{ X: pr.X + pr.Width/2, Y: pr.Y + pr.Height/2 }, min(70, pr.Height*0.35), t, 0, true)
		}
		textX = pr.X + pr.Width + 16
	} else if st.Who != "" && box.Width >= 280 {
		pr := rl.Rectangle{ X: box.X + 12, Y: box.Y + 12, Width: 64, Height: 64 }
		if img := s.portrait(st.Who, st.Mood); img.ID != 0 {
			drawContain(img, pr)
		} else {
			drawFoe("hero", col, rl.Vector2{ X: pr.X + 32, Y: pr.Y + 32 }, 26, t, 0, true)
		}
		textX = pr.X + pr.Width + 12
	}
	padR := float32(20)
	view := rl.Rectangle{ X: textX, Y: box.Y + 12, Width: box.X + box.Width - padR - textX, Height: box.Height - 24 }
	if view.Width < 40 {
		view.Width = 40
	}
	c := ui.Column{ X: view.X, Y: view.Y, W: view.Width, Gap: 6 }
	c.Y = s.lineScroll.Begin(view, uiScale())
	if s.lineScroll.Overflows() {
		c.W -= 12
	}
	if name != "" {
		r := c.Next(30)
		ui.TextFit(name, r.X, r.Y, 26, r.Width, col)
	}
	text := []rune(st.Text)
	n := min(len(text), int(s.shown))
	textCol := ui.TextColor
	if st.Who == "" {
		textCol = rl.Color{ R: 210, G: 205, B: 230, A: 255 }
	}
	c.Paragraph(string(text[:n]), textSize, textCol)
	s.lineScroll.End(c.Y)
	if n < len(text) {
		s.lineScroll.Pos = s.lineScroll.Max
	}
	if n >= len(text) && len(s.choices) == 0 {
		a := float32(0.5 + 0.5*math.Sin(t*5))
		ui.Text(">", box.X+box.Width-36, box.Y+box.Height-40, 26, rl.Fade(ui.Accent, a))
	}
}

func (s *SceneRunner) speaker(who string) (string, rl.Color) {
	if who == "" {
		return "", ui.TextColor
	}
	if who == "hero" {
		if h := s.v.Hero; h != nil {
			return rpg.ClassByName(h.Class).Label(), foeColor(rpg.ClassByName(h.Class).Color)
		}
		return "You", ui.Accent
	}
	if who == rpg.ENTITIES {
		return text.T("meta.entities", "The Others"), rl.Color{R: 150, G: 150, B: 165, A: 255}
	}
	if c := rpg.CharacterByID(who); c != nil {
		return c.Name, foeColor(c.Color)
	}
	return who, ui.TextColor
}

// portrait finds a character's picture: <portrait>_<mood> first, then <portrait>. The hero
// looks for hero_<class> before hero.
func (s *SceneRunner) portrait(who, mood string) rl.Texture2D {
	var bases []string
	if who == "hero" {
		if h := s.v.Hero; h != nil {
			bases = append(bases, "hero_"+strings.ToLower(h.Class))
		}
		bases = append(bases, "hero")
	} else if c := rpg.CharacterByID(who); c != nil && c.Portrait != "" {
		bases = append(bases, c.Portrait)
	} else {
		bases = append(bases, who)
	}
	for _, b := range bases {
		if mood != "" {
			if img := storyImage("portraits", b+"_"+mood); img.ID != 0 {
				return img
			}
		}
		if img := storyImage("portraits", b); img.ID != 0 {
			return img
		}
	}
	return rl.Texture2D{}
}

// Story art is loaded on first use and kept; a missing file is remembered so it isn't retried
// every frame. Images use nearest-neighbour scaling so pixel art stays crisp.
var storyArt = map[string]rl.Texture2D{}

func storyImage(dir, name string) rl.Texture2D {
	if name == "" {
		return rl.Texture2D{}
	}
	key := storyRoot + "/" + dir + "/" + name
	if tex, ok := storyArt[key]; ok {
		return tex
	}
	var tex rl.Texture2D
	candidates := []string{ name }
	if path.Ext(name) == "" {
		candidates = []string{ name + ".png", name + ".jpg", name + ".jpeg" }
	}
	for _, f := range candidates {
		if tex = rl.LoadTexture(storyPath(dir, f)); tex.ID != 0 {
			rl.SetTextureFilter(tex, rl.FilterPoint)
			break
		}
	}
	storyArt[key] = tex
	return tex
}

// forgetStoryArt drops loaded art so edited images are picked up.
func forgetStoryArt() {
	for _, tex := range storyArt {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
	}
	clear(storyArt)
}

// drawCover fills r with the image, keeping its shape; it may spill past r.
func drawCover(tex rl.Texture2D, r rl.Rectangle, alpha uint8) {
	scale := max(r.Width/float32(tex.Width), r.Height/float32(tex.Height))
	w, h := float32(tex.Width)*scale, float32(tex.Height)*scale
	src := rl.Rectangle{ Width: float32(tex.Width), Height: float32(tex.Height) }
	dst := rl.Rectangle{ X: r.X + (r.Width-w)/2, Y: r.Y + (r.Height-h)/2, Width: w, Height: h }
	rl.DrawTexturePro(tex, src, dst, rl.Vector2{}, 0, rl.Color{ R: 255, G: 255, B: 255, A: alpha })
}

// drawContain fits the whole image inside r, centered.
func drawContain(tex rl.Texture2D, r rl.Rectangle) {
	scale := min(r.Width/float32(tex.Width), r.Height/float32(tex.Height))
	w, h := float32(tex.Width)*scale, float32(tex.Height)*scale
	src := rl.Rectangle{ Width: float32(tex.Width), Height: float32(tex.Height) }
	dst := rl.Rectangle{ X: r.X + (r.Width-w)/2, Y: r.Y + (r.Height-h)/2, Width: w, Height: h }
	rl.DrawTexturePro(tex, src, dst, rl.Vector2{}, 0, rl.White)
}

// storyFiles reads the story from disk, falling back to the copy built into the game.
type storyFiles struct{ embedded bool }

func (f storyFiles) ReadFile(name string) ([]byte, error) {
	if f.embedded {
		return rl.ReadEmbedded(name)
	}
	return rl.ReadFile(name)
}

func (f storyFiles) ReadDir(name string) ([]string, error) {
	if f.embedded {
		return rl.ReadDirEmbedded(name)
	}
	return rl.ReadDir(name)
}
