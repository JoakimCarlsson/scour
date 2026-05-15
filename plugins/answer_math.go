package plugins

import (
	"context"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"strings"
)

type AnswerMath struct{}

func (AnswerMath) Name() string { return "answer_math" }

func (AnswerMath) Apply(_ context.Context, c *Context) error {
	terms := strings.TrimSpace(c.Query.Terms)
	if terms == "" {
		return nil
	}
	if !isMathOnly(terms) {
		return nil
	}
	expr, err := parser.ParseExpr(terms)
	if err != nil {
		return nil
	}
	val, ok := evalConstExpr(expr)
	if !ok || val.Kind() == constant.Unknown {
		return nil
	}
	c.Answer = &Answer{Text: val.ExactString(), Source: "answer_math"}
	return nil
}

func isMathOnly(s string) bool {
	hasDigit := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '+' || r == '-' || r == '*' || r == '/' || r == '%' || r == '(' || r == ')' || r == '.' || r == ' ':
		default:
			return false
		}
	}
	return hasDigit
}

func evalConstExpr(expr ast.Expr) (constant.Value, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.INT && e.Kind != token.FLOAT {
			return constant.MakeUnknown(), false
		}
		return constant.MakeFromLiteral(e.Value, e.Kind, 0), true
	case *ast.ParenExpr:
		return evalConstExpr(e.X)
	case *ast.UnaryExpr:
		x, ok := evalConstExpr(e.X)
		if !ok {
			return constant.MakeUnknown(), false
		}
		return constant.UnaryOp(e.Op, x, 0), true
	case *ast.BinaryExpr:
		x, ok := evalConstExpr(e.X)
		if !ok {
			return constant.MakeUnknown(), false
		}
		y, ok := evalConstExpr(e.Y)
		if !ok {
			return constant.MakeUnknown(), false
		}
		if (e.Op == token.QUO || e.Op == token.REM) && constant.Sign(y) == 0 {
			return constant.MakeUnknown(), false
		}
		return constant.BinaryOp(x, e.Op, y), true
	}
	return constant.MakeUnknown(), false
}
