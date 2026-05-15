package query

import "strings"

type SafeLevel int

const (
	SafeOff SafeLevel = iota
	SafeModerate
	SafeStrict
)

func (s SafeLevel) String() string {
	switch s {
	case SafeOff:
		return "off"
	case SafeModerate:
		return "moderate"
	case SafeStrict:
		return "strict"
	default:
		return "unknown"
	}
}

func ParseSafeLevel(s string) (SafeLevel, bool) {
	switch strings.ToLower(s) {
	case "off":
		return SafeOff, true
	case "moderate":
		return SafeModerate, true
	case "strict":
		return SafeStrict, true
	default:
		return SafeOff, false
	}
}
