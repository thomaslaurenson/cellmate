package edition

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    Edition
		wantErr error
	}{
		{name: "windows 95", input: "95", want: Win95},
		{name: "windows xp", input: "xp", want: WinXP},
		{name: "unknown", input: "vista", wantErr: ErrUnknown},
		{name: "case matters", input: "XP", wantErr: ErrUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Parse(%q) error = %v, want %v", tc.input, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("Parse(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestCheckDeal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		edition Edition
		deal    int
		ok      bool
	}{
		{name: "95 first", edition: Win95, deal: 1, ok: true},
		{name: "95 last", edition: Win95, deal: 32000, ok: true},
		{name: "95 past end", edition: Win95, deal: 32001},
		{name: "xp past 95 range", edition: WinXP, deal: 32001, ok: true},
		{name: "xp last", edition: WinXP, deal: 1000000, ok: true},
		{name: "xp past end", edition: WinXP, deal: 1000001},
		{name: "zero", edition: WinXP, deal: 0},
		{name: "95 impossible deal", edition: Win95, deal: -1, ok: true},
		{name: "xp impossible deal", edition: WinXP, deal: -2, ok: true},
		{name: "past the impossible deals", edition: WinXP, deal: -3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.edition.CheckDeal(tc.deal)
			if tc.ok {
				if err != nil {
					t.Errorf("CheckDeal(%d) = %v, want nil", tc.deal, err)
				}
				return
			}
			var re *DealRangeError
			if !errors.As(err, &re) {
				t.Fatalf("CheckDeal(%d) = %v, want *DealRangeError", tc.deal, err)
			}
			if re.Deal != tc.deal || re.Edition != tc.edition {
				t.Errorf("DealRangeError = %+v", re)
			}
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	if got := Edition(0).String(); got != "Edition(0)" {
		t.Errorf("Edition(0).String() = %q", got)
	}
}

func TestHasUndo(t *testing.T) {
	t.Parallel()
	if Win95.HasUndo() || !WinXP.HasUndo() {
		t.Error("only the XP edition has undo")
	}
}
