//go:build js && wasm

package store

import (
	"fmt"
	"syscall/js"
)

// Browser keeps the data in the page's local storage under one key.
type Browser struct {
	key string
}

// NewBrowser returns a store that keeps its data under key.
func NewBrowser(key string) *Browser { return &Browser{key: key} }

// Load reads the stored document, returning the defaults if there is none.
func (b *Browser) Load() (d Data, err error) {
	// Set before the call so an exception still hands back defaults to play
	// with, not a zero value with every option switched off.
	d = Defaults()
	defer catch(&err, "read local storage")
	v := js.Global().Get("localStorage").Call("getItem", b.key)
	if v.IsNull() || v.IsUndefined() {
		return Defaults(), nil
	}
	return Decode([]byte(v.String()))
}

// Save writes the document.
func (b *Browser) Save(d Data) (err error) {
	defer catch(&err, "write local storage")
	enc, err := Encode(d)
	if err != nil {
		return err
	}
	js.Global().Get("localStorage").Call("setItem", b.key, string(enc))
	return nil
}

// catch turns a JavaScript exception into an error. syscall/js reports an
// exception thrown by the browser, such as storage being disabled in a
// private window or the quota being full, as a panic, and there is no
// error-returning form of Call to use instead.
func catch(err *error, doing string) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("%s: %v", doing, r)
	}
}
