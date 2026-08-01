package lox

import (
	"bytes"
	"fmt"
)

type AstPrinter struct{}

func (printer *AstPrinter) printExpr(expr Expr) string {
	v, _ := expr.accept(printer)
	return v.(string)
}

func (printer *AstPrinter) printStmt(stmt Stmt) string {
	v, _ := stmt.accept(printer)
	return v.(string)
}

func (printer *AstPrinter) visitBlockStmt(stmt *Block) (interface{}, error) {
	bs := bytes.NewBufferString("")
	bs.WriteString("block ")
	for _, statement := range stmt.statements {
		v, _ := statement.accept(printer)
		bs.WriteString(v.(string))
	}
	bs.WriteString(")")
	return bs.String(), nil
}

func (printer *AstPrinter) visitClassStmt(stmt *Class) (interface{}, error) {
	bs := bytes.NewBufferString("")
	bs.WriteString("class " + stmt.name.Lexeme)
	if stmt.superclass != nil {
		bs.WriteString(" < " + printer.printExpr(stmt.superclass))
	}
	for _, method := range stmt.methods {
		bs.WriteString(" " + printer.printStmt(method))
	}
	bs.WriteString(")")
	return bs.String(), nil
}

func (printer *AstPrinter) visitExpressionStmt(stmt *Expression) (interface{}, error) {
	return printer.parenthesize(";", stmt.expression)
}

func (printer *AstPrinter) visitFunctionStmt(stmt *Function) (interface{}, error) {
	bs := bytes.NewBufferString("")
	bs.WriteString("fun " + stmt.name.Lexeme + " (")
	for _, param := range stmt.params {
		if param != stmt.params[0] {
			bs.WriteString(" ")
		}
		bs.WriteString(param.Lexeme)
	}
	bs.WriteString(") ")
	for _, body := range stmt.body {
		v, _ := body.accept(printer)
		bs.WriteString(v.(string))
	}
	bs.WriteString(")")
	return bs.String(), nil
}

func (printer *AstPrinter) visitIfStmt(stmt *If) (interface{}, error) {
	if stmt.elseBranch == nil {
		return printer.parenthesize2("if", stmt.condition, stmt.thenBranch)
	}
	return printer.parenthesize2("if-else", stmt.condition, stmt.thenBranch)
}

func (printer *AstPrinter) visitPrintStmt(stmt *Print) (interface{}, error) {
	return printer.parenthesize("print", stmt.expression)
}

func (printer *AstPrinter) visitReturnStmt(stmt *Return) (interface{}, error) {
	if stmt.value == nil {
		return "(return)", nil
	}
	return printer.parenthesize("return", stmt.value)
}

func (printer *AstPrinter) visitVarStmt(stmt *Var) (interface{}, error) {
	if stmt.initializer == nil {
		return printer.parenthesize2("var", stmt.name)
	}
	return printer.parenthesize2("var", stmt.name, "=", stmt.initializer)
}

func (printer *AstPrinter) visitWhileStmt(stmt *While) (interface{}, error) {
	return printer.parenthesize2("while", stmt.condition, stmt.body)
}

func (printer *AstPrinter) visitBreakStmt(stmt *Break) (interface{}, error) {
	return "(break)", nil
}

func (printer *AstPrinter) visitContinueStmt(stmt *Continue) (interface{}, error) {
	return "(continue)", nil
}

func (printer *AstPrinter) visitBinaryExpr(expr *Binary) (interface{}, error) {
	return printer.parenthesize(expr.operator.Lexeme, expr.left, expr.right)
}

func (printer *AstPrinter) visitGroupingExpr(expr *Grouping) (interface{}, error) {
	return printer.parenthesize("group", expr.expression)
}

func (printer *AstPrinter) visitLiteralExpr(expr *Literal) (interface{}, error) {
	if expr.value == nil {
		return "nil", nil
	}
	return expr.value, nil
}

func (printer *AstPrinter) visitUnaryExpr(expr *Unary) (interface{}, error) {
	return printer.parenthesize(expr.operator.Lexeme, expr.right)
}

func (printer *AstPrinter) visitVariableExpr(expr *Variable) (interface{}, error) {
	return expr.name, nil
}

func (printer *AstPrinter) visitAssignExpr(expr *Assign) (interface{}, error) {
	return printer.parenthesize(expr.name.Lexeme, expr.value)
}

func (printer *AstPrinter) visitLogicalExpr(expr *Logical) (interface{}, error) {
	return printer.parenthesize(expr.operator.Lexeme, expr.left, expr.right)
}

func (printer *AstPrinter) visitCallExpr(expr *Call) (interface{}, error) {
	return printer.parenthesize2("call", expr.callee, expr.arguments)
}

func (printer *AstPrinter) visitTernaryExpr(expr *Ternary) (interface{}, error) {
	return printer.parenthesize("?", expr.expr, expr.thenBranch, expr.elseBranch)
}

func (printer *AstPrinter) visitArrayLiteralExpr(expr *ArrayLiteral) (interface{}, error) {
	return printer.parenthesize(expr.bracket.Lexeme, expr.items...)
}

func (printer *AstPrinter) visitLambdaExpr(expr *Lambda) (interface{}, error) {
	bs := bytes.NewBufferString("")
	bs.WriteString("fun (")
	for _, param := range expr.params {
		if param != expr.params[0] {
			bs.WriteString(" ")
		}
		bs.WriteString(param.Lexeme)
	}
	bs.WriteString(") ")
	for _, body := range expr.body {
		v, _ := body.accept(printer)
		bs.WriteString(v.(string))
	}
	bs.WriteString(")")
	return bs.String(), nil
}

func (printer *AstPrinter) visitIndexExpr(expr *Index) (interface{}, error) {
	return "", nil
}

func (printer *AstPrinter) visitGetExpr(expr *Get) (interface{}, error) {
	return printer.parenthesize2(".", expr.object, expr.name.Lexeme)
}

func (printer *AstPrinter) visitSetExpr(expr *Set) (interface{}, error) {
	return printer.parenthesize2("=", expr.object, expr.name.Lexeme, expr.value)
}

func (printer *AstPrinter) visitArraySetExpr(expr *ArraySet) (interface{}, error) {
	return printer.parenthesize2("=", expr.left, expr.name.Lexeme, expr.index, expr.value)
}

func (printer *AstPrinter) visitThisExpr(expr *This) (interface{}, error) {
	return "this", nil
}

func (printer *AstPrinter) visitSuperExpr(expr *Super) (interface{}, error) {
	return printer.parenthesize2("super", expr.method)
}

func (printer *AstPrinter) parenthesize(name string, exprs ...Expr) (interface{}, error) {
	bs := bytes.NewBufferString("")
	bs.WriteString("(")
	bs.WriteString(name)
	for _, expr := range exprs {
		bs.WriteString(" ")
		v, _ := expr.accept(printer)
		bs.WriteString(fmt.Sprintf("%v", v))
	}
	bs.WriteString(")")
	return bs.String(), nil
}

func (printer *AstPrinter) parenthesize2(name string, parts ...interface{}) (interface{}, error) {
	bs := bytes.NewBufferString("")
	bs.WriteString("(")
	bs.WriteString(name)
	printer.transform(bs, parts...)
	bs.WriteString(")")

	return bs.String(), nil
}

func (printer *AstPrinter) transform(bs *bytes.Buffer, parts ...interface{}) {
	for _, part := range parts {
		bs.WriteString(" ")
		if expr, ok := part.(Expr); ok {
			v, _ := expr.accept(printer)
			bs.WriteString(v.(string))
		} else if stmt, ok := part.(Stmt); ok {
			v, _ := stmt.accept(printer)
			bs.WriteString(v.(string))
		} else if tk, ok := part.(*Token); ok {
			bs.WriteString(tk.Lexeme)
		} else if list, ok := part.([]Expr); ok {
			for _, expr := range list {
				v, _ := expr.accept(printer)
				bs.WriteString(v.(string))
			}
		} else {
			bs.WriteString(part.(string))
		}
	}
}
