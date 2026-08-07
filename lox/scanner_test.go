package lox

import (
	"fmt"
	"testing"
)

func TestFloatLiteral(t *testing.T) {
	scanner := NewScanner("print 3.14")
	scanner.ScanTokens()
	fmt.Println(scanner)
}
