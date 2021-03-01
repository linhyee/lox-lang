package lox

type FunctionType int

type ClassType int

const (
	FT_NONE FunctionType = iota
	FT_FUNCTION
	FT_INITIALIZER
	FT_METHOD

	CT_NONE ClassType = iota
	CT_SUBCLASS
	CT_CLASS
)

func NewResolver(interpreter *Interpreter) *Resolver {
	return &Resolver{
		interpreter:     interpreter,
		scopes:          NewStack(),
		currentFunction: FT_NONE,
		currentClass:    CT_NONE,
	}
}

type Resolver struct {
	interpreter     *Interpreter
	scopes          *Stack
	currentFunction FunctionType
	currentClass    ClassType
}

func (this *Resolver) visitBlockStmt(stmt *Block) interface{} {
	this.beginScope()
	this.resolve(stmt.statements)
	this.endScope()
	return nil
}

func (this *Resolver) resolve(statements []Stmt) {
	for _, statement := range statements {
		this.resolveStmt(statement)
	}
}

func (this *Resolver) resolveStmt(stmt Stmt) {
	stmt.accept(this)
}

func (this *Resolver) resolveExpr(expr Expr) {
	expr.accept(this)
}

func (this *Resolver) beginScope() {
	this.scopes.Push(map[string]bool{})
}

func (this *Resolver) endScope() {
	_, _ = this.scopes.Pop()
}

func (this *Resolver) visitVarStmt(stmt *Var) interface{} {
	this.declare(stmt.name)
	if stmt.initializer != nil {
		this.resolveExpr(stmt.initializer)
	}
	this.define(stmt.name)
	return nil
}

func (this *Resolver) declare(name *Token) {
	if this.scopes.IsEmpty() {
		return
	}
	scope := this.scopes.Top().(map[string]bool)
	if _, ok := scope[name.Lexeme]; ok {
		errorToken(name, "already variable with this name in this scope.")
	}
	scope[name.Lexeme] = false
}

func (this *Resolver) define(name *Token) {
	if this.scopes.IsEmpty() {
		return
	}
	scope := this.scopes.Top().(map[string]bool)
	scope[name.Lexeme] = true
}

func (this *Resolver) visitVariableExpr(expr *Variable) interface{} {
	if !this.scopes.IsEmpty() {
		if value, ok := this.scopes.Top().(map[string]bool)[expr.name.Lexeme]; ok && value == false {
			errorToken(expr.name, "can't read local variable in its own initializer.")
		}
	}
	this.resolveLocal(expr, expr.name)
	return nil
}

func (this *Resolver) resolveLocal(expr Expr, name *Token) {
	for i := this.scopes.Size() - 1; i >= 0; i-- {
		if value, err := this.scopes.Get(i); err == nil {
			if _, ok := value.(map[string]bool)[name.Lexeme]; ok {
				this.interpreter.resolve(expr, this.scopes.Size()-1-i)
				return
			}
		}
	}
}

func (this *Resolver) visitAssignExpr(expr *Assign) interface{} {
	this.resolveExpr(expr.value)
	this.resolveLocal(expr, expr.name)
	return nil
}

func (this *Resolver) visitClassStmt(stmt *Class) interface{} {
	enclosingClass := this.currentClass
	this.currentClass = CT_CLASS

	this.declare(stmt.name)
	this.define(stmt.name)

	if stmt.superclass != nil && stmt.name.Lexeme == stmt.superclass.name.Lexeme {
		errorToken(stmt.superclass.name, "a class can't inherit from itself.")
	}

	if stmt.superclass != nil {
		this.currentClass = CT_SUBCLASS
		this.resolveExpr(stmt.superclass)
	}
	if stmt.superclass != nil {
		this.beginScope()
		this.scopes.Top().(map[string]bool)["super"] = true
	}

	this.beginScope()
	this.scopes.Top().(map[string]bool)["this"] = true
	for _, method := range stmt.methods {
		declaration := FT_METHOD
		if method.name.Lexeme == "init" {
			declaration = FT_INITIALIZER
		}
		this.resolveFunction(method, declaration)
	}
	this.endScope()
	if stmt.superclass != nil {
		this.endScope()
	}
	this.currentClass = enclosingClass
	return nil
}

func (this *Resolver) visitFunctionStmt(stmt *Function) interface{} {
	this.declare(stmt.name)
	this.define(stmt.name)
	this.resolveFunction(stmt, FT_FUNCTION)
	return nil
}

func (this *Resolver) resolveFunction(function *Function, ft FunctionType) {
	this.function(function.params, function.body, ft)
}

func (this *Resolver) resolveLambda(lambda *Lambda, ft FunctionType) {
	this.function(lambda.params, lambda.body, ft)
}

func (this *Resolver) function(params []*Token, body []Stmt, ft FunctionType) {
	enclosingFunction := this.currentFunction
	this.currentFunction = ft

	this.beginScope()

	for _, param := range params {
		this.declare(param)
		this.define(param)
	}

	this.resolve(body)

	this.endScope()
	this.currentFunction = enclosingFunction
}

func (this *Resolver) visitExpressionStmt(stmt *Expression) interface{} {
	this.resolveExpr(stmt.expression)
	return nil
}

func (this *Resolver) visitIfStmt(stmt *If) interface{} {
	this.resolveExpr(stmt.condition)
	this.resolveStmt(stmt.thenBranch)
	if stmt.elseBranch != nil {
		this.resolveStmt(stmt.elseBranch)
	}
	return nil
}

func (this *Resolver) visitPrintStmt(stmt *Print) interface{} {
	this.resolveExpr(stmt.expression)
	return nil
}

func (this *Resolver) visitReturnStmt(stmt *Return) interface{} {
	if this.currentFunction == FT_NONE {
		errorToken(stmt.keyword, "can't return from top-level code.")
	}
	if stmt.value != nil {
		if this.currentFunction == FT_INITIALIZER {
			errorToken(stmt.keyword, "can't return a value from an initializer.")
		}
		this.resolveExpr(stmt.value)
	}
	return nil
}

func (this *Resolver) visitWhileStmt(stmt *While) interface{} {
	this.resolveExpr(stmt.condition)
	this.resolveStmt(stmt.body)
	return nil
}

func (this *Resolver) visitBreakStmt(stmt *Break) interface{} {
	return nil
}

func (this *Resolver) visitContinueStmt(stmt *Continue) interface{} {
	return nil
}

func (this *Resolver) visitBinaryExpr(expr *Binary) interface{} {
	this.resolveExpr(expr.left)
	this.resolveExpr(expr.right)
	return nil
}

func (this *Resolver) visitCallExpr(expr *Call) interface{} {
	this.resolveExpr(expr.callee)
	for _, argument := range expr.arguments {
		this.resolveExpr(argument)
	}
	return nil
}

func (this *Resolver) visitGroupingExpr(expr *Grouping) interface{} {
	this.resolveExpr(expr.expression)
	return nil
}

func (this *Resolver) visitLiteralExpr(expr *Literal) interface{} {
	return nil
}

func (this *Resolver) visitLogicalExpr(expr *Logical) interface{} {
	this.resolveExpr(expr.left)
	this.resolveExpr(expr.right)
	return nil
}

func (this *Resolver) visitSetExpr(expr *Set) interface{} {
	this.resolveExpr(expr.value)
	this.resolveExpr(expr.object)
	return nil
}

func (this *Resolver) visitSuperExpr(expr *Super) interface{} {
	if this.currentClass == CT_NONE {
		errorToken(expr.keyword, "can't use 'super' outside of a class.")
	} else if this.currentClass != CT_SUBCLASS {
		errorToken(expr.keyword, "can't use 'super' in a class with no superclass.")
	}
	this.resolveLocal(expr, expr.keyword)
	return nil
}

func (this *Resolver) visitThisExpr(expr *This) interface{} {
	if this.currentClass == CT_NONE {
		errorToken(expr.keyword, "can't use 'this' outside of a class.")
		return nil
	}
	this.resolveLocal(expr, expr.keyword)
	return nil
}

func (this *Resolver) visitGetExpr(expr *Get) interface{} {
	this.resolveExpr(expr.object)
	return nil
}

func (this *Resolver) visitUnaryExpr(expr *Unary) interface{} {
	this.resolveExpr(expr.right)
	return nil
}

func (this *Resolver) visitTernaryExpr(expr *Ternary) interface{} {
	this.resolveExpr(expr.expr)
	this.resolveExpr(expr.thenBranch)
	this.resolveExpr(expr.elseBranch)
	return nil
}

func (this *Resolver) visitLambdaExpr(expr *Lambda) interface{} {
	this.resolveLambda(expr, FT_FUNCTION)
	return nil
}

func (this *Resolver) visitIndexExpr(expr *Index) interface{} {
	this.resolveExpr(expr.left)
	this.resolveExpr(expr.index)
	return nil
}

func (this *Resolver) visitArraySetExpr(expr *ArraySet) interface{} {
	this.resolveExpr(expr.left)
	if expr.index != nil {
		this.resolveExpr(expr.index)
	}
	this.resolveExpr(expr.value)
	return nil
}

func (this *Resolver) visitArrayLiteralExpr(expr *ArrayLiteral) interface{} {
	for _, expr := range expr.items {
		this.resolveExpr(expr)
	}
	return nil
}
