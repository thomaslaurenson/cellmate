// Package edition describes the Windows releases of FreeCell that cellmate
// reproduces, and the rules that differ between them.
package edition

import (
	"errors"
	"fmt"

	"github.com/thomaslaurenson/cellmate/internal/deal"
)

// Edition is a Windows release of FreeCell.
type Edition int

// The editions cellmate reproduces. The zero value is not an edition, so a
// forgotten assignment fails validation rather than quietly picking one.
const (
	Win95 Edition = iota + 1
	WinXP
)

// Default is the edition used when none is chosen.
const Default = WinXP

// ErrUnknown is returned by Parse for a name that is not an edition.
var ErrUnknown = errors.New("unknown edition")

// All lists every edition, oldest first.
var All = []Edition{Win95, WinXP}

// Parse returns the edition named s, as written by String.
func Parse(s string) (Edition, error) {
	for _, e := range All {
		if e.String() == s {
			return e, nil
		}
	}
	return 0, fmt.Errorf("%w %q, want 95 or xp", ErrUnknown, s)
}

// String returns the short name used on the command line: "95" or "xp".
func (e Edition) String() string {
	switch e {
	case Win95:
		return "95"
	case WinXP:
		return "xp"
	default:
		return fmt.Sprintf("Edition(%d)", int(e))
	}
}

// MaxDeal returns the highest deal number the edition offers. Windows 95
// offered 32,000 deals and Windows XP extended the range to a million.
func (e Edition) MaxDeal() int {
	if e == Win95 {
		return 32000
	}
	return 1000000
}

// HasUndo reports whether the edition offers Undo. Windows XP had it; sources
// disagree about Windows 95, and cellmate follows those that say it did not.
func (e Edition) HasUndo() bool { return e == WinXP }

// DealRangeError reports a deal number outside an edition's range.
type DealRangeError struct {
	Deal    int
	Edition Edition
}

func (e *DealRangeError) Error() string {
	return fmt.Sprintf("deal %d is outside 1 to %d for edition %s", e.Deal, e.Edition.MaxDeal(), e.Edition)
}

// CheckDeal returns a *DealRangeError when n is not a deal the edition
// offers: the numbered deals from 1 to MaxDeal, and the two impossible deals
// that both editions hid at -1 and -2.
func (e Edition) CheckDeal(n int) error {
	if deal.Impossible(n) {
		return nil
	}
	if n < 1 || n > e.MaxDeal() {
		return &DealRangeError{Deal: n, Edition: e}
	}
	return nil
}
