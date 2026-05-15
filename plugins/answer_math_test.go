package plugins

import (
	"context"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestAnswerMath(t *testing.T) {
	tests := []struct {
		terms string
		want  string // empty => answer should remain nil
	}{
		{"2 + 2", "4"},
		{"12 * 7", "84"},
		{"100 / 4", "25"},
		{"(3 + 4) * 2", "14"},
		{"-5 + 10", "5"},
		{"two plus two", ""},
		{"3 best restaurants", ""},
		{"", ""},
		{"1 / 0", ""},
	}
	for _, tc := range tests {
		t.Run(tc.terms, func(t *testing.T) {
			c := &Context{Query: query.Query{Terms: tc.terms}}
			if err := (AnswerMath{}).Apply(context.Background(), c); err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if tc.want == "" {
				if c.Answer != nil {
					t.Fatalf("expected nil answer, got %+v", c.Answer)
				}
				return
			}
			if c.Answer == nil {
				t.Fatalf("expected answer %q, got nil", tc.want)
			}
			if c.Answer.Text != tc.want {
				t.Fatalf("answer = %q, want %q", c.Answer.Text, tc.want)
			}
		})
	}
}
