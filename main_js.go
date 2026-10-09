//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"os"
	"syscall/js"

	"github.com/thomaslaurenson/cellmate/internal/store"
	"github.com/thomaslaurenson/cellmate/internal/ui"
)

// version is stamped by build_wasm through ldflags. The desktop build's
// version lives in cmd, which the browser build does not link, so stamping
// cmd.Version here would be silently ignored.
var version = "dev"

// main runs the game in the browser. There is no command line, so the page
// URL stands in for it; see optionsFromQuery.
func main() {
	b := store.NewBrowser("cellmate")
	saved, err := b.Load()
	var st ui.Saver = b
	if err != nil {
		// Leave data this build cannot read untouched rather than
		// overwrite it with defaults.
		fmt.Fprintf(os.Stderr, "[!] %v; playing without saving statistics\n", err)
		st = nil
	}

	opts, err := optionsFromQuery(js.Global().Get("location").Get("search").String(), saved.Settings.Edition)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
	}
	opts.Saved = saved
	opts.Store = st
	opts.Version = version
	if err := ui.Run(context.Background(), opts); err != nil {
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
	}
}
