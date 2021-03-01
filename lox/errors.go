package lox

import (
	"strconv"
)

type RuntimeError interface {
	GetToken() *Token
	GetMessage() string
}

// runtimeError 运行时错误类型
type runtimeError struct {
	token   *Token
	message string
}

func NewRuntimeError(token *Token, message string) *runtimeError {
	return &runtimeError{token: token, message: message}
}

func (re *runtimeError) GetToken() *Token {
	return re.token
}

func (re *runtimeError) GetMessage() string {
	return re.message
}

func (re runtimeError) Error() string {
	return re.token.String() + " " + re.message
}

// parserError 语法解析错误类型
type parserError struct {
	token   *Token
	message string
}

func NewParseError(token *Token, message string) parserError {
	return parserError{token: token, message: message}
}

func (pe parserError) Error() string {
	if pe.token.Type == EOF {
		return report(pe.token.Line, " at end", pe.message)
	} else {
		return report(pe.token.Line, " at '"+pe.token.Lexeme+"'", pe.message)
	}
}

func NewIllegalIndexError(index int, message string) *illegalIndexError {
	return &illegalIndexError{index: index, runtimeError: runtimeError{message: message}}
}

type illegalIndexError struct {
	index int
	runtimeError
}

func (iie illegalIndexError) Error() string {
	return "out of bound:" + strconv.Itoa(iie.index) + ", " + iie.message
}

func NewStackError(top int, message string) *stackError {
	return &stackError{top: top, message: message}
}

type stackError struct {
	top     int
	message string
}

func (se stackError) Error() string {
	return "out of bound:" + strconv.Itoa(se.top) + ", " + se.message
}
