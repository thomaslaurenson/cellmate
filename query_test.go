package main

import (
	"errors"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/edition"
)

func TestOptionsFromQuery(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		query   string
		saved   string
		edition edition.Edition
		deal    int
		fails   bool
		errIs   error
	}{
		{name: "empty query", edition: edition.Default},
		{name: "saved edition", saved: "95", edition: edition.Win95},
		{name: "query edition wins over saved", query: "?edition=xp", saved: "95", edition: edition.WinXP},
		{name: "unknown saved edition ignored", saved: "vista", edition: edition.Default},
		{name: "game", query: "?game=617&edition=95", edition: edition.Win95, deal: 617},
		{name: "impossible game", query: "?game=-2", edition: edition.Default, deal: -2},
		{name: "unknown edition keeps saved", query: "?edition=vista", saved: "95", edition: edition.Win95, fails: true, errIs: edition.ErrUnknown},
		{name: "game not a number keeps saved", query: "?game=abc", saved: "95", edition: edition.Win95, fails: true},
		{name: "game outside edition range", query: "?game=32001", saved: "95", edition: edition.Win95, fails: true},
		{name: "malformed query keeps saved", query: "?game=%zz", saved: "95", edition: edition.Win95, fails: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			opts, err := optionsFromQuery(tc.query, tc.saved)
			switch {
			case tc.fails && err == nil:
				t.Error("want an error, got nil")
			case !tc.fails && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tc.errIs != nil && !errors.Is(err, tc.errIs):
				t.Errorf("error = %v, want %v", err, tc.errIs)
			}
			if opts.Edition != tc.edition || opts.Deal != tc.deal {
				t.Errorf("options = edition %v, deal %d; want edition %v, deal %d",
					opts.Edition, opts.Deal, tc.edition, tc.deal)
			}
		})
	}
}
