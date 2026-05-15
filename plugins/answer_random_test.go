package plugins

import (
	"context"
	"strconv"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

func TestAnswerRandomRange(t *testing.T) {
	a := AnswerRandom{Rand: func(n int) int { return 0 }}
	c := &Context{Query: query.Query{Terms: "random 1-10"}}
	if err := a.Apply(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.Answer == nil || c.Answer.Text != "1" {
		t.Fatalf("got %+v", c.Answer)
	}
}

func TestAnswerRandomPick(t *testing.T) {
	a := AnswerRandom{Rand: func(n int) int { return 1 }}
	c := &Context{Query: query.Query{Terms: "random pick a b c"}}
	if err := a.Apply(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.Answer == nil || c.Answer.Text != "b" {
		t.Fatalf("got %+v", c.Answer)
	}
}

func TestAnswerRandomCoin(t *testing.T) {
	a := AnswerRandom{Rand: func(n int) int { return 0 }}
	c := &Context{Query: query.Query{Terms: "flip coin"}}
	_ = a.Apply(context.Background(), c)
	if c.Answer == nil || c.Answer.Text != "Heads" {
		t.Fatalf("got %+v", c.Answer)
	}
}

func TestAnswerRandomRoll(t *testing.T) {
	a := AnswerRandom{Rand: func(n int) int { return 5 }}
	c := &Context{Query: query.Query{Terms: "roll d20"}}
	_ = a.Apply(context.Background(), c)
	if c.Answer == nil {
		t.Fatal("no answer")
	}
	n, err := strconv.Atoi(c.Answer.Text)
	if err != nil || n != 6 {
		t.Fatalf("got %q", c.Answer.Text)
	}
}

func TestAnswerRandomNoMatch(t *testing.T) {
	a := AnswerRandom{}
	c := &Context{Query: query.Query{Terms: "weather in london"}}
	_ = a.Apply(context.Background(), c)
	if c.Answer != nil {
		t.Fatalf("expected nil answer, got %+v", c.Answer)
	}
}
