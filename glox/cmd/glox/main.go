package main

import (
	"bufio"
	"fmt"
	"os"

	"glox"
)

func main() {
	if len(os.Args) >= 2 {
		vm := glox.NewVM(glox.Options{Args: os.Args[2:]})
		if err := vm.RunFile(os.Args[1]); err != nil {
			if !isReported(err) {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(70)
		}
		return
	}

	vm := glox.NewVM(glox.Options{})
	reader := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !reader.Scan() {
			break
		}
		if err := vm.RunString(reader.Text()); err != nil {
			if !isReported(err) {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}

func isReported(err error) bool {
	switch err.(type) {
	case *glox.DiagnosticError, *glox.RuntimeError:
		return true
	default:
		return false
	}
}
