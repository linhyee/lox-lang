package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: generate_ast <output directory>")
		os.Exit(64)
	}
	outputDir := os.Args[1]
	defineAst(outputDir, "Expr", []string{
		"Assign       : name *Token, value Expr",
		"Binary       : left Expr, operator *Token, right Expr",
		"Lambda       : params []*Token, body []Stmt",
		"Call         : callee Expr, paren *Token, arguments []Expr",
		"Get          : object Expr, name *Token",
		"Index        : left Expr, name *Token, index Expr",
		"ArrayLiteral : bracket *Token, items []Expr",
		"Grouping     : expression Expr",
		"Literal      : value interface{}",
		"Logical      : left Expr, operator *Token, right Expr",
		"Set          : object Expr, name *Token, value Expr",
		"Super        : keyword *Token, method *Token",
		"This         : keyword *Token",
		"ArraySet     : left Expr, name *Token, index Expr, value Expr",
		"Unary        : operator *Token, right Expr, postfix bool",
		"Ternary      : expr Expr, thenBranch Expr, elseBranch Expr",
		"Variable     : name *Token",
	})
	defineAst(outputDir, "Stmt", []string{
		"Block     : statements []Stmt",
		"Class     : name *Token, superclass *Variable, methods []*Function",
		"Expression: expression Expr",
		"Function  : name *Token, params []*Token, body []Stmt",
		"If        : condition Expr, thenBranch Stmt, elseBranch Stmt",
		"Print     : expression Expr",
		"Return    : keyword *Token, value Expr",
		"Var       : name *Token, initializer Expr",
		"While     : condition Expr, body Stmt",
		"Break     : ",
		"Continue  : ",
	})
}

func defineAst(outputDir, baseName string, types []string) {
	path := outputDir + "/" + strings.ToLower(baseName) + ".go"
	writer, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(64)
	}
	defer writer.Close()

	writer.WriteString("package lox\n\n")
	writer.WriteString("type " + baseName + " interface {\n")
	writer.WriteString("\taccept(visitor Visitor" + baseName + ") interface{}\n")
	writer.WriteString("}\n\n")

	defineVisitor(writer, baseName, types)

	for _, ty := range types {
		className := strings.Trim(strings.Split(ty, ":")[0], " ")
		fields := strings.Trim(strings.Split(ty, ":")[1], " ")
		defineType(writer, baseName, className, fields)

		writer.WriteString("func (this *" + className +
			") accept(visitor Visitor" + baseName + ") interface{} {\n")
		writer.WriteString("\treturn visitor.visit" + className + baseName + "(this)\n")
		writer.WriteString("}\n\n")
	}
}

func defineType(writer io.StringWriter, baseName, className, fieldList string) {
	fields := strings.Split(fieldList, ",")

	// 定义构造方法
	writer.WriteString("func New" + className + "(" + fieldList + ") *" + className + " {\n")
	writer.WriteString("\treturn &" + className + "{\n")
	for _, field := range fields {
		if strings.Trim(field, " ") == "" {
			continue
		}
		fieldName := strings.Split(strings.Trim(field, " "), " ")[0]
		fmt.Println(field)
		writer.WriteString("\t\t" + fieldName + " : " + fieldName + ",\n")
	}
	writer.WriteString("\t}\n")
	writer.WriteString("}\n\n")

	// 定义类型
	writer.WriteString("type " + className + " struct {\n")

	// 字段
	for _, field := range fields {
		writer.WriteString("\t" + strings.Trim(field, " ") + "\n")
	}
	writer.WriteString("}\n\n")
}

func defineVisitor(writer io.StringWriter, baseName string, types []string) {
	writer.WriteString("type Visitor" + baseName + " interface {\n")
	for _, ty := range types {
		typeName := strings.Trim(strings.Split(ty, ":")[0], " ")
		writer.WriteString("\tvisit" + typeName + baseName + "(" +
			strings.ToLower(baseName) + " *" + typeName + ") interface{}\n")
	}
	writer.WriteString("}\n\n")
}
