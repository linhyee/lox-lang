package glox

import "fmt"

type TokenType uint8

const (
	TokenLeftParen TokenType = iota
	TokenRightParen
	TokenLeftBrace
	TokenRightBrace
	TokenLeftBracket
	TokenRightBracket
	TokenComma
	TokenDot
	TokenMinus
	TokenPlus
	TokenSemicolon
	TokenSlash
	TokenStar
	TokenBang
	TokenBangEqual
	TokenEqual
	TokenEqualEqual
	TokenGreater
	TokenGreaterEqual
	TokenLess
	TokenLessEqual
	TokenPlusPlus
	TokenMinusMinus
	TokenIdentifier
	TokenString
	TokenNumber
	TokenQuestion
	TokenColon
	TokenAnd
	TokenClass
	TokenConst
	TokenElse
	TokenExport
	TokenFalse
	TokenFun
	TokenFor
	TokenIf
	TokenNil
	TokenOr
	TokenPrint
	TokenReturn
	TokenSuper
	TokenThis
	TokenTrue
	TokenVar
	TokenWhile
	TokenBreak
	TokenContinue
	TokenError
	TokenEOF
)

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal Value
	Line    int
}

func (t Token) String() string {
	return fmt.Sprintf("%v %q %v", t.Type, t.Lexeme, t.Literal)
}
