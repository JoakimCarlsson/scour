package plugins

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type AnswerUnits struct{}

func (AnswerUnits) Name() string { return "answer_units" }

type unitCategory string

const (
	catLength unitCategory = "length"
	catMass   unitCategory = "mass"
	catTemp   unitCategory = "temperature"
	catData   unitCategory = "data"
)

type unit struct {
	cat   unitCategory
	scale float64
	abbr  string
}

var unitTable = map[string]unit{
	"mm":          {catLength, 0.001, "mm"},
	"millimeter":  {catLength, 0.001, "mm"},
	"millimeters": {catLength, 0.001, "mm"},
	"cm":          {catLength, 0.01, "cm"},
	"centimeter":  {catLength, 0.01, "cm"},
	"centimeters": {catLength, 0.01, "cm"},
	"m":           {catLength, 1, "m"},
	"meter":       {catLength, 1, "m"},
	"meters":      {catLength, 1, "m"},
	"km":          {catLength, 1000, "km"},
	"kilometer":   {catLength, 1000, "km"},
	"kilometers":  {catLength, 1000, "km"},
	"in":          {catLength, 0.0254, "in"},
	"inch":        {catLength, 0.0254, "in"},
	"inches":      {catLength, 0.0254, "in"},
	"ft":          {catLength, 0.3048, "ft"},
	"foot":        {catLength, 0.3048, "ft"},
	"feet":        {catLength, 0.3048, "ft"},
	"yd":          {catLength, 0.9144, "yd"},
	"yard":        {catLength, 0.9144, "yd"},
	"yards":       {catLength, 0.9144, "yd"},
	"mi":          {catLength, 1609.344, "mi"},
	"mile":        {catLength, 1609.344, "mi"},
	"miles":       {catLength, 1609.344, "mi"},

	"mg":        {catMass, 0.001, "mg"},
	"g":         {catMass, 1, "g"},
	"gram":      {catMass, 1, "g"},
	"grams":     {catMass, 1, "g"},
	"kg":        {catMass, 1000, "kg"},
	"kilogram":  {catMass, 1000, "kg"},
	"kilograms": {catMass, 1000, "kg"},
	"oz":        {catMass, 28.3495, "oz"},
	"ounce":     {catMass, 28.3495, "oz"},
	"ounces":    {catMass, 28.3495, "oz"},
	"lb":        {catMass, 453.592, "lb"},
	"lbs":       {catMass, 453.592, "lb"},
	"pound":     {catMass, 453.592, "lb"},
	"pounds":    {catMass, 453.592, "lb"},

	"b":     {catData, 1, "B"},
	"byte":  {catData, 1, "B"},
	"bytes": {catData, 1, "B"},
	"kb":    {catData, 1024, "KB"},
	"kib":   {catData, 1024, "KiB"},
	"mb":    {catData, 1024 * 1024, "MB"},
	"mib":   {catData, 1024 * 1024, "MiB"},
	"gb":    {catData, 1024 * 1024 * 1024, "GB"},
	"gib":   {catData, 1024 * 1024 * 1024, "GiB"},
	"tb":    {catData, 1024 * 1024 * 1024 * 1024, "TB"},

	"c":          {catTemp, 0, "°C"},
	"celsius":    {catTemp, 0, "°C"},
	"f":          {catTemp, 0, "°F"},
	"fahrenheit": {catTemp, 0, "°F"},
	"k":          {catTemp, 0, "K"},
	"kelvin":     {catTemp, 0, "K"},
}

var unitRe = regexp.MustCompile(
	`^(-?\d+(?:\.\d+)?)\s*([a-z]+)\s+(?:in|to)\s+([a-z]+)$`,
)

func (AnswerUnits) Apply(_ context.Context, c *Context) error {
	terms := strings.ToLower(strings.TrimSpace(c.Query.Terms))
	m := unitRe.FindStringSubmatch(terms)
	if m == nil {
		return nil
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil
	}
	from, ok1 := unitTable[m[2]]
	to, ok2 := unitTable[m[3]]
	if !ok1 || !ok2 || from.cat != to.cat {
		return nil
	}
	var result float64
	if from.cat == catTemp {
		var k float64
		switch from.abbr {
		case "°C":
			k = val + 273.15
		case "°F":
			k = (val-32)*5/9 + 273.15
		case "K":
			k = val
		}
		switch to.abbr {
		case "°C":
			result = k - 273.15
		case "°F":
			result = (k-273.15)*9/5 + 32
		case "K":
			result = k
		}
	} else {
		result = val * from.scale / to.scale
	}
	c.Answer = &Answer{
		Text:   fmt.Sprintf("%g %s", roundTo(result, 5), to.abbr),
		Source: "answer_units",
	}
	return nil
}

func roundTo(v float64, places int) float64 {
	scale := 1.0
	for range places {
		scale *= 10
	}
	if v >= 0 {
		return float64(int64(v*scale+0.5)) / scale
	}
	return float64(int64(v*scale-0.5)) / scale
}
