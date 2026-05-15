package plugins

import (
	"context"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
)

type AnswerRandom struct {
	// Rand is optional; if nil a fresh PCG is used per call.
	Rand func(n int) int
}

func (AnswerRandom) Name() string { return "answer_random" }

var randRangeRe = regexp.MustCompile(`^random\s+(-?\d+)\s*-\s*(-?\d+)$`)
var randPickRe = regexp.MustCompile(`^random\s+pick\s+(.+)$`)
var rollRe = regexp.MustCompile(`^roll\s+d(\d+)$`)

func (a AnswerRandom) Apply(_ context.Context, c *Context) error {
	terms := strings.ToLower(strings.TrimSpace(c.Query.Terms))
	if terms == "" {
		return nil
	}
	intn := a.Rand
	if intn == nil {
		intn = func(n int) int {
			if n <= 0 {
				return 0
			}
			return rand.IntN(n)
		}
	}
	if terms == "flip coin" || terms == "coin flip" {
		opts := []string{"Heads", "Tails"}
		c.Answer = &Answer{Text: opts[intn(2)], Source: "answer_random"}
		return nil
	}
	if m := rollRe.FindStringSubmatch(terms); m != nil {
		sides, _ := strconv.Atoi(m[1])
		if sides > 0 {
			c.Answer = &Answer{
				Text:   fmt.Sprintf("%d", intn(sides)+1),
				Source: "answer_random",
			}
			return nil
		}
	}
	if m := randRangeRe.FindStringSubmatch(terms); m != nil {
		lo, _ := strconv.Atoi(m[1])
		hi, _ := strconv.Atoi(m[2])
		if hi < lo {
			lo, hi = hi, lo
		}
		c.Answer = &Answer{
			Text:   fmt.Sprintf("%d", lo+intn(hi-lo+1)),
			Source: "answer_random",
		}
		return nil
	}
	if m := randPickRe.FindStringSubmatch(terms); m != nil {
		fields := strings.Fields(m[1])
		if len(fields) > 0 {
			c.Answer = &Answer{Text: fields[intn(len(fields))], Source: "answer_random"}
			return nil
		}
	}
	return nil
}
