//go:build js

package core

import "syscall/js"

func init() { mapSaveDeferred = true }

func liveMapBytes() ([]byte, bool) {
	v := js.Global().Get("jewelMapYAML")
	if v.IsUndefined() || v.IsNull() {
		return nil, false
	}
	s := v.String()
	if s == "" {
		return nil, false
	}
	return []byte(s), true
}

// writeHexMap asks the dev server to store the map. The browser has no project folder.
func writeHexMap(path string, m *worldMap) error {
	body, err := encodeHexMap(m)
	if err != nil {
		return err
	}
	js.Global().Set("jewelMapYAML", string(body))
	opts := js.Global().Get("Object").New()
	opts.Set("method", "POST")
	opts.Set("body", string(body))
	headers := js.Global().Get("Object").New()
	headers.Set("Content-Type", "application/yaml")
	opts.Set("headers", headers)
	fail := js.FuncOf(func(js.Value, []js.Value) any {
		mapSaveNote = "fail"
		return nil
	})
	ok := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if args[0].Get("ok").Bool() {
			mapSaveNote = "saved"
		} else {
			mapSaveNote = "fail"
		}
		return nil
	})
	js.Global().Call("fetch", "/__map", opts).Call("then", ok).Call("catch", fail)
	return nil
}
