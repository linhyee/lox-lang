package lox

type Stmt interface {
	accept(visitor VisitorStmt) interface{}
}

type VisitorStmt interface {
	visitBlockStmt(stmt *Block) interface{}
	visitClassStmt(stmt *Class) interface{}
	visitExpressionStmt(stmt *Expression) interface{}
	visitFunctionStmt(stmt *Function) interface{}
	visitIfStmt(stmt *If) interface{}
	visitPrintStmt(stmt *Print) interface{}
	visitReturnStmt(stmt *Return) interface{}
	visitVarStmt(stmt *Var) interface{}
	visitWhileStmt(stmt *While) interface{}
	visitBreakStmt(stmt *Break) interface{}
	visitContinueStmt(stmt *Continue) interface{}
}

func NewBlock(statements []Stmt) *Block {
	return &Block{
		statements : statements,
	}
}

type Block struct {
	statements []Stmt
}

func (this *Block) accept(visitor VisitorStmt) interface{} {
	return visitor.visitBlockStmt(this)
}

func NewClass(name *Token, superclass *Variable, methods []*Function) *Class {
	return &Class{
		name : name,
		superclass : superclass,
		methods : methods,
	}
}

type Class struct {
	name *Token
	superclass *Variable
	methods []*Function
}

func (this *Class) accept(visitor VisitorStmt) interface{} {
	return visitor.visitClassStmt(this)
}

func NewExpression(expression Expr) *Expression {
	return &Expression{
		expression : expression,
	}
}

type Expression struct {
	expression Expr
}

func (this *Expression) accept(visitor VisitorStmt) interface{} {
	return visitor.visitExpressionStmt(this)
}

func NewFunction(name *Token, params []*Token, body []Stmt) *Function {
	return &Function{
		name : name,
		params : params,
		body : body,
	}
}

type Function struct {
	name *Token
	params []*Token
	body []Stmt
}

func (this *Function) accept(visitor VisitorStmt) interface{} {
	return visitor.visitFunctionStmt(this)
}

func NewIf(condition Expr, thenBranch Stmt, elseBranch Stmt) *If {
	return &If{
		condition : condition,
		thenBranch : thenBranch,
		elseBranch : elseBranch,
	}
}

type If struct {
	condition Expr
	thenBranch Stmt
	elseBranch Stmt
}

func (this *If) accept(visitor VisitorStmt) interface{} {
	return visitor.visitIfStmt(this)
}

func NewPrint(expression Expr) *Print {
	return &Print{
		expression : expression,
	}
}

type Print struct {
	expression Expr
}

func (this *Print) accept(visitor VisitorStmt) interface{} {
	return visitor.visitPrintStmt(this)
}

func NewReturn(keyword *Token, value Expr) *Return {
	return &Return{
		keyword : keyword,
		value : value,
	}
}

type Return struct {
	keyword *Token
	value Expr
}

func (this *Return) accept(visitor VisitorStmt) interface{} {
	return visitor.visitReturnStmt(this)
}

func NewVar(name *Token, initializer Expr) *Var {
	return &Var{
		name : name,
		initializer : initializer,
	}
}

type Var struct {
	name *Token
	initializer Expr
}

func (this *Var) accept(visitor VisitorStmt) interface{} {
	return visitor.visitVarStmt(this)
}

func NewWhile(condition Expr, body Stmt) *While {
	return &While{
		condition : condition,
		body : body,
	}
}

type While struct {
	condition Expr
	body Stmt
}

func (this *While) accept(visitor VisitorStmt) interface{} {
	return visitor.visitWhileStmt(this)
}

func NewBreak() *Break {
	return &Break{
	}
}

type Break struct {
	
}

func (this *Break) accept(visitor VisitorStmt) interface{} {
	return visitor.visitBreakStmt(this)
}

func NewContinue() *Continue {
	return &Continue{
	}
}

type Continue struct {
	
}

func (this *Continue) accept(visitor VisitorStmt) interface{} {
	return visitor.visitContinueStmt(this)
}

