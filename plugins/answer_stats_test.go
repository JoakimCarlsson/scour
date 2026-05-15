package plugins

import (
	"context"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestAnswerStats(t *testing.T) {
	cases := []struct{ in, want string }{
		{"mean 1 2 3 4 5", "3"},
		{"mean 1 2 3 4", "2.5"},
		{"median 1 2 3", "2"},
		{"median 1 2 3 4", "2.5"},
		{"sum 1 2 3", "6"},
		{"min 5 1 3", "1"},
		{"max 5 1 3", "5"},
		{"stddev 2 4 4 4 5 5 7 9", "2"},
	}
	for _, c := range cases {
		ctx := &Context{Query: query.Query{Terms: c.in}}
		_ = (AnswerStats{}).Apply(context.Background(), ctx)
		if ctx.Answer == nil || ctx.Answer.Text != c.want {
			t.Errorf("%q: got %+v, want %q", c.in, ctx.Answer, c.want)
		}
	}
}

func TestAnswerStatsRejectMixed(t *testing.T) {
	c := &Context{Query: query.Query{Terms: "mean 1 two 3"}}
	_ = (AnswerStats{}).Apply(context.Background(), c)
	if c.Answer != nil {
		t.Fatalf("expected nil, got %+v", c.Answer)
	}
}

func TestAnswerStatsNoMatch(t *testing.T) {
	c := &Context{Query: query.Query{Terms: "what is golang"}}
	_ = (AnswerStats{}).Apply(context.Background(), c)
	if c.Answer != nil {
		t.Fatalf("expected nil, got %+v", c.Answer)
	}
}
