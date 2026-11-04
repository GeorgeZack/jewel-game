//go:build js

package rpg

import (
	"errors"
	"syscall/js"
)

func savePath() string {
	return "jewel-game/hero.json"
}

func storage() (js.Value, error) {
	s := js.Global().Get("localStorage")
	if s.IsUndefined() || s.IsNull() {
		return js.Value{}, errors.New("localStorage is not available")
	}
	return s, nil
}

func writeSave(key string, data []byte) (err error) {
	s, err := storage()
	if err != nil {
		return err
	}
	// setItem throws when storage is full or disabled
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("could not write to localStorage")
		}
	}()
	s.Call("setItem", key, string(data))
	return nil
}

func readSave(key string) ([]byte, error) {
	s, err := storage()
	if err != nil {
		return nil, err
	}
	v := s.Call("getItem", key)
	if v.IsNull() {
		return nil, nil
	}
	return []byte(v.String()), nil
}

func removeSave(key string) error {
	s, err := storage()
	if err != nil {
		return err
	}
	s.Call("removeItem", key)
	return nil
}
