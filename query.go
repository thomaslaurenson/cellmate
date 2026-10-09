package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/thomaslaurenson/cellmate/internal/edition"
	"github.com/thomaslaurenson/cellmate/internal/ui"
)

// optionsFromQuery reads the page query, which stands in for the command line
// in the browser: ?edition=95&game=617 opens deal 617 in 95 mode. The edition
// comes from the query when it names one, then from the one saved in Options,
// then the default.
//
// A page cannot refuse to start the way a command can, so an error comes with
// options that are still fit to play: the edition settled so far and a
// random deal.
//
// Only the browser build calls this. It carries no build tag so that the
// ordinary test run covers it, which a file behind the js tag never gets.
func optionsFromQuery(query, savedEdition string) (ui.Options, error) {
	opts := ui.Options{Edition: edition.Default}
	if ed, err := edition.Parse(savedEdition); err == nil {
		opts.Edition = ed
	}
	q, err := url.ParseQuery(strings.TrimPrefix(query, "?"))
	if err != nil {
		return opts, fmt.Errorf("parse page query %q: %w", query, err)
	}
	if s := q.Get("edition"); s != "" {
		ed, err := edition.Parse(s)
		if err != nil {
			return opts, err
		}
		opts.Edition = ed
	}
	if s := q.Get("game"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			return opts, fmt.Errorf("game %q is not a whole number", s)
		}
		if err := opts.Edition.CheckDeal(n); err != nil {
			return opts, err
		}
		opts.Deal = n
	}
	return opts, nil
}
