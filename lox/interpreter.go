package lox

import (
	"fmt"
	"strconv"
)

type Interpreter struct {
	environment *Environment
	locals      map[Expr]int
}

var (
	globals = NewEnvironment(nil)
)

func NewInterpreter() *Interpreter {
	/**
	 * 注册内置函数
	 */
	globals.Define("clock", NewClock())
	globals.Define("len", NewLen())
	globals.Define("string", NewString())

	return &Interpreter{
		environment: globals,
		locals:      map[Expr]int{},
	}
}

func (this *Interpreter) Interpret(statements []Stmt) {
	defer func(p *Interpreter) {
		if re, ok := recover().(*runtimeError); ok {
			p.runtimeError(re)
		} else if re != nil {
			panic(re)
		}
	}(this)
	for _, statement := range statements {
		this.execute(statement)
	}
}

// execute
func (this *Interpreter) execute(stmt Stmt) {
	stmt.accept(this)
}

// resolve
func (this *Interpreter) resolve(expr Expr, depth int) {
	this.locals[expr] = depth
}

// executeBlock
func (this *Interpreter) executeBlock(statements []Stmt, env *Environment) {
	previous := this.environment
	defer func() {
		this.environment = previous
	}()
	this.environment = env
	for _, statement := range statements {
		this.execute(statement)
	}
}

// visitBlockStmt
func (this *Interpreter) visitBlockStmt(stmt *Block) interface{} {
	this.executeBlock(stmt.statements, NewEnvironment(this.environment))
	return nil
}

// visitClassStmt
func (this *Interpreter) visitClassStmt(stmt *Class) interface{} {
	var superclass interface{} = nil
	if stmt.superclass != nil {
		superclass = this.evaluate(stmt.superclass)
		if _, ok := superclass.(*LoxClass); !ok {
			panic(NewRuntimeError(stmt.superclass.name, "superclass must be a class."))
		}
	}

	this.environment.Define(stmt.name.Lexeme, nil)
	if stmt.superclass != nil {
		this.environment = NewEnvironment(this.environment)
		this.environment.Define("super", superclass)
	}

	methods := map[string]LoxCallable{}
	for _, method := range stmt.methods {
		function := NewLoxFunction(method, this.environment, method.name.Lexeme == "init")
		methods[method.name.Lexeme] = function
	}

	superklass, _ := superclass.(*LoxClass)
	class := NewLoxClass(stmt.name.Lexeme, superklass, methods)
	if superklass != nil {
		this.environment = this.environment.enclosing
	}

	this.environment.Assign(stmt.name, class)
	return nil
}

// visitLiteralExpr
func (this *Interpreter) visitLiteralExpr(expr *Literal) interface{} {
	return expr.value
}

// visitLogicalExpr
func (this *Interpreter) visitLogicalExpr(expr *Logical) interface{} {
	left := this.evaluate(expr.left)
	if expr.operator.Type == OR {
		if this.isTruthy(left) {
			return left
		}
	} else {
		if !this.isTruthy(left) {
			return left
		}
	}
	return this.evaluate(expr.right)
}

// visitSetExpr
func (this *Interpreter) visitSetExpr(expr *Set) interface{} {
	object := this.evaluate(expr.object)

	instance, ok := object.(*LoxInstance)
	if !ok {
		panic(NewRuntimeError(expr.name, "only instance have fields."))
	}
	value := this.evaluate(expr.value)
	instance.Set(expr.name, value)
	return value
}

// visitSuperExpr
func (this *Interpreter) visitSuperExpr(expr *Super) interface{} {
	distance := this.locals[expr]
	superclass, _ := this.environment.GetAt(distance, "super").(*LoxClass)
	object, _ := this.environment.GetAt(distance-1, "this").(*LoxInstance)

	method := superclass.findMethod(expr.method.Lexeme)
	if method == nil {
		panic(NewRuntimeError(expr.method, "undefined property '"+expr.method.Lexeme+"'."))
	}
	return method.(*LoxFunction).Bind(object)
}

// visitThisExpr
func (this *Interpreter) visitThisExpr(expr *This) interface{} {
	return this.lookUpVariable(expr.keyword, expr)
}

// visitGroupingExpr
func (this *Interpreter) visitGroupingExpr(expr *Grouping) interface{} {
	return this.evaluate(expr.expression)
}

// visitUnaryExpr
func (this *Interpreter) visitUnaryExpr(expr *Unary) interface{} {
	right := this.evaluate(expr.right)
	switch expr.operator.Type {
	case MINUS:
		this.checkNumberOperand(expr.operator, right)
		return -right.(float64)
	case BANG:
		return !this.isTruthy(right)
	case PLUS_PLUS:
		this.checkVariable(expr.operator, expr.right, "operand of an increment operator must be a variable.")
		this.checkNumberOperand(expr.operator, right)

		value := right.(float64)
		this.environment.Assign(expr.right.(*Variable).name, value+1)
		return IfFloat(expr.postfix, value, value+1)
	case MINUS_MINUS:
		this.checkVariable(expr.operator, expr.right, "operand of a decrement operator must be a variable.")
		this.checkNumberOperand(expr.operator, right)

		value := right.(float64)
		this.environment.Assign(expr.right.(*Variable).name, value-1)
		return IfFloat(expr.postfix, value, value-1)
	}
	return nil
}

// visitVariableExpr
func (this *Interpreter) visitVariableExpr(expr *Variable) interface{} {
	return this.lookUpVariable(expr.name, expr)
}

func (this *Interpreter) lookUpVariable(name *Token, expr Expr) interface{} {
	distance, ok := this.locals[expr]
	if ok {
		return this.environment.GetAt(distance, name.Lexeme)
	} else {
		return globals.Get(name)
	}
}

// visitTernaryExpr
func (this *Interpreter) visitTernaryExpr(expr *Ternary) interface{} {
	expression := this.evaluate(expr.expr)
	if this.isTruthy(expression) {
		return this.evaluate(expr.thenBranch)
	} else {
		return this.evaluate(expr.elseBranch)
	}
}

// visitBinaryExpr
func (this *Interpreter) visitBinaryExpr(expr *Binary) interface{} {
	left := this.evaluate(expr.left)
	right := this.evaluate(expr.right)

	switch expr.operator.Type {
	case GREATER:
		this.checkNumberOperands(expr.operator, left, right)
		return left.(float64) > right.(float64)
	case GREATER_EQUAL:
		this.checkNumberOperands(expr.operator, left, right)
		return left.(float64) >= right.(float64)
	case LESS:
		this.checkNumberOperands(expr.operator, left, right)
		return left.(float64) < right.(float64)
	case LESS_EQUAL:
		this.checkNumberOperands(expr.operator, left, right)
		return left.(float64) <= right.(float64)
	case MINUS:
		this.checkNumberOperands(expr.operator, left, right)
		return left.(float64) - right.(float64)
	case BANG_EQUAL:
		return !this.isEqual(left, right)
	case EQUAL_EQUAL:
		return this.isEqual(left, right)
	case PLUS:
		v1, ok1 := left.(float64)
		v2, ok2 := right.(float64)
		if ok1 && ok2 {
			return v1 + v2
		}

		s1, ok1 := left.(string)
		s2, ok2 := right.(string)
		if ok1 && ok2 {
			return s1 + s2
		}
		panic(NewRuntimeError(expr.operator, "operands must be two numbers or two strings."))
	case SLASH:
		this.checkNumberOperands(expr.operator, left, right)
		return left.(float64) / right.(float64)
	case STAR:
		this.checkNumberOperands(expr.operator, left, right)
		return left.(float64) * right.(float64)
	case COMMA:
		return right
	}
	return nil
}

func (this *Interpreter) visitCallExpr(expr *Call) interface{} {
	callee := this.evaluate(expr.callee)

	var arguments []interface{}
	for _, argument := range expr.arguments {
		arguments = append(arguments, this.evaluate(argument))
	}
	function, ok := callee.(LoxCallable)
	if !ok {
		panic(NewRuntimeError(expr.paren, "can only call functions and classes."))
	}

	if len(arguments) != function.Arity() {
		panic(NewRuntimeError(expr.paren, "expected "+strconv.Itoa(function.Arity())+
			" arguments but got "+strconv.Itoa(len(arguments))))
	}

	return function.Call(this, arguments)
}

func (this *Interpreter) visitGetExpr(expr *Get) interface{} {
	object := this.evaluate(expr.object)
	if instance, ok := object.(*LoxInstance); ok && instance != nil {
		return instance.Get(expr.name)
	}
	panic(NewRuntimeError(expr.name, "only instances have properties."))
}

func (this *Interpreter) visitIndexExpr(expr *Index) interface{} {
	left := this.evaluate(expr.left)
	array, ok := left.(LoxIterator)
	if !ok {
		panic(NewRuntimeError(expr.name, "can only iterator can be subsetted."))
	}
	index := this.evaluate(expr.index)
	idx, ok := index.(float64)
	if !ok {
		panic(NewRuntimeError(expr.name, "expected valid express after '['"))
	}
	v, err := array.Get(int(idx))
	if err != nil {
		panic(NewRuntimeError(expr.name, err.Error()))
	}
	return v
}

func (this *Interpreter) visitExpressionStmt(stmt *Expression) interface{} {
	this.evaluate(stmt.expression)
	return nil
}

func (this *Interpreter) visitFunctionStmt(stmt *Function) interface{} {
	function := NewLoxFunction(stmt, this.environment, false)
	this.environment.Define(stmt.name.Lexeme, function)
	return nil
}

func (this *Interpreter) visitLambdaExpr(expr *Lambda) interface{} {
	return NewLoxLambda(expr, this.environment, false)
}

func (this *Interpreter) visitArrayLiteralExpr(expr *ArrayLiteral) interface{} {
	var items []interface{}
	for _, express := range expr.items {
		items = append(items, this.evaluate(express))
	}
	return NewLoxArray(items)
}

func (this *Interpreter) visitIfStmt(stmt *If) interface{} {
	if this.isTruthy(this.evaluate(stmt.condition)) {
		this.execute(stmt.thenBranch)
	} else if stmt.elseBranch != nil {
		this.execute(stmt.elseBranch)
	}
	return nil
}

func (this *Interpreter) visitReturnStmt(stmt *Return) interface{} {
	var value interface{} = nil
	if stmt.value != nil {
		value = this.evaluate(stmt.value)
	}
	panic(NewReturnExp(value))
}

func (this *Interpreter) visitPrintStmt(stmt *Print) interface{} {
	value := this.evaluate(stmt.expression)
	fmt.Println(this.stringify(value))
	return nil
}

func (this *Interpreter) visitVarStmt(stmt *Var) interface{} {
	var value interface{}
	if stmt.initializer != nil {
		value = this.evaluate(stmt.initializer)
	}
	this.environment.Define(stmt.name.Lexeme, value)
	return nil
}

func (this *Interpreter) visitWhileStmt(stmt *While) interface{} {
	visit := func(this *Interpreter, stmt *While) (value interface{}) {
		defer func() {
			if r := recover(); r != nil {
				switch v := r.(type) {
				case ContinueJump, BreakJump:
					value = v
				default:
					panic(r) //继续传播异常
				}
			}
		}()
		for this.isTruthy(this.evaluate(stmt.condition)) {
			this.execute(stmt.body)
		}
		return
	}
	for {
		_, ok := visit(this, stmt).(ContinueJump)
		if !ok {
			break
		}
	}
	return nil
}

func (this *Interpreter) visitBreakStmt(stmt *Break) interface{} {
	panic(BreakJump{})
}

func (this *Interpreter) visitContinueStmt(stmt *Continue) interface{} {
	panic(ContinueJump{})
}

func (this *Interpreter) visitAssignExpr(expr *Assign) interface{} {
	value := this.evaluate(expr.value)

	distance, ok := this.locals[expr]
	if ok {
		this.environment.AssignAt(distance, expr.name, value)
	} else {
		globals.Assign(expr.name, value)
	}
	return value
}

func (this *Interpreter) visitArraySetExpr(expr *ArraySet) interface{} {
	value := this.evaluate(expr.left)
	array, ok := value.(LoxIterator)
	if !ok {
		panic(NewRuntimeError(expr.name, "can only set array."))
	}
	if expr.index == nil {
		array.Add(this.evaluate(expr.value))
	} else {
		index := this.evaluate(expr.index).(float64)
		if err := array.Set(int(index), this.evaluate(expr.value)); err != nil {
			panic(NewRuntimeError(expr.name, err.Error()))
		}
	}
	return nil
}

func (this *Interpreter) evaluate(expr Expr) interface{} {
	return expr.accept(this)
}

func (this *Interpreter) isTruthy(obj interface{}) bool {
	if obj == nil {
		return false
	}
	if v, ok := obj.(bool); ok {
		return v
	}
	return true
}

func (this *Interpreter) isEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		return false
	}
	return a == b
}

func (this *Interpreter) checkNumberOperand(operator *Token, operand interface{}) {
	if _, ok := operand.(float64); ok {
		return
	}
	panic(NewRuntimeError(operator, "operand must be a number."))
}

func (this *Interpreter) checkNumberOperands(operator *Token, left, right interface{}) {
	_, ok1 := left.(float64)
	_, ok2 := right.(float64)
	if ok1 && ok2 {
		return
	}
	panic(NewRuntimeError(operator, "operands must be numbers."))
}

func (this *Interpreter) checkVariable(operator *Token, right interface{}, message string) {
	if _, ok := right.(*Variable); ok {
		return
	}
	panic(NewRuntimeError(operator, message))
}

func (this *Interpreter) stringify(obj interface{}) string {
	if obj == nil {
		return "nil"
	}
	if v, ok := obj.(float64); ok {
		return FloatVal(v)
	}
	return fmt.Sprintf("%v", obj)
}

func (this *Interpreter) runtimeError(err RuntimeError) {
	fmt.Println("[line " + strconv.Itoa(err.GetToken().Line) + "] " + err.GetMessage())
	hadRuntimeError = true
}
