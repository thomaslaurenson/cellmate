// Command devserve serves a built browser game over HTTP so it can be
// played locally, the way GitHub Pages will serve it.
//
// It exists so that trying the browser build needs nothing installed but
// a Go toolchain, and so that the one thing a plain file server gets wrong
// for WebAssembly is got right here.
//
//	go run ./internal/devserve -dir dist/web
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// errNotBuilt is returned when there is nothing to serve yet.
var errNotBuilt = errors.New("build it first with make build_wasm")

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
		os.Exit(1)
	}
}

// run serves until it fails, writing progress and flag errors to errw.
func run(args []string, errw io.Writer) error {
	ln, root, err := start(args, errw)
	if err != nil {
		return err
	}
	return serve(ln, root)
}

// start reads the arguments and opens the port, so that everything that
// can go wrong has gone wrong before anything is served.
func start(args []string, errw io.Writer) (net.Listener, fs.FS, error) {
	flags := flag.NewFlagSet("devserve", flag.ContinueOnError)
	flags.SetOutput(errw)
	dir := flags.String("dir", filepath.Join("dist", "web"), "directory to serve")
	addr := flags.String("addr", "localhost:8080", "address to listen on")
	if err := flags.Parse(args); err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(*dir); err != nil {
		return nil, nil, fmt.Errorf("%w; %w", err, errNotBuilt)
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return nil, nil, err
	}
	fmt.Fprintf(errw, "[*] Serving %s on http://%s\n", *dir, ln.Addr())
	return ln, os.DirFS(*dir), nil
}

// serve answers requests until the listener is closed.
func serve(ln net.Listener, root fs.FS) error {
	srv := &http.Server{
		Handler:           handler(root),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// handler serves the files in root.
func handler(root fs.FS) http.Handler {
	files := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A browser only compiles WebAssembly as it arrives if it is
		// served as WebAssembly, and Go's own table does not know the
		// type. Without this the page still works, but by the slower
		// path, so the local game would behave unlike the published one.
		if strings.HasSuffix(r.URL.Path, ".wasm") {
			w.Header().Set("Content-Type", "application/wasm")
		}
		// The point of running this is to see a change, so nothing is
		// worth caching.
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
}
