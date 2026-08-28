package durationparse

import (
	"testing"
	"time"
)

func TestParseFlexibleDuration_Valid(t *testing.T) {
	cases := []struct {
		name   string
		target string
		opts   Options
		want   time.Duration
	}{
		{"single hour", "1h", Options{}, time.Hour},
		{"hours and minutes", "1h30m", Options{}, time.Hour + 30*time.Minute},
		{"seconds", "90s", Options{}, 90 * time.Second},
		{"days and hours", "2d4h", Options{}, 2*24*time.Hour + 4*time.Hour},
		{"one week", "1w", Options{}, 7 * 24 * time.Hour},
		{"milliseconds", "250ms", Options{}, 250 * time.Millisecond},
		{"microseconds ascii", "5us", Options{}, 5 * time.Microsecond},
		{"microseconds unicode", "5µs", Options{}, 5 * time.Microsecond},
		{"nanoseconds", "7ns", Options{}, 7 * time.Nanosecond},
		{"leading plus", "+1h", Options{}, time.Hour},
		{"whitespace trimmed", "  2h  ", Options{}, 2 * time.Hour},
		{"default unit applied", "42", Options{DefaultUnit: "s"}, 42 * time.Second},
		{"default unit with suffix mix", "1h30", Options{DefaultUnit: "m"}, time.Hour + 30*time.Minute},
		{"negative allowed", "-30m", Options{AllowNegative: true}, -30 * time.Minute},
		{"fraction allowed", "1.5h", Options{AllowFraction: true}, 90 * time.Minute},
		{"fraction seconds", "0.5s", Options{AllowFraction: true}, 500 * time.Millisecond},
		{"negative fraction", "-2.5h", Options{AllowNegative: true, AllowFraction: true}, -(2*time.Hour + 30*time.Minute)},
		{"within max days", "3d", Options{MaxDays: 5}, 3 * 24 * time.Hour},
		{"strict largest first ok", "1h30m", Options{StrictMode: true}, time.Hour + 30*time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseFlexibleDuration(tc.target, tc.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseFlexibleDuration_Errors(t *testing.T) {
	cases := []struct {
		name   string
		target string
		opts   Options
	}{
		{"empty", "", Options{}},
		{"whitespace only", "   ", Options{}},
		{"negative disallowed", "-5m", Options{}},
		{"sign with no digits", "-", Options{AllowNegative: true}},
		{"fraction disallowed", "1.5h", Options{}},
		{"multiple dots", "1.2.3h", Options{AllowFraction: true}},
		{"unknown unit", "5x", Options{}},
		{"missing unit no default", "42", Options{}},
		{"leading unit no number", "h30m", Options{}},
		{"exceeds max days", "10d", Options{MaxDays: 5}},
		{"strict repeated unit", "1h1h", Options{StrictMode: true}},
		{"strict out of order", "30m1h", Options{StrictMode: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseFlexibleDuration(tc.target, tc.opts); err == nil {
				t.Fatalf("expected error for %q, got nil", tc.target)
			}
		})
	}
}

func TestParseFlexibleDuration_VerboseErrorsIncludeTarget(t *testing.T) {
	_, err := ParseFlexibleDuration("5x", Options{VerboseErrors: true})
	if err == nil {
		t.Fatal("expected error")
	}
	if want := `"5x"`; !contains(err.Error(), want) {
		t.Fatalf("verbose error %q should contain %s", err.Error(), want)
	}

	_, err = ParseFlexibleDuration("5x", Options{VerboseErrors: false})
	if err == nil {
		t.Fatal("expected error")
	}
	if contains(err.Error(), `"5x"`) {
		t.Fatalf("terse error %q should not contain the target", err.Error())
	}
}

func TestParseFlexibleDuration_MaxDaysUsesAbsoluteValue(t *testing.T) {
	// A negative value beyond the cap must also be rejected.
	if _, err := ParseFlexibleDuration("-10d", Options{AllowNegative: true, MaxDays: 5}); err == nil {
		t.Fatal("expected negative value to be capped by MaxDays")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
