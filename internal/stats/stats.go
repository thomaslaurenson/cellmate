// Package stats keeps the win and loss record shown in the Statistics box.
package stats

import "strconv"

// Record is a tally of games won and lost, with the longest and current
// streaks. The zero value is an empty record.
type Record struct {
	Won            int
	Lost           int
	BestWinStreak  int
	BestLossStreak int
	// Streak is the current run: positive counts wins, negative losses.
	Streak int
}

// Win records a won game.
func (r *Record) Win() {
	r.Won++
	r.Streak = max(r.Streak, 0) + 1
	r.BestWinStreak = max(r.BestWinStreak, r.Streak)
}

// Lose records a lost game.
func (r *Record) Lose() {
	r.Lost++
	r.Streak = min(r.Streak, 0) - 1
	r.BestLossStreak = max(r.BestLossStreak, -r.Streak)
}

// Percent returns the share of games won, rounded down, or 0 before any
// game has finished.
func (r Record) Percent() int {
	total := r.Won + r.Lost
	if total == 0 {
		return 0
	}
	return r.Won * 100 / total
}

// Current describes the current streak, such as "3 wins" or "1 loss".
func (r Record) Current() string {
	switch {
	case r.Streak == 1:
		return "1 win"
	case r.Streak > 1:
		return strconv.Itoa(r.Streak) + " wins"
	case r.Streak == -1:
		return "1 loss"
	case r.Streak < -1:
		return strconv.Itoa(-r.Streak) + " losses"
	}
	return "none"
}
