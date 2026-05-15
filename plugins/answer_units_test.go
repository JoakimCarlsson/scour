package plugins

import (
	"context"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestAnswerUnitsLength(t *testing.T) {
	c := &Context{Query: query.Query{Terms: "5 km in miles"}}
	if err := (AnswerUnits{}).Apply(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.Answer == nil || c.Answer.Text != "3.10686 mi" {
		t.Fatalf("got %+v", c.Answer)
	}
}

func TestAnswerUnitsTemp(t *testing.T) {
	c := &Context{Query: query.Query{Terms: "100 f in c"}}
	_ = (AnswerUnits{}).Apply(context.Background(), c)
	if c.Answer == nil {
		t.Fatal("nil answer")
	}
	if c.Answer.Text != "37.77778 °C" {
		t.Fatalf("got %q", c.Answer.Text)
	}
}

func TestAnswerUnitsData(t *testing.T) {
	c := &Context{Query: query.Query{Terms: "3 gb in mb"}}
	_ = (AnswerUnits{}).Apply(context.Background(), c)
	if c.Answer == nil || c.Answer.Text != "3072 MB" {
		t.Fatalf("got %+v", c.Answer)
	}
}

func TestAnswerUnitsMismatch(t *testing.T) {
	c := &Context{Query: query.Query{Terms: "5 km in kg"}}
	_ = (AnswerUnits{}).Apply(context.Background(), c)
	if c.Answer != nil {
		t.Fatalf("expected nil, got %+v", c.Answer)
	}
}

func TestAnswerUnitsNoMatch(t *testing.T) {
	c := &Context{Query: query.Query{Terms: "weather in london"}}
	_ = (AnswerUnits{}).Apply(context.Background(), c)
	if c.Answer != nil {
		t.Fatalf("expected nil, got %+v", c.Answer)
	}
}
