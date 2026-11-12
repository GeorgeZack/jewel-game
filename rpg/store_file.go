//go:build !js

package rpg

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

func savePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "jewel-game", "hero.json")
}

func profilePath() string {
	return filepath.Join(filepath.Dir(savePath()), "profile.json")
}

func writeSave(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// written whole then renamed, so a crash can't leave half a save
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// readSave returns nil data and no error when there is no save.
func readSave(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

func removeSave(path string) error {
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
