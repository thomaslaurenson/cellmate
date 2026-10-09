package stats

import "testing"

func TestRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		games   string // w for a win, l for a loss
		want    Record
		percent int
		current string
	}{
		{name: "empty", games: "", want: Record{}, percent: 0, current: "none"},
		{name: "one win", games: "w", want: Record{Won: 1, BestWinStreak: 1, Streak: 1}, percent: 100, current: "1 win"},
		{name: "one loss", games: "l", want: Record{Lost: 1, BestLossStreak: 1, Streak: -1}, percent: 0, current: "1 loss"},
		{
			name: "streaks reset on a change", games: "wwwllw",
			want:    Record{Won: 4, Lost: 2, BestWinStreak: 3, BestLossStreak: 2, Streak: 1},
			percent: 66, current: "1 win",
		},
		{
			name: "losing run", games: "wlll",
			want:    Record{Won: 1, Lost: 3, BestWinStreak: 1, BestLossStreak: 3, Streak: -3},
			percent: 25, current: "3 losses",
		},
		{
			name: "winning run", games: "lww",
			want:    Record{Won: 2, Lost: 1, BestWinStreak: 2, BestLossStreak: 1, Streak: 2},
			percent: 66, current: "2 wins",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var r Record
			for _, g := range tc.games {
				if g == 'w' {
					r.Win()
				} else {
					r.Lose()
				}
			}
			if r != tc.want {
				t.Errorf("record = %+v, want %+v", r, tc.want)
			}
			if got := r.Percent(); got != tc.percent {
				t.Errorf("Percent() = %d, want %d", got, tc.percent)
			}
			if got := r.Current(); got != tc.current {
				t.Errorf("Current() = %q, want %q", got, tc.current)
			}
		})
	}
}
