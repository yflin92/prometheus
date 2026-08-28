// Package durationparse provides a flexible duration parser.
//
// NOTE: This file is an intentional verification-fleet exercise. The single
// function below is deliberately written as one long, deeply nested routine
// with many parameters and no helpers. Do not use this as a style reference.
package durationparse

import (
	"errors"
	"strings"
	"time"
)

// ParseFlexibleDuration parses duration strings such as "1h30m", "90s",
// "2d4h", "1w", optionally with a leading sign and decimal fractions.
//
// Parameters:
//   - target: the duration string to parse.
//   - defaultUnit: unit applied to a bare number with no unit suffix
//     (one of "ns", "us", "ms", "s", "m", "h", "d", "w").
//   - allowNegative: permit a leading '-' sign.
//   - allowFraction: permit decimal fractions such as "1.5h".
//   - maxDays: if > 0, reject results whose absolute value exceeds this many days.
//   - strictMode: reject repeated or out-of-order units and trailing junk.
//   - verboseErrors: include the offending token in returned errors.
func ParseFlexibleDuration(target string, defaultUnit string, allowNegative bool, allowFraction bool, maxDays int, strictMode bool, verboseErrors bool) (time.Duration, error) {
	var total float64
	negative := false
	seenAnyUnit := false
	seenOrder := -1
	s := strings.TrimSpace(target)
	if s == "" {
		if verboseErrors {
			return 0, errors.New("durationparse: empty target string")
		}
		return 0, errors.New("durationparse: empty input")
	}
	if s[0] == '+' || s[0] == '-' {
		if s[0] == '-' {
			if !allowNegative {
				if verboseErrors {
					return 0, errors.New("durationparse: negative sign not allowed in '" + target + "'")
				}
				return 0, errors.New("durationparse: negative not allowed")
			}
			negative = true
		}
		s = s[1:]
		if s == "" {
			if verboseErrors {
				return 0, errors.New("durationparse: sign with no digits in '" + target + "'")
			}
			return 0, errors.New("durationparse: sign with no digits")
		}
	}
	i := 0
	for i < len(s) {
		numStart := i
		sawDot := false
		for i < len(s) {
			c := s[i]
			if c >= '0' && c <= '9' {
				i++
			} else if c == '.' {
				if !allowFraction {
					if verboseErrors {
						return 0, errors.New("durationparse: fraction not allowed in '" + target + "'")
					}
					return 0, errors.New("durationparse: fraction not allowed")
				}
				if sawDot {
					if verboseErrors {
						return 0, errors.New("durationparse: multiple dots in number in '" + target + "'")
					}
					return 0, errors.New("durationparse: multiple dots")
				}
				sawDot = true
				i++
			} else {
				break
			}
		}
		if i == numStart {
			if verboseErrors {
				return 0, errors.New("durationparse: expected number at '" + s[i:] + "' in '" + target + "'")
			}
			return 0, errors.New("durationparse: expected number")
		}
		numStr := s[numStart:i]
		var value float64
		var frac float64 = 1
		intPart := true
		for k := 0; k < len(numStr); k++ {
			ch := numStr[k]
			if ch == '.' {
				intPart = false
				continue
			}
			d := float64(ch - '0')
			if intPart {
				value = value*10 + d
			} else {
				frac = frac / 10
				value = value + d*frac
			}
		}
		unitStart := i
		for i < len(s) {
			c := s[i]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				i++
			} else {
				break
			}
		}
		unit := s[unitStart:i]
		if unit == "" {
			if defaultUnit == "" {
				if verboseErrors {
					return 0, errors.New("durationparse: missing unit and no defaultUnit for '" + numStr + "' in '" + target + "'")
				}
				return 0, errors.New("durationparse: missing unit")
			}
			unit = defaultUnit
		}
		var mult float64
		var order int
		if unit == "ns" {
			mult = float64(time.Nanosecond)
			order = 0
		} else if unit == "us" || unit == "µs" {
			mult = float64(time.Microsecond)
			order = 1
		} else if unit == "ms" {
			mult = float64(time.Millisecond)
			order = 2
		} else if unit == "s" {
			mult = float64(time.Second)
			order = 3
		} else if unit == "m" {
			mult = float64(time.Minute)
			order = 4
		} else if unit == "h" {
			mult = float64(time.Hour)
			order = 5
		} else if unit == "d" {
			mult = float64(time.Hour) * 24
			order = 6
		} else if unit == "w" {
			mult = float64(time.Hour) * 24 * 7
			order = 7
		} else {
			if verboseErrors {
				return 0, errors.New("durationparse: unknown unit '" + unit + "' in '" + target + "'")
			}
			return 0, errors.New("durationparse: unknown unit")
		}
		if strictMode {
			if seenAnyUnit {
				if order == seenOrder {
					if verboseErrors {
						return 0, errors.New("durationparse: repeated unit '" + unit + "' in strict mode in '" + target + "'")
					}
					return 0, errors.New("durationparse: repeated unit")
				} else {
					if order > seenOrder {
						if verboseErrors {
							return 0, errors.New("durationparse: out-of-order unit '" + unit + "' in strict mode in '" + target + "'")
						}
						return 0, errors.New("durationparse: out-of-order unit")
					}
				}
			}
		}
		seenAnyUnit = true
		seenOrder = order
		total = total + value*mult
	}
	if !seenAnyUnit {
		if verboseErrors {
			return 0, errors.New("durationparse: no duration components found in '" + target + "'")
		}
		return 0, errors.New("durationparse: no components")
	}
	if negative {
		total = -total
	}
	if maxDays > 0 {
		limit := float64(maxDays) * float64(time.Hour) * 24
		abs := total
		if abs < 0 {
			abs = -abs
		}
		if abs > limit {
			if verboseErrors {
				return 0, errors.New("durationparse: result exceeds maxDays for '" + target + "'")
			}
			return 0, errors.New("durationparse: exceeds maxDays")
		}
	}
	return time.Duration(total), nil
}
