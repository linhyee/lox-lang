package lox

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

var hadError bool
var hadRuntimeError = false
var interpreter = NewInterpreter()

func RunInterpreter(args []string) {
	var err error
	if len(args) > 1 {
		fmt.Println("usage: glox [script]")
		os.Exit(64)
	} else if len(args) == 1 {
		err = runFile(args[0])
	} else {
		err = runPrompt()
	}
	if err != nil {
		fmt.Println(err)
	}
}

// runFile give it a path to file,it reads the file and execute it.
func runFile(path string) (err error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	run(string(bytes))
	return
}

// runPrompt shell mod, execute code one line at a time
func runPrompt() (err error) {
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, _, err := r.ReadLine()
		if err != nil {
			return err
		}
		if len(line) == 0 {
			break
		}
		run(string(line))
		hadError = false
	}
	return
}

// run execute code, both the prompt and the file runner are thin
// wrappers around this core
func run(source string) {
	// token scanning
	scanner := NewScanner(source)
	tokens := scanner.ScanTokens()

	// parse
	parser := NewParser(tokens)
	statements := parser.Parse()

	if hadError {
		return
	}

	//resolve, semantic analysis
	resolver := NewResolver(interpreter)
	resolver.resolve(statements)
	if hadError {
		return
	}

	// interpret
	interpreter.Interpret(statements)
	if hadRuntimeError {
		os.Exit(70)
	}
}

// errorLine its report() wrapper function
func errorLine(line int, message string) {
	report(line, "", message)
}

// errorToken Parsing Expressions token-error
func errorToken(token *Token, message string) {
	if token.Type == EOF {
		report(token.Line, " at end", message)
	} else {
		report(token.Line, " at '"+token.Lexeme+"'", message)
	}
}

// report tells the user come syntax error and given line
func report(line int, where string, message string) string {
	message = "[line " + strconv.Itoa(line) + "] Error" + where + ": " + message
	hadError = true

	fmt.Println(message)
	return message
}
