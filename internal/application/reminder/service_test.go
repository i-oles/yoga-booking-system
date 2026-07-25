package reminder

import (
	"testing"
	"time"
)

func TestIsTimeToRemind(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		start time.Time
		want  bool
	}{
		{
			name:  "within 24h (10h)",
			start: now.Add(10 * time.Hour),
			want:  true,
		},
		{
			name:  "just below 24h",
			start: now.Add(24*time.Hour - time.Nanosecond),
			want:  true,
		},
		{
			name:  "exactly 24h",
			start: now.Add(24 * time.Hour),
			want:  false,
		},
		{
			name:  "just above 24h",
			start: now.Add(24*time.Hour + time.Nanosecond),
			want:  false,
		},
		{
			name:  "exactly now",
			start: now,
			want:  false,
		},
		{
			name:  "in the past",
			start: now.Add(-1 * time.Hour),
			want:  false,
		},
		{
			name:  "zero time",
			start: time.Time{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isTimeToRemind(tt.start, now)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBookedSameOrPreviousDayAsClassDay(t *testing.T) {
	t.Parallel()

	loc := time.UTC

	tests := []struct {
		name string
		a    time.Time
		b    time.Time
		want bool
	}{
		{
			name: "same day",
			a:    time.Date(2026, 4, 6, 10, 0, 0, 0, loc),
			b:    time.Date(2026, 4, 6, 18, 0, 0, 0, loc),
			want: true,
		},
		{
			name: "previous day",
			a:    time.Date(2026, 4, 5, 23, 59, 0, 0, loc),
			b:    time.Date(2026, 4, 6, 0, 1, 0, 0, loc),
			want: true,
		},
		{
			name: "two days before",
			a:    time.Date(2026, 4, 4, 12, 0, 0, 0, loc),
			b:    time.Date(2026, 4, 6, 12, 0, 0, 0, loc),
			want: false,
		},
		{
			name: "next day",
			a:    time.Date(2026, 4, 7, 10, 0, 0, 0, loc),
			b:    time.Date(2026, 4, 6, 10, 0, 0, 0, loc),
			want: false,
		},
		{
			name: "zero time values",
			a:    time.Time{},
			b:    time.Time{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isBookedSameOrPreviousDayAsClassDay(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
