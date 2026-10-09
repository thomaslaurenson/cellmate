package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestHandlerServesWhatTheBrowserNeeds(t *testing.T) {
	t.Parallel()
	root := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<!DOCTYPE html>")},
		"cellmate.wasm": &fstest.MapFile{Data: []byte("\x00asm")},
		"icon.png":      &fstest.MapFile{Data: []byte("\x89PNG\r\n\x1a\n")},
	}
	h := handler(root)

	tests := []struct {
		name     string
		path     string
		wantCode int
		wantType string
	}{
		{
			name: "the game is served as WebAssembly so it compiles as it arrives",
			path: "/cellmate.wasm", wantCode: http.StatusOK, wantType: "application/wasm",
		},
		{name: "the root is the page", path: "/", wantCode: http.StatusOK, wantType: "text/html"},
		{
			name: "the page by name is sent back to the root, as a file server does",
			path: "/index.html", wantCode: http.StatusMovedPermanently,
		},
		{name: "an image beside the page", path: "/icon.png", wantCode: http.StatusOK, wantType: "image/png"},
		{name: "anything else", path: "/nothing.txt", wantCode: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != tc.wantCode {
				t.Fatalf("GET %s = %d, want %d", tc.path, rec.Code, tc.wantCode)
			}
			if tc.wantType == "" {
				return
			}
			if got := rec.Header().Get("Content-Type"); !contains(got, tc.wantType) {
				t.Errorf("GET %s came back as %q, want %q", tc.path, got, tc.wantType)
			}
		})
	}
}

func TestHandlerTellsTheBrowserNotToCache(t *testing.T) {
	t.Parallel()
	// The only reason to run this is to see a change, so a cached copy of
	// the last build is the one thing it must never serve.
	h := handler(fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("hello")}})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control is %q, want %q", got, "no-store")
	}
}

func TestRunReportsAMissingDirectory(t *testing.T) {
	t.Parallel()
	var log bytes.Buffer
	err := run([]string{"-dir", filepath.Join(t.TempDir(), "nothing")}, &log)
	if err == nil {
		t.Fatal("run reported success with nothing to serve")
	}
	if !errors.Is(err, errNotBuilt) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want errNotBuilt wrapping fs.ErrNotExist", err)
	}
}

func TestRunReportsAnUnknownFlag(t *testing.T) {
	t.Parallel()
	var log bytes.Buffer
	if err := run([]string{"-nonsense"}, &log); err == nil {
		t.Error("run accepted a flag it does not have")
	}
}

func TestStartAndServeAnswerOverTheNetwork(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const page = "<!DOCTYPE html>cellmate"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	var log bytes.Buffer
	// Port zero means the machine picks a free one, so the test never
	// fights whatever else is running.
	ln, root, err := start([]string{"-dir", dir, "-addr", "localhost:0"}, &log)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- serve(ln, root) }()

	res, err := http.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != page {
		t.Errorf("served %q, want %q", body, page)
	}
	if !contains(log.String(), ln.Addr().String()) {
		t.Errorf("start did not say where it was listening:\n%s", log.String())
	}

	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-stopped; err != nil && !errors.Is(err, net.ErrClosed) {
		t.Errorf("serve stopped with %v", err)
	}
}

func TestStartReportsAPortItCannotHave(t *testing.T) {
	t.Parallel()
	taken, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()

	var log bytes.Buffer
	_, _, err = start([]string{"-dir", t.TempDir(), "-addr", taken.Addr().String()}, &log)
	if err == nil {
		t.Error("start took a port that was already in use")
	}
}

func contains(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}
