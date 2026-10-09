package store

import (
	"errors"
	"strings"
	"testing"

	"github.com/thomaslaurenson/cellmate/internal/stats"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data Data
	}{
		{name: "defaults", data: Defaults()},
		{
			name: "every option on",
			data: Data{
				Version:  version,
				Settings: Settings{Edition: "95", Messages: true, QuickPlay: true, DoubleClick: true, GameNumber: true},
				Stats:    stats.Record{Won: 3, Lost: 2, BestWinStreak: 3, BestLossStreak: 1, Streak: 2},
			},
		},
		{
			name: "every option off and a losing streak",
			data: Data{
				Version:  version,
				Settings: Settings{Edition: "xp"},
				Stats:    stats.Record{Won: 0, Lost: 7, BestLossStreak: 7, Streak: -7},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b, err := Encode(tc.data)
			if err != nil {
				t.Fatalf("Encode returned %v", err)
			}
			got, err := Decode(b)
			if err != nil {
				t.Fatalf("Decode(%s) returned %v", b, err)
			}
			if got != tc.data {
				t.Errorf("round trip gave %+v, want %+v", got, tc.data)
			}
		})
	}
}

func TestDecodeKeepsDefaultsForMissingFields(t *testing.T) {
	t.Parallel()
	// A document from an older cellmate has no doubleClick, which must come
	// back switched to how Windows shipped it rather than off.
	got, err := Decode([]byte(`{"version":1,"settings":{"quickPlay":true}}`))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	want := Defaults()
	want.Settings.QuickPlay = true
	if got != want {
		t.Errorf("decoded %+v, want %+v", got, want)
	}
}

func TestDecodeSkipsUnknownFields(t *testing.T) {
	t.Parallel()
	// A field a newer cellmate added is stepped over whatever its shape
	in := `{"version":1,"theme":{"name":"dark","sizes":[1,2,3]},"debug":null,
	        "settings":{"messages":false,"sound":true},"scale":1.5}`
	got, err := Decode([]byte(in))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if got.Settings.Messages {
		t.Error("messages came back true, want the false the document carried")
	}
	if !got.Settings.DoubleClick {
		t.Error("doubleClick came back false, want the default true")
	}
}

func TestDecodeRefusesANewerDocument(t *testing.T) {
	t.Parallel()
	got, err := Decode([]byte(`{"version":99}`))
	if !errors.Is(err, ErrNewerVersion) {
		t.Fatalf("Decode returned %v, want ErrNewerVersion", err)
	}
	if got != Defaults() {
		t.Errorf("refused document gave %+v, want the defaults", got)
	}
}

func TestDecodeRejectsMalformedDocuments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "not an object", in: `[1,2,3]`},
		{name: "unclosed object", in: `{"version":1`},
		{name: "unclosed string", in: `{"version`},
		{name: "missing colon", in: `{"version" 1}`},
		{name: "version is not a number", in: `{"version":"one"}`},
		{name: "setting is not a boolean", in: `{"settings":{"messages":"yes"}}`},
		{name: "trailing comma", in: `{"version":1,}`},
		{name: "unclosed array in an unknown field", in: `{"extra":[1,2`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Decode([]byte(tc.in))
			if err == nil {
				t.Fatalf("Decode(%q) returned no error", tc.in)
			}
			if got != Defaults() {
				t.Errorf("failed decode gave %+v, want the defaults", got)
			}
		})
	}
}

func TestDecodeReadsStringEscapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: `"95"`, want: "95"},
		{name: "quote", in: `"a\"b"`, want: `a"b`},
		{name: "backslash", in: `"a\\b"`, want: `a\b`},
		{name: "control", in: `"a\nb"`, want: "a\nb"},
		{name: "short unicode escape", in: `"\u0041"`, want: "A"},
		{name: "surrogate pair", in: `"\ud83c\udfb4"`, want: "\U0001F3B4"},
		{name: "unpaired surrogate", in: `"\ud83c"`, want: "\uFFFD"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Decode([]byte(`{"settings":{"edition":` + tc.in + `}}`))
			if err != nil {
				t.Fatalf("Decode returned %v", err)
			}
			if got.Settings.Edition != tc.want {
				t.Errorf("edition = %q, want %q", got.Settings.Edition, tc.want)
			}
		})
	}
}

func TestEncodeQuotesAwkwardStrings(t *testing.T) {
	t.Parallel()
	d := Defaults()
	d.Settings.Edition = "a\"b\\c\nd"

	b, err := Encode(d)
	if err != nil {
		t.Fatalf("Encode returned %v", err)
	}
	got, err := Decode(b)
	if err != nil {
		t.Fatalf("Decode of the encoded document returned %v", err)
	}
	if got.Settings.Edition != d.Settings.Edition {
		t.Errorf("edition survived encoding as %q, want %q", got.Settings.Edition, d.Settings.Edition)
	}
}

func TestEncodeOmitsAnEmptyEdition(t *testing.T) {
	t.Parallel()
	b, err := Encode(Defaults())
	if err != nil {
		t.Fatalf("Encode returned %v", err)
	}
	if strings.Contains(string(b), "edition") {
		t.Errorf("encoded an empty edition:\n%s", b)
	}
}

func TestEncodeStampsTheCurrentVersion(t *testing.T) {
	t.Parallel()
	d := Defaults()
	d.Version = 0

	b, err := Encode(d)
	if err != nil {
		t.Fatalf("Encode returned %v", err)
	}
	got, err := Decode(b)
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if got.Version != version {
		t.Errorf("version = %d, want %d", got.Version, version)
	}
}
