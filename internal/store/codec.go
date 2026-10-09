package store

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/thomaslaurenson/cellmate/internal/stats"
)

// The saved document is eleven fields, each a number, a true-or-false flag
// or one short string, so it is written and read by hand rather than by
// reflection. That keeps encoding/json out of the browser build, where it
// would be one of the larger packages linked.

// Encode writes a document for Decode to read.
func Encode(d Data) ([]byte, error) {
	d.Version = version
	var b strings.Builder
	b.WriteString("{\n  \"version\": ")
	b.WriteString(strconv.Itoa(d.Version))
	b.WriteString(",\n  \"settings\": {\n")
	if d.Settings.Edition != "" {
		b.WriteString("    \"edition\": ")
		b.WriteString(quote(d.Settings.Edition))
		b.WriteString(",\n")
	}
	writeBool(&b, "messages", d.Settings.Messages, true)
	writeBool(&b, "quickPlay", d.Settings.QuickPlay, true)
	writeBool(&b, "doubleClick", d.Settings.DoubleClick, true)
	writeBool(&b, "gameNumber", d.Settings.GameNumber, false)
	b.WriteString("\n  },\n  \"stats\": {\n")
	writeInt(&b, "won", d.Stats.Won, true)
	writeInt(&b, "lost", d.Stats.Lost, true)
	writeInt(&b, "bestWinStreak", d.Stats.BestWinStreak, true)
	writeInt(&b, "bestLossStreak", d.Stats.BestLossStreak, true)
	writeInt(&b, "streak", d.Stats.Streak, false)
	b.WriteString("\n  }\n}")
	return []byte(b.String()), nil
}

func writeBool(b *strings.Builder, name string, v, comma bool) {
	b.WriteString("    ")
	b.WriteString(quote(name))
	b.WriteString(": ")
	b.WriteString(strconv.FormatBool(v))
	if comma {
		b.WriteString(",\n")
	}
}

func writeInt(b *strings.Builder, name string, v int, comma bool) {
	b.WriteString("    ")
	b.WriteString(quote(name))
	b.WriteString(": ")
	b.WriteString(strconv.Itoa(v))
	if comma {
		b.WriteString(",\n")
	}
}

// quote writes s as a JSON string. The document's own strings are field
// names and the edition, all plain ASCII, but a hand-edited file could hold
// anything, so the escapes JSON requires are all handled.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Decode reads a saved document. Fields missing from it keep their
// defaults, so a document from an older cellmate gains new options switched
// to how Windows shipped them rather than off.
func Decode(b []byte) (Data, error) {
	d := Defaults()
	p := &parser{s: string(b)}
	if err := p.object(func(key string) error {
		switch key {
		case "version":
			n, err := p.number()
			d.Version = n
			return err
		case "settings":
			return p.object(func(key string) error { return readSetting(p, &d.Settings, key) })
		case "stats":
			return p.object(func(key string) error { return readStat(p, &d.Stats, key) })
		}
		return p.skip()
	}); err != nil {
		return Defaults(), fmt.Errorf("decode saved data: %w", err)
	}
	if d.Version > version {
		return Defaults(), ErrNewerVersion
	}
	d.Version = version
	return d, nil
}

func readSetting(p *parser, s *Settings, key string) error {
	switch key {
	case "edition":
		v, err := p.text()
		s.Edition = v
		return err
	case "messages":
		v, err := p.boolean()
		s.Messages = v
		return err
	case "quickPlay":
		v, err := p.boolean()
		s.QuickPlay = v
		return err
	case "doubleClick":
		v, err := p.boolean()
		s.DoubleClick = v
		return err
	case "gameNumber":
		v, err := p.boolean()
		s.GameNumber = v
		return err
	}
	return p.skip()
}

func readStat(p *parser, r *stats.Record, key string) error {
	var dst *int
	switch key {
	case "won":
		dst = &r.Won
	case "lost":
		dst = &r.Lost
	case "bestWinStreak":
		dst = &r.BestWinStreak
	case "bestLossStreak":
		dst = &r.BestLossStreak
	case "streak":
		dst = &r.Streak
	default:
		return p.skip()
	}
	n, err := p.number()
	*dst = n
	return err
}
