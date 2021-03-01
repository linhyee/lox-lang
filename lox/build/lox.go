package main

import (
	"lox"
	"os"
)

func main() {
	lox.RunInterpreter(os.Args[1:])
}
