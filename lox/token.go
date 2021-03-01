package lox

import (
	"fmt"
	"strconv"
)

const (
	// single-character token
	LEFT_PAREN    TokenType = iota // (
	RIGHT_PAREN                    // )
	LEFT_BRACE                     // {
	RIGHT_BRACE                    // }
	LEFT_BRACKET                   // [
	RIGHT_BRACKET                  // ]
	COMMA                          // ,
	DOT                            // .
	MINUS                          // -
	PLUS                           // +
	SEMICOLON                      // ;
	SLASH                          // /
	STAR                           // *

	// one or tow character tokens
	BANG          // !
	BANG_EQUAL    // !=
	EQUAL         // =
	EQUAL_EQUAL   // ==
	GREATER       // >
	GREATER_EQUAL // >=
	LESS          // <
	LESS_EQUAL    //<=
	PLUS_PLUS     // ++
	MINUS_MINUS   // --

	// literals
	IDENTIFIER // abc
	STRING     // "abc"
	NUMBER     // 123

	// conditional
	QUESTION_MARK // ?
	COLON         // :

	// keywords
	AND      // and
	CLASS    // Class
	ELSE     // else
	FALSE    // false
	FUN      // fun
	FOR      // for
	IF       // if
	NIL      // nil
	OR       // or
	PRINT    // print
	RETURN   // return
	SUPER    // super
	THIS     // this
	TRUE     // true
	VAR      // var
	WHILE    // while
	BREAK    // break
	CONTINUE // continue

	EOF
)

// TokenType  which kind of lexeme it represents
type TokenType int

// String stringer
func (tt TokenType) String() string {
	return strconv.Itoa(int(tt))
}

// Token structure to be useful for all of the later phases of the interpreter
type Token struct {
	Type    TokenType
	Lexeme  string
	Literal interface{}
	Line    int
}

// NewToken new a Token type object
func NewToken(typ TokenType, lexeme string, literal interface{}, line int) *Token {
	return &Token{
		Type:    typ,
		Lexeme:  lexeme,
		Literal: literal,
		Line:    line,
	}
}

// String stringer
func (t Token) String() string {
	return fmt.Sprintf("%v %v %v", t.Type, t.Lexeme, t.Literal)
}
