package plugins

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type AnswerStats struct{}

func (AnswerStats) Name() string { return "answer_stats" }

func (AnswerStats) Apply(_ context.Context, c *Context) error {
	terms := strings.TrimSpace(c.Query.Terms)
	if terms == "" {
		return nil
	}
	fields := strings.Fields(terms)
	if len(fields) < 2 {
		return nil
	}
	op := strings.ToLower(fields[0])
	switch op {
	case "mean", "avg", "average", "median", "sum", "min", "max", "stddev":
	default:
		return nil
	}
	nums := make([]float64, 0, len(fields)-1)
	for _, f := range fields[1:] {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil
		}
		nums = append(nums, v)
	}
	if len(nums) == 0 {
		return nil
	}
	var result float64
	switch op {
	case "sum":
		for _, n := range nums {
			result += n
		}
	case "mean", "avg", "average":
		for _, n := range nums {
			result += n
		}
		result /= float64(len(nums))
	case "median":
		s := append([]float64(nil), nums...)
		sort.Float64s(s)
		mid := len(s) / 2
		if len(s)%2 == 0 {
			result = (s[mid-1] + s[mid]) / 2
		} else {
			result = s[mid]
		}
	case "min":
		result = nums[0]
		for _, n := range nums[1:] {
			if n < result {
				result = n
			}
		}
	case "max":
		result = nums[0]
		for _, n := range nums[1:] {
			if n > result {
				result = n
			}
		}
	case "stddev":
		var mean float64
		for _, n := range nums {
			mean += n
		}
		mean /= float64(len(nums))
		var sq float64
		for _, n := range nums {
			d := n - mean
			sq += d * d
		}
		result = math.Sqrt(sq / float64(len(nums)))
	}
	c.Answer = &Answer{
		Text:   fmt.Sprintf("%g", result),
		Source: "answer_stats",
	}
	return nil
}
