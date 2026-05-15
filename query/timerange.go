package query

import "strings"

type TimeRange string

const (
	TimeRangeAny   TimeRange = ""
	TimeRangeDay   TimeRange = "day"
	TimeRangeWeek  TimeRange = "week"
	TimeRangeMonth TimeRange = "month"
	TimeRangeYear  TimeRange = "year"
)

func ParseTimeRange(s string) (TimeRange, bool) {
	switch strings.ToLower(s) {
	case "", "any":
		return TimeRangeAny, true
	case "day", "d":
		return TimeRangeDay, true
	case "week", "w":
		return TimeRangeWeek, true
	case "month", "m":
		return TimeRangeMonth, true
	case "year", "y":
		return TimeRangeYear, true
	default:
		return TimeRangeAny, false
	}
}
