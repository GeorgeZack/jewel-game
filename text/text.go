// Package text is the game's language catalog. English in the code is the fallback.
// A language file replaces any string it names, and a missing key keeps the fallback,
// so a translation can be partial and the game still runs.
package text

import (
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	mu       sync.RWMutex
	catalog  map[string]string
	language = "en"
)

// Load replaces the catalog with the strings in a language file.
// Nested maps become dotted keys: ui.spellbook, spell.smite.name.
func Load(lang string, data []byte) error {
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	flat := map[string]string{}
	flatten("", raw, flat)
	mu.Lock()
	catalog = flat
	if lang != "" {
		language = lang
	}
	mu.Unlock()
	return nil
}

// Reset drops the catalog. Fallbacks in the code are used until the next Load.
func Reset(lang string) {
	mu.Lock()
	catalog = nil
	if lang != "" {
		language = lang
	}
	mu.Unlock()
}

// Language is the code of the catalog last loaded, or "en".
func Language() string {
	mu.RLock()
	defer mu.RUnlock()
	return language
}

// T returns the catalog string for key, or fallback when the key is absent.
func T(key, fallback string) string {
	mu.RLock()
	s, ok := catalog[key]
	mu.RUnlock()
	if ok {
		return s
	}
	if fallback != "" {
		return fallback
	}
	return key
}

// Tf formats the catalog string for key. The fallback is a fmt pattern,
// and so is the translation, so a language can reorder arguments with %[1]s.
func Tf(key, fallback string, args ...any) string {
	return fmt.Sprintf(T(key, fallback), args...)
}

func flatten(prefix string, v any, out map[string]string) {
	switch m := v.(type) {
	case map[string]any:
		for k, child := range m {
			next := k
			if prefix != "" {
				next = prefix + "." + k
			}
			flatten(next, child, out)
		}
	case string:
		if prefix != "" {
			out[prefix] = m
		}
	}
}
