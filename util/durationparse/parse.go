// Package durationparse provides a flexible duration parser that understands
// strings such as "1h30m", "90s", "2d4h" and "1w", with an optional leading
// sign and decimal fractions.
package durationparse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Options configures ParseFlexibleDuration. The zero value is a reasonable
// default: no default unit, negatives and fractions rejected, no day limit,
// non-strict, terse errors.
type Options struct {
	// DefaultUnit is applied to a bare number with no unit suffix. It must be
	// one of the recognised unit strings ("ns", "us", "ms", "s", "m", "h",
	// "d", "w"). If empty, a bare number is an error.
	DefaultUnit string
	// AllowNegative permits a leading '-' sign.
	AllowNegative bool
	// AllowFraction permits decimal fractions such as "1.5h".
	AllowFraction bool
	// MaxDays, if greater than zero, rejects results whose absolute value
	// exceeds this many days.
	MaxDays int
	// StrictMode rejects repeated units and units given in increasing
	// (small-to-large) order.
	StrictMode bool
	// VerboseErrors includes the offending token in returned errors.
	VerboseErrors bool
}

// unit describes a recognised unit suffix: its multiplier in nanoseconds and
// its ordering rank (smaller units rank lower).
type unit struct {
	mult float64
	rank int
}

// units maps each recognised suffix to its multiplier and ordering rank.
var units = map[string]unit{
	"ns": {float64(time.Nanosecond), 0},
	"us": {float64(time.Microsecond), 1},
	"µs": {float64(time.Microsecond), 1},
	"ms": {float64(time.Millisecond), 2},
	"s":  {float64(time.Second), 3},
	"m":  {float64(time.Minute), 4},
	"h":  {float64(time.Hour), 5},
	"d":  {float64(time.Hour) * 24, 6},
	"w":  {float64(time.Hour) * 24 * 7, 7},
}

// ParseFlexibleDuration parses target according to opts and returns the total
// duration. See Options for the meaning of each setting.
func ParseFlexibleDuration(target string, opts Options) (time.Duration, error) {
	s := strings.TrimSpace(target)
	if s == "" {
		return 0, opts.errf(target, "empty input")
	}

	rest, negative, err := opts.consumeSign(s, target)
	if err != nil {
		return 0, err
	}

	var total float64
	lastRank := -1
	seenUnit := false
	for rest != "" {
		value, name, remainder, perr := opts.parseComponent(rest, target)
		if perr != nil {
			return 0, perr
		}
		u := units[name]
		if err := opts.checkOrder(seenUnit, lastRank, u.rank, name, target); err != nil {
			return 0, err
		}
		total += value * u.mult
		lastRank = u.rank
		seenUnit = true
		rest = remainder
	}

	if !seenUnit {
		return 0, opts.errf(target, "no duration components found")
	}
	if negative {
		total = -total
	}
	if err := opts.checkMaxDays(total, target); err != nil {
		return 0, err
	}
	return time.Duration(total), nil
}

// consumeSign strips a leading '+' or '-' from s, reporting whether the value
// is negative. It errors when a '-' is present but disallowed, or when a sign
// has no following digits.
func (opts Options) consumeSign(s, target string) (rest string, negative bool, err error) {
	if s[0] != '+' && s[0] != '-' {
		return s, false, nil
	}
	if s[0] == '-' {
		if !opts.AllowNegative {
			return "", false, opts.errf(target, "negative not allowed")
		}
		negative = true
	}
	rest = s[1:]
	if rest == "" {
		return "", false, opts.errf(target, "sign with no digits")
	}
	return rest, negative, nil
}

// parseComponent reads one "<number><unit>" component from the front of s and
// returns its numeric value, unit name, and the unconsumed remainder.
func (opts Options) parseComponent(s, target string) (value float64, name, rest string, err error) {
	numStr, afterNum, err := opts.scanNumber(s, target)
	if err != nil {
		return 0, "", "", err
	}
	value, convErr := strconv.ParseFloat(numStr, 64)
	if convErr != nil {
		return 0, "", "", opts.errf(target, "invalid number %q", numStr)
	}

	name, rest = scanUnit(afterNum)
	if name == "" {
		if opts.DefaultUnit == "" {
			return 0, "", "", opts.errf(target, "missing unit for %q", numStr)
		}
		name = opts.DefaultUnit
	}
	if _, ok := units[name]; !ok {
		return 0, "", "", opts.errf(target, "unknown unit %q", name)
	}
	return value, name, rest, nil
}

// scanNumber consumes the leading numeric literal of s (digits with at most one
// dot when fractions are allowed) and returns it plus the remainder.
func (opts Options) scanNumber(s, target string) (numStr, rest string, err error) {
	i, sawDot := 0, false
scan:
	for i < len(s) {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			i++
		case c == '.' && !opts.AllowFraction:
			return "", "", opts.errf(target, "fraction not allowed")
		case c == '.' && sawDot:
			return "", "", opts.errf(target, "multiple dots in number")
		case c == '.':
			sawDot = true
			i++
		default:
			break scan
		}
	}
	if i == 0 {
		return "", "", opts.errf(target, "expected number at %q", s)
	}
	return s[:i], s[i:], nil
}

// scanUnit consumes the leading run of letters of s and returns it plus the
// remainder.
func scanUnit(s string) (name, rest string) {
	i := 0
	for i < len(s) {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == 0xC2 || c == 0xB5 {
			i++
			continue
		}
		break
	}
	return s[:i], s[i:]
}

// checkOrder enforces StrictMode's no-repeat, largest-first ordering rule.
func (opts Options) checkOrder(seenUnit bool, lastRank, rank int, name, target string) error {
	if !opts.StrictMode || !seenUnit {
		return nil
	}
	if rank == lastRank {
		return opts.errf(target, "repeated unit %q in strict mode", name)
	}
	if rank > lastRank {
		return opts.errf(target, "out-of-order unit %q in strict mode", name)
	}
	return nil
}

// checkMaxDays enforces the MaxDays cap when it is set.
func (opts Options) checkMaxDays(total float64, target string) error {
	if opts.MaxDays <= 0 {
		return nil
	}
	limit := float64(opts.MaxDays) * float64(time.Hour) * 24
	abs := total
	if abs < 0 {
		abs = -abs
	}
	if abs > limit {
		return opts.errf(target, "result exceeds maxDays")
	}
	return nil
}

// errf builds an error, appending the offending target when VerboseErrors is
// set.
func (opts Options) errf(target, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	if opts.VerboseErrors {
		return fmt.Errorf("durationparse: %s in %q", msg, target)
	}
	return errors.New("durationparse: " + msg)
}
