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
	return &Scanner{source: strings.NewReader(source), line: 1, tokens: make([]*Token, 0, 100)}
}

func (this *Scanner) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%-15s\t%-20s\t%-20v\n", "Type", "Lexeme", "Literal")
	for _, t := range this.tokens {
		fmt.Fprintf(&sb, "%-15s\t", tokenNames[t.Type])
		fmt.Fprintf(&sb, "%-20s\t", t.Lexeme)
		fmt.Fprintf(&sb, "%-20v", t.Literal)
		sb.WriteString("\n")
	}
	return sb.String()
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
	// 先用 peek 查看，避免消耗
	ch := this.peek()
	if ch != expected {
		return false
	}
	// 确认匹配，消耗掉这个 rune
	this.advance()
	return true
}

func (this *Scanner) peek() rune {
	if this.isAtEnd() {
		return '\x00'
	}
	// 保存当前位置
	pos, _ := this.source.Seek(0, 1)
	ch, _, _ := this.source.ReadRune()
	// 恢复位置
	this.source.Seek(pos, 0)
	return ch
}

func (this *Scanner) peekNext() rune {
	if int64(this.current)+1 >= this.source.Size() {
		return '\x00'
	}
	// 保存当前位置
	pos, _ := this.source.Seek(0, 1)
	// 先读取当前 rune，再读取下一个
	_, size, _ := this.source.ReadRune()
	if int64(this.current+size) >= this.source.Size() {
		this.source.Seek(pos, 0)
		return 0
	}
	nextCh, _, _ := this.source.ReadRune()
	// 恢复位置
	this.source.Seek(pos, 0)
	return nextCh
}

func (this *Scanner) strings() {
	var sb strings.Builder

	for !this.isAtEnd() {
		ch := this.peek()

		if ch == '"' {
			this.advance()
			this.addTokenWithLiteral(STRING, sb.String())
			return
		}

		if ch == '\n' {
			this.line++
		}

		// 处理转义字符
		if ch == '\\' {
			this.advance()
			if this.isAtEnd() {
				errorLine(this.line, "unterminated string escape")
				return
			}

			esc := this.advance()
			switch esc {
			case '"':
				sb.WriteRune('"')
			case '\\':
				sb.WriteRune('\\')
			case 'b':
				sb.WriteRune('\b')
			case 'f':
				sb.WriteRune('\f')
			case 'n':
				sb.WriteRune('\n')
			case 'r':
				sb.WriteRune('\r')
			case 't':
				sb.WriteRune('\t')
			case 'v':
				sb.WriteRune('\v')
			default:
				// 未知转义，保留原字符
				sb.WriteRune(esc)
			}
			continue
		}

		// 普通字符
		this.advance()
		sb.WriteRune(ch)
	}

	errorLine(this.line, "unterminated string")
}

func (this *Scanner) isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func (this *Scanner) number() {
	for this.isDigit(this.peek()) {
		this.advance()
	}

	// look for a fractional part
	isF := false
	if this.peek() == '.' && this.isDigit(this.peekNext()) {
		isF = true
		// consume the "."
		this.advance()

		for this.isDigit(this.peek()) {
			this.advance()
		}
	}

	if this.peek() == 'e' || this.peek() == 'E' {
		isF = true
		this.advance()
		if this.peek() == '+' || this.peek() == '-' {
			this.advance()
		}
		if !this.isDigit(this.peek()) {
			errorLine(this.line, "invalid number format")
			return
		}
		for this.isDigit(this.peek()) {
			this.advance()
		}
	}

	b := make([]byte, this.current-this.start)
	_, _ = this.source.ReadAt(b, int64(this.start))

	isF = true
	if isF {
		value, _ := strconv.ParseFloat(string(b), 0)
		this.addTokenWithLiteral(NUMBER, value)
	} else {
		value, _ := strconv.ParseInt(string(b), 10, 64)
		this.addTokenWithLiteral(NUMBER, value)
	}
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
