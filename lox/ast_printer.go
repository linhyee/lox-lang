package lox

import (
	"bytes"
	"fmt"
)

type AstPrinter struct{}

func (printer *AstPrinter) printExpr(expr Expr) string {
	return expr.accept(printer).(string)
}

func (printer *AstPrinter) printStmt(stmt Stmt) string {
	return stmt.accept(printer).(string)
}

func (printer *AstPrinter) visitBlockStmt(stmt *Block) interface{} {
	bs := bytes.NewBufferString("")
	bs.WriteString("block ")
	for _, statement := range stmt.statements {
		bs.WriteString(statement.accept(printer).(string))
	}
	bs.WriteString(")")
	return bs.String()
}

func (printer *AstPrinter) visitClassStmt(stmt *Class) interface{} {
	bs := bytes.NewBufferString("")
	bs.WriteString("class " + stmt.name.Lexeme)
	if stmt.superclass != nil {
		bs.WriteString(" < " + printer.printExpr(stmt.superclass))
	}
	for _, method := range stmt.methods {
		bs.WriteString(" " + printer.printStmt(method))
	}
	bs.WriteString(")")
	return bs.String()
}

func (printer *AstPrinter) visitExpressionStmt(stmt *Expression) interface{} {
	return printer.parenthesize(";", stmt.expression)
}

func (printer *AstPrinter) visitFunctionStmt(stmt *Function) interface{} {
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
		bs.WriteString(body.accept(printer).(string))
	}
	bs.WriteString(")")
	return bs.String()
}

func (printer *AstPrinter) visitIfStmt(stmt *If) interface{} {
	if stmt.elseBranch == nil {
		return printer.parenthesize2("if", stmt.condition, stmt.thenBranch)
	}
	return printer.parenthesize2("if-else", stmt.condition, stmt.thenBranch)
}

func (printer *AstPrinter) visitPrintStmt(stmt *Print) interface{} {
	return printer.parenthesize("print", stmt.expression)
}

func (printer *AstPrinter) visitReturnStmt(stmt *Return) interface{} {
	if stmt.value == nil {
		return "(return)"
	}
	return printer.parenthesize("return", stmt.value)
}

func (printer *AstPrinter) visitVarStmt(stmt *Var) interface{} {
	if stmt.initializer == nil {
		return printer.parenthesize2("var", stmt.name)
	}
	return printer.parenthesize2("var", stmt.name, "=", stmt.initializer)
}

func (printer *AstPrinter) visitWhileStmt(stmt *While) interface{} {
	return printer.parenthesize2("while", stmt.condition, stmt.body)
}

func (printer *AstPrinter) visitBreakStmt(stmt *Break) interface{} {
	return "(break)"
}

func (printer *AstPrinter) visitContinueStmt(stmt *Continue) interface{} {
	return "(continue)"
}

func (printer *AstPrinter) visitBinaryExpr(expr *Binary) interface{} {
	return printer.parenthesize(expr.operator.Lexeme, expr.left, expr.right)
}

func (printer *AstPrinter) visitGroupingExpr(expr *Grouping) interface{} {
	return printer.parenthesize("group", expr.expression)
}

func (printer *AstPrinter) visitLiteralExpr(expr *Literal) interface{} {
	if expr.value == nil {
		return "nil"
	}
	return expr.value
}

func (printer *AstPrinter) visitUnaryExpr(expr *Unary) interface{} {
	return printer.parenthesize(expr.operator.Lexeme, expr.right)
}

func (printer *AstPrinter) visitVariableExpr(expr *Variable) interface{} {
	return expr.name
}

func (printer *AstPrinter) visitAssignExpr(expr *Assign) interface{} {
	return printer.parenthesize(expr.name.Lexeme, expr.value)
}

func (printer *AstPrinter) visitLogicalExpr(expr *Logical) interface{} {
	return printer.parenthesize(expr.operator.Lexeme, expr.left, expr.right)
}

func (printer *AstPrinter) visitCallExpr(expr *Call) interface{} {
	return printer.parenthesize2("call", expr.callee, expr.arguments)
}

func (printer *AstPrinter) visitTernaryExpr(expr *Ternary) interface{} {
	return printer.parenthesize("?", expr.expr, expr.thenBranch, expr.elseBranch)
}

func (printer *AstPrinter) visitArrayLiteralExpr(expr *ArrayLiteral) interface{} {
	return printer.parenthesize(expr.bracket.Lexeme, expr.items...)
}

func (printer *AstPrinter) visitLambdaExpr(expr *Lambda) interface{} {
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
		bs.WriteString(body.accept(printer).(string))
	}
	bs.WriteString(")")
	return bs.String()
}

func (printer *AstPrinter) visitIndexExpr(expr *Index) interface{} {
	return ""
}

func (printer *AstPrinter) visitGetExpr(expr *Get) interface{} {
	return printer.parenthesize2(".", expr.object, expr.name.Lexeme)
}

func (printer *AstPrinter) visitSetExpr(expr *Set) interface{} {
	return printer.parenthesize2("=", expr.object, expr.name.Lexeme, expr.value)
}

func (printer *AstPrinter) visitArraySetExpr(expr *ArraySet) interface{} {
	return printer.parenthesize2("=", expr.left, expr.name.Lexeme, expr.index, expr.value)
}

func (printer *AstPrinter) visitThisExpr(expr *This) interface{} {
	return "this"
}

func (printer *AstPrinter) visitSuperExpr(expr *Super) interface{} {
	return printer.parenthesize2("super", expr.method)
}

func (printer *AstPrinter) parenthesize(name string, exprs ...Expr) string {
	bs := bytes.NewBufferString("")
	bs.WriteString("(")
	bs.WriteString(name)
	for _, expr := range exprs {
		bs.WriteString(" ")
		bs.WriteString(fmt.Sprintf("%v", expr.accept(printer)))
	}
	bs.WriteString(")")
	return bs.String()
}

func (printer *AstPrinter) parenthesize2(name string, parts ...interface{}) string {
	bs := bytes.NewBufferString("")
	bs.WriteString("(")
	bs.WriteString(name)
	printer.transform(bs, parts)
	bs.WriteString(")")

	return bs.String()
}

func (printer *AstPrinter) transform(bs *bytes.Buffer, parts ...interface{}) {
	for _, part := range parts {
		bs.WriteString(" ")
		if expr, ok := part.(Expr); ok {
			bs.WriteString(expr.accept(printer).(string))
		} else if stmt, ok := part.(Stmt); ok {
			bs.WriteString(stmt.accept(printer).(string))
		} else if tk, ok := part.(*Token); ok {
			bs.WriteString(tk.Lexeme)
		} else if list, ok := part.([]Expr); ok {
			for _, expr := range list {
				bs.WriteString(expr.accept(printer).(string))
			}
		} else {
			bs.WriteString(part.(string))
		}
	}
}
