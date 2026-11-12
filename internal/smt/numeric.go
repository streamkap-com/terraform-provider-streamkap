package smt

import (
	"math"
	"strconv"
	"strings"
)

const (
	minJSONSafeInteger = int64(-9007199254740991)
	maxJSONSafeInteger = int64(9007199254740991)
)

func isJSONSafeInteger(value int64) bool {
	return value >= minJSONSafeInteger && value <= maxJSONSafeInteger
}

func jsonSafeIntegerFromFloat(value float64) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value != math.Trunc(value) || value < float64(minJSONSafeInteger) || value > float64(maxJSONSafeInteger) {
		return 0, false
	}
	return int64(value), true
}

func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func exactJSONSafeIntegerLiteral(literal string) (int64, bool) {
	negative := strings.HasPrefix(literal, "-")
	unsigned := strings.TrimPrefix(literal, "-")
	mantissa, exponentText := unsigned, ""
	if exponentAt := strings.IndexAny(unsigned, "eE"); exponentAt >= 0 {
		mantissa, exponentText = unsigned[:exponentAt], unsigned[exponentAt+1:]
	}
	decimalPlaces := 0
	if decimalAt := strings.IndexByte(mantissa, '.'); decimalAt >= 0 {
		decimalPlaces = len(mantissa) - decimalAt - 1
		mantissa = mantissa[:decimalAt] + mantissa[decimalAt+1:]
	}
	for _, digit := range mantissa {
		if digit < '0' || digit > '9' {
			return 0, false
		}
	}
	digits := strings.TrimLeft(mantissa, "0")
	if digits == "" {
		return 0, true
	}
	exponent, ok := decimalExponent(exponentText, len(digits)+decimalPlaces+len("9007199254740991"))
	if !ok {
		return 0, false
	}
	scale := exponent - decimalPlaces
	if scale < 0 {
		trim := -scale
		if trim >= len(digits) || strings.Trim(digits[len(digits)-trim:], "0") != "" {
			return 0, false
		}
		digits = digits[:len(digits)-trim]
	} else if scale > 0 {
		if len(digits)+scale > len("9007199254740991") {
			return 0, false
		}
		digits += strings.Repeat("0", scale)
	}
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return 0, true
	}
	if len(digits) > len("9007199254740991") {
		return 0, false
	}
	magnitude, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || magnitude > maxJSONSafeInteger {
		return 0, false
	}
	integer := magnitude
	if negative {
		integer = -integer
	}
	return integer, isJSONSafeInteger(integer)
}

func decimalExponent(raw string, limit int) (int, bool) {
	if raw == "" {
		return 0, true
	}
	negative := strings.HasPrefix(raw, "-")
	raw = strings.TrimPrefix(strings.TrimPrefix(raw, "-"), "+")
	if raw == "" {
		return 0, false
	}
	value := 0
	for _, digit := range raw {
		if digit < '0' || digit > '9' {
			return 0, false
		}
		value = value*10 + int(digit-'0')
		if value > limit {
			return 0, false
		}
	}
	if negative {
		value = -value
	}
	return value, true
}
