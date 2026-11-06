package core

import (
	"path/filepath"
	"strings"

	"jewel-game/board"
	"jewel-game/text"

	"gopkg.in/yaml.v3"
	"jewel-game/rl"
	"jewel-game/ui"
)

const (
	defaultStoryDir = "assets/story"
	langFile        = "assets/lang.yaml"
	langDir         = "assets/lang"
)

// storyRoot is the story folder for the active language. A translation may
// ship assets/lang/<code>/story/story.yaml; otherwise the English story is used.
var storyRoot = defaultStoryDir

func loadLocale() {
	lang := "en"
	if data, err := rl.ReadFile(langFile); err == nil {
		var cfg struct {
			Language string `yaml:"language"`
		}
		if yaml.Unmarshal(data, &cfg) == nil && strings.TrimSpace(cfg.Language) != "" {
			lang = strings.TrimSpace(cfg.Language)
		}
	}
	path := filepath.Join(langDir, lang+".yaml")
	data, err := rl.ReadFile(path)
	if err != nil && lang != "en" {
		ui.AddMessage("%s", text.Tf("log.lang_read", "Could not read %s: %v", path, err))
		lang = "en"
		path = filepath.Join(langDir, "en.yaml")
		data, err = rl.ReadFile(path)
	}
	if err != nil {
		ui.AddMessage("%s", text.Tf("log.lang_read", "Could not read %s: %v", path, err))
		text.Reset(lang)
	} else if err := text.Load(lang, data); err != nil {
		ui.AddMessage("%s", text.Tf("log.lang_parse", "Could not read language file: %v", err))
	}
	storyRoot = defaultStoryDir
	alt := filepath.Join(langDir, text.Language(), "story")
	if _, err := rl.ReadFile(filepath.Join(alt, "story.yaml")); err == nil {
		storyRoot = alt
	}
	rl.SetWindowTitle(text.T("ui.window_title", "The Hollow King"))
}

// storyPath is a file inside the active story, falling back to the English
// story when a translation did not include that file (usually a picture).
func storyPath(parts ...string) string {
	if storyRoot != defaultStoryDir {
		p := filepath.Join(append([]string{storyRoot}, parts...)...)
		if _, err := rl.ReadFile(p); err == nil {
			return p
		}
	}
	return filepath.Join(append([]string{defaultStoryDir}, parts...)...)
}

func storyDirs(root string) []string {
	return []string{
		root,
		filepath.Join(root, "scenes"),
		filepath.Join(root, "portraits"),
		filepath.Join(root, "backgrounds"),
		filepath.Join(root, "foes"),
	}
}

func isLocaleFile(name string) bool {
	base := filepath.Base(name)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") {
		return false
	}
	clean := filepath.Clean(name)
	if clean == filepath.Clean(langFile) {
		return true
	}
	rel, err := filepath.Rel(langDir, clean)
	if err != nil || strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)) {
		return false
	}
	switch strings.ToLower(filepath.Ext(clean)) {
	case ".yaml", ".yml":
		return true
	}
	return false
}

func trDir(d board.Direction) string {
	name := d.String()
	return text.T("dir."+strings.ToLower(name), name)
}

func trSpecial(s board.Special) string {
	name := s.String()
	if name == "" {
		return ""
	}
	return text.T("special."+strings.ToLower(name), name)
}
