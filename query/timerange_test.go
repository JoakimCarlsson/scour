package query

import "testing"

func TestParseTimeRange(t *testing.T) {
	cases := map[string]TimeRange{
		"":      TimeRangeAny,
		"day":   TimeRangeDay,
		"d":     TimeRangeDay,
		"WEEK":  TimeRangeWeek,
		"month": TimeRangeMonth,
		"year":  TimeRangeYear,
	}
	for in, want := range cases {
		got, ok := ParseTimeRange(in)
		if !ok || got != want {
			t.Errorf("ParseTimeRange(%q) = (%v, %v), want (%v, true)", in, got, ok, want)
		}
	}
	if _, ok := ParseTimeRange("nope"); ok {
		t.Error("ParseTimeRange(\"nope\") ok = true, want false")
	}
}

func TestParseQueryTimeRange(t *testing.T) {
	q, err := Parse("golang :day", defaultPrefs())
	if err != nil {
		t.Fatal(err)
	}
	if q.TimeRange != TimeRangeDay {
		t.Fatalf("TimeRange = %q, want day", q.TimeRange)
	}
	if q.Terms != "golang" {
		t.Fatalf("Terms = %q, want golang", q.Terms)
	}
}
