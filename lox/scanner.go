package lox

import (
	"fmt"
	"strconv"
	"strings"
)

var keywords = map[string]TokenType{
	"and":      AND,
	"class":    CLASS,
	"else":     ELSE,
	"false":    FALSE,
	"for":      FOR,
	"fun":      FUN,
	"if":       IF,
	"nil":      NIL,
	"or":       OR,
	"print":    PRINT,
	"return":   RETURN,
	"super":    SUPER,
	"this":     THIS,
	"true":     TRUE,
	"var":      VAR,
	"while":    WHILE,
	"break":    BREAK,
	"continue": CONTINUE,
}

type Scanner struct {
	source  *strings.Reader
	tokens  []*Token
	start   int
	current int
	line    int
}

// NewScanner return an pointer that points to scanner object
func NewScanner(source string) *Scanner {
	return &Scanner{source: strings.NewReader(source), line: 1}
}

// ScanTokens adding tokens until it runs out of characters
func (this *Scanner) ScanTokens() []*Token {
	for !this.isAtEnd() {
		// we are at the beginning of the next lexeme.
		this.start = this.current
		this.scanToken()
	}
	this.tokens = append(this.tokens, NewToken(EOF, "", nil, this.line))
	return this.tokens
}

func (this *Scanner) isAtEnd() bool {
	return int64(this.current) >= this.source.Size()
}

func (this *Scanner) scanToken() {
	c := this.advance()

	ifExp := func(e bool, a, b TokenType) TokenType {
		if e {
			return a
		}
		return b
	}
	switch c {
	case '(':
		this.addToken(LEFT_PAREN)
	case ')':
		this.addToken(RIGHT_PAREN)
	case '{':
		this.addToken(LEFT_BRACE)
	case '}':
		this.addToken(RIGHT_BRACE)
	case '[':
		this.addToken(LEFT_BRACKET)
	case ']':
		this.addToken(RIGHT_BRACKET)
	case ',':
		this.addToken(COMMA)
	case '.':
		this.addToken(DOT)
	case '-':
		this.addToken(ifExp(this.match('-'), MINUS_MINUS, MINUS))
	case '+':
		this.addToken(ifExp(this.match('+'), PLUS_PLUS, PLUS))
	case ';':
		this.addToken(SEMICOLON)
	case ':':
		this.addToken(COLON)
	case '?':
		this.addToken(QUESTION_MARK)
	case '*':
		this.addToken(STAR)
	case '!':
		this.addToken(ifExp(this.match('='), BANG_EQUAL, BANG))
	case '=':
		this.addToken(ifExp(this.match('='), EQUAL_EQUAL, EQUAL))
	case '<':
		this.addToken(ifExp(this.match('='), LESS_EQUAL, LESS))
	case '>':
		this.addToken(ifExp(this.match('='), GREATER_EQUAL, GREATER))
	case '/':
		if this.match('/') {
			for this.peek() != '\n' && !this.isAtEnd() {
				this.advance()
			}
		} else {
			this.addToken(SLASH)
		}
	case ' ', '\r', '\t':
	case '\n':
		this.line++
	case '"':
		this.strings()
	default:
		if this.isDigit(c) {
			this.number()
		} else if this.isAlpha(c) {
			this.identifier()
		} else {
			errorLine(this.line, "unexpected character.")
			fmt.Println(c, string(c))
		}
	}
}

func (this *Scanner) advance() rune {
	_, _ = this.source.Seek(int64(this.current), 0)
	ch, size, err := this.source.ReadRune()
	if err != nil {
		errorLine(this.line, err.Error())
	}
	this.current += int(size)
	return ch
}

func (this *Scanner) addToken(typ TokenType) {
	this.addTokenWithLiteral(typ, nil)
}

func (this *Scanner) addTokenWithLiteral(typ TokenType, literal interface{}) {
	text := make([]byte, this.current-this.start)
	_, _ = this.source.ReadAt(text, int64(this.start))
	this.tokens = append(this.tokens, NewToken(typ, string(text), literal, this.line))
}

func (this *Scanner) match(expected rune) bool {
	if this.isAtEnd() {
		return false
	}
	b := make([]byte, 1)
	_, _ = this.source.ReadAt(b, int64(this.current))
	if rune(b[0]) != expected {
		return false
	}
	this.current++
	return true
}

func (this *Scanner) peek() rune {
	if this.isAtEnd() {
		return '\x00'
	}
	b := make([]byte, 1)
	_, _ = this.source.ReadAt(b, int64(this.current))
	return rune(b[0])
}

func (this *Scanner) peekNext() rune {
	if int64(this.current)+1 >= this.source.Size() {
		return '\x00'
	}
	b := make([]byte, 1)
	_, _ = this.source.ReadAt(b, int64(this.current))
	return rune(b[0])
}

func (this *Scanner) strings() {
	for this.peek() != '"' && !this.isAtEnd() {
		ch := this.peek()
		if ch == '\n' {
			this.line++
		}
		//处理转义字符
		if ch == '\\' {
			ch2 := this.peekNext()
			//TODO
			if ch2 == '"' {
			} else if ch2 == '\\' {
			} else if ch2 == 'b' {
			} else if ch2 == 'r' {
			} else if ch2 == 'n' {
			} else if ch2 == 't' {
			}
		}

		this.advance()
	}
	if this.isAtEnd() {
		errorLine(this.line, "unterminated string.")
		return
	}
	// The closing ".
	this.advance()

	// Trim the surrounding quotes.
	value := make([]byte, this.current-1-this.start-1)
	_, _ = this.source.ReadAt(value, int64(this.start)+1)
	this.addTokenWithLiteral(STRING, string(value))
}

func (this *Scanner) isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func (this *Scanner) number() {
	for this.isDigit(this.peek()) {
		this.advance()
	}

	// look for a fractional part
	if this.peek() == '.' && this.isDigit(this.peekNext()) {
		// consume the "."
		this.advance()

		for this.isDigit(this.peek()) {
			this.advance()
		}
	}

	b := make([]byte, this.current-this.start)
	_, _ = this.source.ReadAt(b, int64(this.start))

	value, _ := strconv.ParseFloat(string(b), 0)
	this.addTokenWithLiteral(NUMBER, value)
}

func (this *Scanner) isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func (this *Scanner) isAlphaNumeric(c rune) bool {
	return this.isAlpha(c) || this.isDigit(c)
}

func (this *Scanner) identifier() {
	for this.isAlphaNumeric(this.peek()) {
		this.advance()
	}
	text := make([]byte, this.current-this.start)
	_, _ = this.source.ReadAt(text, int64(this.start))
	typ, ok := keywords[string(text)]
	if !ok {
		typ = IDENTIFIER
	}
	this.addToken(typ)
}
