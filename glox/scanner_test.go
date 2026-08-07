package glox

import "testing"

func scanAll(source string) []Token {
	scanner := NewScanner(source)
	var tokens []Token
	for {
		token := scanner.ScanToken()
		tokens = append(tokens, token)
		if token.Type == TokenEOF || token.Type == TokenError {
			return tokens
		}
	}
}

func TestScannerTokensAndLiterals(t *testing.T) {
	source := `(){}[],.-+;/*?! != = == > >= < <= ++ --
and class const else export false fun for if nil or print return super this true var while break continue
identifier _name abc123 123 45.67 1e3 1.5e-2 "a\n\"b"`
	tokens := scanAll(source)
	want := []TokenType{
		TokenLeftParen, TokenRightParen, TokenLeftBrace, TokenRightBrace,
		TokenLeftBracket, TokenRightBracket, TokenComma, TokenDot, TokenMinus,
		TokenPlus, TokenSemicolon, TokenSlash, TokenStar, TokenQuestion,
		TokenBang, TokenBangEqual, TokenEqual, TokenEqualEqual, TokenGreater,
		TokenGreaterEqual, TokenLess, TokenLessEqual, TokenPlusPlus, TokenMinusMinus,
		TokenAnd, TokenClass, TokenConst, TokenElse, TokenExport, TokenFalse,
		TokenFun, TokenFor, TokenIf, TokenNil, TokenOr, TokenPrint, TokenReturn,
		TokenSuper, TokenThis, TokenTrue, TokenVar, TokenWhile, TokenBreak, TokenContinue,
		TokenIdentifier, TokenIdentifier, TokenIdentifier, TokenNumber, TokenNumber,
		TokenNumber, TokenNumber, TokenString, TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("token count mismatch: want %d got %d: %#v", len(want), len(tokens), tokens)
	}
	for i, typ := range want {
		if tokens[i].Type != typ {
			t.Fatalf("token %d type mismatch: want %v got %v (%q)", i, typ, tokens[i].Type, tokens[i].Lexeme)
		}
	}
	if tokens[47].Literal != int64(123) {
		t.Fatalf("number literal mismatch: %#v", tokens[47].Literal)
	}
	if tokens[48].Literal != float64(45.67) {
		t.Fatalf("decimal literal mismatch: %#v", tokens[48].Literal)
	}
	if tokens[49].Literal != float64(1000) {
		t.Fatalf("scientific literal mismatch: %#v", tokens[49].Literal)
	}
	if tokens[50].Literal != float64(0.015) {
		t.Fatalf("scientific decimal literal mismatch: %#v", tokens[50].Literal)
	}
	if tokens[51].Literal != "a\n\"b" {
		t.Fatalf("string literal mismatch: %#v", tokens[51].Literal)
	}
}

func TestScannerSkipsCommentsAndTracksLines(t *testing.T) {
	tokens := scanAll("var a = 1; // comment\nprint a;")
	want := []struct {
		typ  TokenType
		line int
	}{
		{TokenVar, 1},
		{TokenIdentifier, 1},
		{TokenEqual, 1},
		{TokenNumber, 1},
		{TokenSemicolon, 1},
		{TokenPrint, 2},
		{TokenIdentifier, 2},
		{TokenSemicolon, 2},
		{TokenEOF, 2},
	}
	if len(tokens) != len(want) {
		t.Fatalf("token count mismatch: want %d got %d", len(want), len(tokens))
	}
	for i, item := range want {
		if tokens[i].Type != item.typ || tokens[i].Line != item.line {
			t.Fatalf("token %d mismatch: want (%v,line %d), got (%v,line %d)", i, item.typ, item.line, tokens[i].Type, tokens[i].Line)
		}
	}
}

func TestScannerErrors(t *testing.T) {
	if token := scanAll("@")[0]; token.Type != TokenError || token.Lexeme != "unexpected character" {
		t.Fatalf("unexpected character token mismatch: %#v", token)
	}
	if token := scanAll(`"unterminated`)[0]; token.Type != TokenError || token.Lexeme != "unterminated string" {
		t.Fatalf("unterminated string token mismatch: %#v", token)
	}
}
