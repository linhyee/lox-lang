package lox

import (
	"fmt"
	"testing"
)

func TestAstPrinter(t *testing.T) {
	expression := NewBinary(
		NewUnary(
			NewToken(MINUS, "-", nil, 1),
			NewLiteral(123), false),
		NewToken(STAR, "*", nil, 1),
		NewGrouping(NewLiteral(45.67)))

	fmt.Println((&AstPrinter{}).printExpr(expression))
}
