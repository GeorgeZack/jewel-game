//go:build !js

package core

import (
	"os"
	"path/filepath"
)

func liveMapBytes() ([]byte, bool) { return nil, false }

func writeHexMap(path string, m *worldMap) error {
	body, err := encodeHexMap(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}
