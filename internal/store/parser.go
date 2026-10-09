package store

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

// parser reads the small subset of JSON a saved document is written in.
// It is deliberately strict about structure and forgiving about content:
// a key it does not know is skipped whole, so a document from a newer
// cellmate still loads the fields this one understands.
type parser struct {
	s string
	i int
}

func (p *parser) space() {
	for p.i < len(p.s) {
		switch p.s[p.i] {
		case ' ', '\t', '\r', '\n':
			p.i++
		default:
			return
		}
	}
}

func (p *parser) byteIs(c byte) bool {
	p.space()
	if p.i < len(p.s) && p.s[p.i] == c {
		p.i++
		return true
	}
	return false
}

func (p *parser) want(c byte) error {
	if p.byteIs(c) {
		return nil
	}
	return fmt.Errorf("want %q at offset %d", c, p.i)
}

// object reads an object, calling field for each key with the parser
// positioned at that key's value.
func (p *parser) object(field func(key string) error) error {
	if err := p.want('{'); err != nil {
		return err
	}
	if p.byteIs('}') {
		return nil
	}
	for {
		key, err := p.text()
		if err != nil {
			return err
		}
		if err := p.want(':'); err != nil {
			return err
		}
		if err := field(key); err != nil {
			return err
		}
		if p.byteIs(',') {
			continue
		}
		return p.want('}')
	}
}

func (p *parser) text() (string, error) {
	if err := p.want('"'); err != nil {
		return "", err
	}
	var b strings.Builder
	for p.i < len(p.s) {
		c := p.s[p.i]
		switch c {
		case '"':
			p.i++
			return b.String(), nil
		case '\\':
			p.i++
			if p.i >= len(p.s) {
				return "", fmt.Errorf("string ends inside an escape")
			}
			switch p.s[p.i] {
			case '"', '\\', '/':
				b.WriteByte(p.s[p.i])
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'u':
				r, err := p.escape()
				if err != nil {
					return "", err
				}
				b.WriteRune(r)
				continue
			default:
				return "", fmt.Errorf("unknown escape %q", p.s[p.i])
			}
			p.i++
		default:
			b.WriteByte(c)
			p.i++
		}
	}
	return "", fmt.Errorf("string is not closed")
}

// escape reads a \u escape, joining a surrogate pair when one follows. It is
// entered with the cursor on the 'u', which hex4 consumes.
func (p *parser) escape() (rune, error) {
	u, err := p.hex4()
	if err != nil {
		return 0, err
	}
	if utf16.IsSurrogate(rune(u)) && strings.HasPrefix(p.s[p.i:], `\u`) {
		save := p.i
		// Step over the backslash only: hex4 consumes the 'u' itself
		p.i++
		if v, err := p.hex4(); err == nil {
			if r := utf16.DecodeRune(rune(u), rune(v)); r != 0xFFFD {
				return r, nil
			}
		}
		p.i = save
	}
	// A surrogate with no partner is not a character. WriteRune turns it
	// into the replacement character, which is what the caller stores.
	return rune(u), nil
}

// hex4 reads the 'u' at the cursor and the four hex digits after it.
func (p *parser) hex4() (uint64, error) {
	p.i++ // Skip the 'u'
	if p.i+4 > len(p.s) {
		return 0, fmt.Errorf("truncated \\u escape")
	}
	v, err := strconv.ParseUint(p.s[p.i:p.i+4], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("bad \\u escape: %w", err)
	}
	p.i += 4
	return v, nil
}

func (p *parser) number() (int, error) {
	p.space()
	start := p.i
	if p.i < len(p.s) && (p.s[p.i] == '-' || p.s[p.i] == '+') {
		p.i++
	}
	for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		p.i++
	}
	if start == p.i {
		return 0, fmt.Errorf("want a number at offset %d", start)
	}
	n, err := strconv.Atoi(p.s[start:p.i])
	if err != nil {
		return 0, fmt.Errorf("bad number at offset %d: %w", start, err)
	}
	return n, nil
}

func (p *parser) boolean() (bool, error) {
	p.space()
	switch {
	case strings.HasPrefix(p.s[p.i:], "true"):
		p.i += 4
		return true, nil
	case strings.HasPrefix(p.s[p.i:], "false"):
		p.i += 5
		return false, nil
	}
	return false, fmt.Errorf("want true or false at offset %d", p.i)
}

// skip discards the value at the cursor, whatever its shape.
func (p *parser) skip() error {
	p.space()
	if p.i >= len(p.s) {
		return fmt.Errorf("want a value at offset %d", p.i)
	}
	switch c := p.s[p.i]; {
	case c == '"':
		_, err := p.text()
		return err
	case c == '{':
		return p.object(func(string) error { return p.skip() })
	case c == '[':
		p.i++
		if p.byteIs(']') {
			return nil
		}
		for {
			if err := p.skip(); err != nil {
				return err
			}
			if p.byteIs(',') {
				continue
			}
			return p.want(']')
		}
	case strings.HasPrefix(p.s[p.i:], "null"):
		p.i += 4
		return nil
	case c == 't' || c == 'f':
		_, err := p.boolean()
		return err
	default:
		// A number, possibly with a fraction or exponent this document
		// never writes but a hand-edited one might.
		start := p.i
		for p.i < len(p.s) && strings.IndexByte("+-.eE0123456789", p.s[p.i]) >= 0 {
			p.i++
		}
		if start == p.i {
			return fmt.Errorf("want a value at offset %d", start)
		}
		return nil
	}
}
