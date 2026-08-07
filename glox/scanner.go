package glox

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

var keywords = map[string]TokenType{
	"and":      TokenAnd,
	"class":    TokenClass,
	"const":    TokenConst,
	"else":     TokenElse,
	"export":   TokenExport,
	"false":    TokenFalse,
	"fun":      TokenFun,
	"for":      TokenFor,
	"if":       TokenIf,
	"nil":      TokenNil,
	"or":       TokenOr,
	"print":    TokenPrint,
	"return":   TokenReturn,
	"super":    TokenSuper,
	"this":     TokenThis,
	"true":     TokenTrue,
	"var":      TokenVar,
	"while":    TokenWhile,
	"break":    TokenBreak,
	"continue": TokenContinue,
}

type Scanner struct {
	source  string
	start   int
	current int
	line    int
}

func NewScanner(source string) *Scanner {
	return &Scanner{source: source, line: 1}
}

func (s *Scanner) ScanToken() Token {
	s.skipWhitespace()
	s.start = s.current
	if s.isAtEnd() {
		return s.makeToken(TokenEOF)
	}

	ch := s.advance()
	if isAlpha(ch) {
		return s.identifier()
	}
	if unicode.IsDigit(ch) {
		return s.number()
	}

	switch ch {
	case '(':
		return s.makeToken(TokenLeftParen)
	case ')':
		return s.makeToken(TokenRightParen)
	case '{':
		return s.makeToken(TokenLeftBrace)
	case '}':
		return s.makeToken(TokenRightBrace)
	case '[':
		return s.makeToken(TokenLeftBracket)
	case ']':
		return s.makeToken(TokenRightBracket)
	case ';':
		return s.makeToken(TokenSemicolon)
	case ',':
		return s.makeToken(TokenComma)
	case '.':
		return s.makeToken(TokenDot)
	case '-':
		if s.match('-') {
			return s.makeToken(TokenMinusMinus)
		}
		return s.makeToken(TokenMinus)
	case '+':
		if s.match('+') {
			return s.makeToken(TokenPlusPlus)
		}
		return s.makeToken(TokenPlus)
	case '/':
		return s.makeToken(TokenSlash)
	case '*':
		return s.makeToken(TokenStar)
	case '?':
		return s.makeToken(TokenQuestion)
	case ':':
		return s.makeToken(TokenColon)
	case '!':
		if s.match('=') {
			return s.makeToken(TokenBangEqual)
		}
		return s.makeToken(TokenBang)
	case '=':
		if s.match('=') {
			return s.makeToken(TokenEqualEqual)
		}
		return s.makeToken(TokenEqual)
	case '<':
		if s.match('=') {
			return s.makeToken(TokenLessEqual)
		}
		return s.makeToken(TokenLess)
	case '>':
		if s.match('=') {
			return s.makeToken(TokenGreaterEqual)
		}
		return s.makeToken(TokenGreater)
	case '"':
		return s.string()
	}

	return s.errorToken("unexpected character")
}

func (s *Scanner) skipWhitespace() {
	for {
		if s.isAtEnd() {
			return
		}
		ch := s.peek()
		switch ch {
		case ' ', '\r', '\t':
			s.advance()
		case '\n':
			s.line++
			s.advance()
		case '/':
			if s.peekNext() == '/' {
				for s.peek() != '\n' && !s.isAtEnd() {
					s.advance()
				}
			} else {
				return
			}
		default:
			return
		}
	}
}

func (s *Scanner) identifier() Token {
	for isAlphaNumeric(s.peek()) {
		s.advance()
	}
	text := s.source[s.start:s.current]
	if typ, ok := keywords[text]; ok {
		return s.makeToken(typ)
	}
	return s.makeToken(TokenIdentifier)
}

func (s *Scanner) number() Token {
	isFloat := false
	for unicode.IsDigit(s.peek()) {
		s.advance()
	}
	if s.peek() == '.' && unicode.IsDigit(s.peekNext()) {
		isFloat = true
		s.advance()
		for unicode.IsDigit(s.peek()) {
			s.advance()
		}
	}
	if s.peek() == 'e' || s.peek() == 'E' {
		if unicode.IsDigit(s.peekExponentDigit()) {
			isFloat = true
			s.advance()
			if s.peek() == '+' || s.peek() == '-' {
				s.advance()
			}
			for unicode.IsDigit(s.peek()) {
				s.advance()
			}
		}
	}
	token := s.makeToken(TokenNumber)
	if isFloat {
		value, _ := strconv.ParseFloat(token.Lexeme, 64)
		token.Literal = value
		return token
	}
	value, err := strconv.ParseInt(token.Lexeme, 10, 64)
	if err != nil {
		floatValue, _ := strconv.ParseFloat(token.Lexeme, 64)
		token.Literal = floatValue
		return token
	}
	token.Literal = value
	return token
}

func (s *Scanner) string() Token {
	out := make([]rune, 0)
	for !s.isAtEnd() {
		ch := s.advance()
		if ch == '"' {
			token := s.makeToken(TokenString)
			token.Literal = string(out)
			return token
		}
		if ch == '\n' {
			s.line++
		}
		if ch == '\\' && !s.isAtEnd() {
			switch esc := s.advance(); esc {
			case '"':
				out = append(out, '"')
			case '\\':
				out = append(out, '\\')
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'b':
				out = append(out, '\b')
			default:
				out = append(out, esc)
			}
			continue
		}
		out = append(out, ch)
	}
	return s.errorToken("unterminated string")
}

func (s *Scanner) makeToken(typ TokenType) Token {
	return Token{Type: typ, Lexeme: s.source[s.start:s.current], Line: s.line}
}

func (s *Scanner) errorToken(message string) Token {
	return Token{Type: TokenError, Lexeme: message, Line: s.line}
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

func (s *Scanner) advance() rune {
	ch, size := utf8.DecodeRuneInString(s.source[s.current:])
	s.current += size
	return ch
}

func (s *Scanner) match(expected rune) bool {
	if s.isAtEnd() || s.peek() != expected {
		return false
	}
	s.advance()
	return true
}

func (s *Scanner) peek() rune {
	if s.isAtEnd() {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(s.source[s.current:])
	return ch
}

func (s *Scanner) peekNext() rune {
	if s.isAtEnd() {
		return 0
	}
	_, size := utf8.DecodeRuneInString(s.source[s.current:])
	next := s.current + size
	if next >= len(s.source) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(s.source[next:])
	return ch
}

func (s *Scanner) peekExponentDigit() rune {
	if s.isAtEnd() {
		return 0
	}
	_, size := utf8.DecodeRuneInString(s.source[s.current:])
	next := s.current + size
	if next >= len(s.source) {
		return 0
	}
	ch, size := utf8.DecodeRuneInString(s.source[next:])
	if ch == '+' || ch == '-' {
		next += size
		if next >= len(s.source) {
			return 0
		}
		ch, _ = utf8.DecodeRuneInString(s.source[next:])
	}
	return ch
}

func isAlpha(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isAlphaNumeric(ch rune) bool {
	return isAlpha(ch) || unicode.IsDigit(ch)
}
