package lox

type Stmt interface {
	accept(visitor VisitorStmt) (interface{}, error)
}

type VisitorStmt interface {
	visitBlockStmt(stmt *Block) (interface{}, error)
	visitClassStmt(stmt *Class) (interface{}, error)
	visitExpressionStmt(stmt *Expression) (interface{}, error)
	visitFunctionStmt(stmt *Function) (interface{}, error)
	visitIfStmt(stmt *If) (interface{}, error)
	visitPrintStmt(stmt *Print) (interface{}, error)
	visitReturnStmt(stmt *Return) (interface{}, error)
	visitVarStmt(stmt *Var) (interface{}, error)
	visitWhileStmt(stmt *While) (interface{}, error)
	visitBreakStmt(stmt *Break) (interface{}, error)
	visitContinueStmt(stmt *Continue) (interface{}, error)
}

func NewBlock(statements []Stmt) *Block {
	return &Block{
		statements : statements,
	}
}

type Block struct {
	statements []Stmt
}

func (this *Block) accept(visitor VisitorStmt) (interface{}, error) {
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

func (this *Class) accept(visitor VisitorStmt) (interface{}, error) {
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

func (this *Expression) accept(visitor VisitorStmt) (interface{}, error) {
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

func (this *Function) accept(visitor VisitorStmt) (interface{}, error) {
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

func (this *If) accept(visitor VisitorStmt) (interface{}, error) {
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

func (this *Print) accept(visitor VisitorStmt) (interface{}, error) {
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

func (this *Return) accept(visitor VisitorStmt) (interface{}, error) {
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

func (this *Var) accept(visitor VisitorStmt) (interface{}, error) {
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

func (this *While) accept(visitor VisitorStmt) (interface{}, error) {
	return visitor.visitWhileStmt(this)
}

func NewBreak() *Break {
	return &Break{
	}
}

type Break struct {
	
}

func (this *Break) accept(visitor VisitorStmt) (interface{}, error) {
	return visitor.visitBreakStmt(this)
}

func NewContinue() *Continue {
	return &Continue{
	}
}

type Continue struct {
	
}

func (this *Continue) accept(visitor VisitorStmt) (interface{}, error) {
	return visitor.visitContinueStmt(this)
}
