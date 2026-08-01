package glox

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type InterpretResult int

const (
	InterpretOK InterpretResult = iota
	InterpretCompileError
	InterpretRuntimeError
)

type DiagnosticError struct {
	Messages []string
}

func (e *DiagnosticError) Error() string {
	return strings.Join(e.Messages, "\n")
}

type RuntimeError struct {
	Token   *Token
	Message string
}

func (e *RuntimeError) Error() string {
	if e == nil {
		return ""
	}
	if e.Token == nil {
		return e.Message
	}
	return "[line " + strconv.Itoa(e.Token.Line) + "] RuntimeError: " + e.Message
}

type Diagnostics struct {
	writer   io.Writer
	messages []string
}

func NewDiagnostics(writer io.Writer) *Diagnostics {
	if writer == nil {
		writer = io.Discard
	}
	return &Diagnostics{writer: writer}
}

func (d *Diagnostics) HadError() bool {
	return d != nil && len(d.messages) > 0
}

func (d *Diagnostics) Reset() {
	if d == nil {
		return
	}
	d.messages = nil
}

func (d *Diagnostics) Err() error {
	if !d.HadError() {
		return nil
	}
	return &DiagnosticError{Messages: append([]string(nil), d.messages...)}
}

func (d *Diagnostics) ErrorAt(token Token, message string) {
	if token.Type == TokenEOF {
		d.report(token.Line, " at end", message)
		return
	}
	if token.Type == TokenError {
		d.report(token.Line, "", message)
		return
	}
	d.report(token.Line, " at '"+token.Lexeme+"'", message)
}

func (d *Diagnostics) Error(line int, message string) {
	d.report(line, "", message)
}

func (d *Diagnostics) Runtime(message string) {
	if d == nil {
		return
	}
	d.messages = append(d.messages, message)
	fmt.Fprintln(d.writer, message)
}

func (d *Diagnostics) report(line int, where string, message string) {
	text := "[line " + strconv.Itoa(line) + "] Error" + where + ": " + message
	if d == nil {
		return
	}
	d.messages = append(d.messages, text)
	fmt.Fprintln(d.writer, text)
}
