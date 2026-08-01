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
		if r := recover(); r != nil {
			if re, ok := r.(*runtimeError); ok {
				p.runtimeError(re)
			} else if re, ok := r.(RuntimeError); ok {
				p.runtimeError(re)
			} else if r != nil {
				panic(r)
			}
		}
	}(this)
	for _, statement := range statements {
		if err := this.execute(statement); err != nil {
			if cf, ok := IsControlFlow(err); ok {
				if cf.IsReturn() {
					continue
				}
			}
			if re, ok := err.(RuntimeError); ok {
				this.runtimeError(re)
				return
			}
			panic(err)
		}
	}
}

func (this *Interpreter) execute(stmt Stmt) error {
	_, err := stmt.accept(this)
	return err
}

func (this *Interpreter) resolve(expr Expr, depth int) {
	this.locals[expr] = depth
}

func (this *Interpreter) executeBlock(statements []Stmt, env *Environment) error {
	previous := this.environment
	defer func() {
		this.environment = previous
	}()
	this.environment = env
	for _, statement := range statements {
		if err := this.execute(statement); err != nil {
			return err
		}
	}
	return nil
}

func (this *Interpreter) visitBlockStmt(stmt *Block) (interface{}, error) {
	return nil, this.executeBlock(stmt.statements, NewEnvironment(this.environment))
}

func (this *Interpreter) visitClassStmt(stmt *Class) (interface{}, error) {
	var superclass interface{} = nil
	if stmt.superclass != nil {
		sc, err := this.evaluate(stmt.superclass)
		if err != nil {
			return nil, err
		}
		superclass = sc
		if _, ok := superclass.(*LoxClass); !ok {
			return nil, NewRuntimeError(stmt.superclass.name, "superclass must be a class.")
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
	return nil, nil
}

func (this *Interpreter) visitLiteralExpr(expr *Literal) (interface{}, error) {
	return expr.value, nil
}

func (this *Interpreter) visitLogicalExpr(expr *Logical) (interface{}, error) {
	left, err := this.evaluate(expr.left)
	if err != nil {
		return nil, err
	}
	if expr.operator.Type == OR {
		if this.isTruthy(left) {
			return left, nil
		}
	} else {
		if !this.isTruthy(left) {
			return left, nil
		}
	}
	return this.evaluate(expr.right)
}

func (this *Interpreter) visitSetExpr(expr *Set) (interface{}, error) {
	object, err := this.evaluate(expr.object)
	if err != nil {
		return nil, err
	}

	instance, ok := object.(*LoxInstance)
	if !ok {
		return nil, NewRuntimeError(expr.name, "only instance have fields.")
	}
	value, err := this.evaluate(expr.value)
	if err != nil {
		return nil, err
	}
	instance.Set(expr.name, value)
	return value, nil
}

func (this *Interpreter) visitSuperExpr(expr *Super) (interface{}, error) {
	distance := this.locals[expr]
	superclass, _ := this.environment.GetAt(distance, "super").(*LoxClass)
	object, _ := this.environment.GetAt(distance-1, "this").(*LoxInstance)

	method := superclass.findMethod(expr.method.Lexeme)
	if method == nil {
		return nil, NewRuntimeError(expr.method, "undefined property '"+expr.method.Lexeme+"'.")
	}
	return method.(*LoxFunction).Bind(object), nil
}

func (this *Interpreter) visitThisExpr(expr *This) (interface{}, error) {
	return this.lookUpVariable(expr.keyword, expr)
}

func (this *Interpreter) visitGroupingExpr(expr *Grouping) (interface{}, error) {
	return this.evaluate(expr.expression)
}

func (this *Interpreter) visitUnaryExpr(expr *Unary) (interface{}, error) {
	right, err := this.evaluate(expr.right)
	if err != nil {
		return nil, err
	}
	switch expr.operator.Type {
	case MINUS:
		if err := this.checkNumberOperand(expr.operator, right); err != nil {
			return nil, err
		}
		return -right.(float64), nil
	case BANG:
		return !this.isTruthy(right), nil
	case PLUS_PLUS:
		if err := this.checkVariable(expr.operator, expr.right, "operand of an increment operator must be a variable."); err != nil {
			return nil, err
		}
		if err := this.checkNumberOperand(expr.operator, right); err != nil {
			return nil, err
		}

		value := right.(float64)
		this.environment.Assign(expr.right.(*Variable).name, value+1)
		return IfFloat(expr.postfix, value, value+1), nil
	case MINUS_MINUS:
		if err := this.checkVariable(expr.operator, expr.right, "operand of a decrement operator must be a variable."); err != nil {
			return nil, err
		}
		if err := this.checkNumberOperand(expr.operator, right); err != nil {
			return nil, err
		}

		value := right.(float64)
		this.environment.Assign(expr.right.(*Variable).name, value-1)
		return IfFloat(expr.postfix, value, value-1), nil
	}
	return nil, nil
}

func (this *Interpreter) visitVariableExpr(expr *Variable) (interface{}, error) {
	return this.lookUpVariable(expr.name, expr)
}

func (this *Interpreter) lookUpVariable(name *Token, expr Expr) (interface{}, error) {
	distance, ok := this.locals[expr]
	if ok {
		return this.environment.GetAt(distance, name.Lexeme), nil
	} else {
		return globals.Get(name), nil
	}
}

func (this *Interpreter) visitTernaryExpr(expr *Ternary) (interface{}, error) {
	expression, err := this.evaluate(expr.expr)
	if err != nil {
		return nil, err
	}
	if this.isTruthy(expression) {
		return this.evaluate(expr.thenBranch)
	} else {
		return this.evaluate(expr.elseBranch)
	}
}

func (this *Interpreter) visitBinaryExpr(expr *Binary) (interface{}, error) {
	left, err := this.evaluate(expr.left)
	if err != nil {
		return nil, err
	}
	right, err := this.evaluate(expr.right)
	if err != nil {
		return nil, err
	}

	switch expr.operator.Type {
	case GREATER:
		if err := this.checkNumberOperands(expr.operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) > right.(float64), nil
	case GREATER_EQUAL:
		if err := this.checkNumberOperands(expr.operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) >= right.(float64), nil
	case LESS:
		if err := this.checkNumberOperands(expr.operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) < right.(float64), nil
	case LESS_EQUAL:
		if err := this.checkNumberOperands(expr.operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) <= right.(float64), nil
	case MINUS:
		if err := this.checkNumberOperands(expr.operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) - right.(float64), nil
	case BANG_EQUAL:
		return !this.isEqual(left, right), nil
	case EQUAL_EQUAL:
		return this.isEqual(left, right), nil
	case PLUS:
		v1, ok1 := left.(float64)
		v2, ok2 := right.(float64)
		if ok1 && ok2 {
			return v1 + v2, nil
		}

		s1, ok1 := left.(string)
		s2, ok2 := right.(string)
		if ok1 && ok2 {
			return s1 + s2, nil
		}
		return nil, NewRuntimeError(expr.operator, "operands must be two numbers or two strings.")
	case SLASH:
		if err := this.checkNumberOperands(expr.operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) / right.(float64), nil
	case STAR:
		if err := this.checkNumberOperands(expr.operator, left, right); err != nil {
			return nil, err
		}
		return left.(float64) * right.(float64), nil
	case COMMA:
		return right, nil
	}
	return nil, nil
}

func (this *Interpreter) visitCallExpr(expr *Call) (interface{}, error) {
	callee, err := this.evaluate(expr.callee)
	if err != nil {
		return nil, err
	}

	var arguments []interface{}
	for _, argument := range expr.arguments {
		arg, err := this.evaluate(argument)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, arg)
	}
	function, ok := callee.(LoxCallable)
	if !ok {
		return nil, NewRuntimeError(expr.paren, "can only call functions and classes.")
	}

	if len(arguments) != function.Arity() {
		return nil, NewRuntimeError(expr.paren, "expected "+strconv.Itoa(function.Arity())+
			" arguments but got "+strconv.Itoa(len(arguments)))
	}

	return function.Call(this, arguments)
}

func (this *Interpreter) visitGetExpr(expr *Get) (interface{}, error) {
	object, err := this.evaluate(expr.object)
	if err != nil {
		return nil, err
	}
	if instance, ok := object.(*LoxInstance); ok && instance != nil {
		return instance.Get(expr.name), nil
	}
	return nil, NewRuntimeError(expr.name, "only instances have properties.")
}

func (this *Interpreter) visitIndexExpr(expr *Index) (interface{}, error) {
	left, err := this.evaluate(expr.left)
	if err != nil {
		return nil, err
	}
	array, ok := left.(LoxIterator)
	if !ok {
		return nil, NewRuntimeError(expr.name, "can only iterator can be subsetted.")
	}
	index, err := this.evaluate(expr.index)
	if err != nil {
		return nil, err
	}
	idx, ok := index.(float64)
	if !ok {
		return nil, NewRuntimeError(expr.name, "expected valid express after '['")
	}
	v, err := array.Get(int(idx))
	if err != nil {
		return nil, NewRuntimeError(expr.name, err.Error())
	}
	return v, nil
}

func (this *Interpreter) visitExpressionStmt(stmt *Expression) (interface{}, error) {
	_, err := this.evaluate(stmt.expression)
	return nil, err
}

func (this *Interpreter) visitFunctionStmt(stmt *Function) (interface{}, error) {
	function := NewLoxFunction(stmt, this.environment, false)
	this.environment.Define(stmt.name.Lexeme, function)
	return nil, nil
}

func (this *Interpreter) visitLambdaExpr(expr *Lambda) (interface{}, error) {
	return NewLoxLambda(expr, this.environment, false), nil
}

func (this *Interpreter) visitArrayLiteralExpr(expr *ArrayLiteral) (interface{}, error) {
	var items []interface{}
	for _, express := range expr.items {
		v, err := this.evaluate(express)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return NewLoxArray(items), nil
}

func (this *Interpreter) visitIfStmt(stmt *If) (interface{}, error) {
	cond, err := this.evaluate(stmt.condition)
	if err != nil {
		return nil, err
	}
	if this.isTruthy(cond) {
		return nil, this.execute(stmt.thenBranch)
	} else if stmt.elseBranch != nil {
		return nil, this.execute(stmt.elseBranch)
	}
	return nil, nil
}

func (this *Interpreter) visitReturnStmt(stmt *Return) (interface{}, error) {
	var value interface{} = nil
	var err error
	if stmt.value != nil {
		value, err = this.evaluate(stmt.value)
		if err != nil {
			return nil, err
		}
	}
	return nil, NewReturnFlow(value)
}

func (this *Interpreter) visitPrintStmt(stmt *Print) (interface{}, error) {
	value, err := this.evaluate(stmt.expression)
	if err != nil {
		return nil, err
	}
	fmt.Println(this.stringify(value))
	return nil, nil
}

func (this *Interpreter) visitVarStmt(stmt *Var) (interface{}, error) {
	var value interface{}
	var err error
	if stmt.initializer != nil {
		value, err = this.evaluate(stmt.initializer)
		if err != nil {
			return nil, err
		}
	}
	this.environment.Define(stmt.name.Lexeme, value)
	return nil, nil
}

func (this *Interpreter) visitWhileStmt(stmt *While) (interface{}, error) {
	for {
		cond, err := this.evaluate(stmt.condition)
		if err != nil {
			return nil, err
		}
		if !this.isTruthy(cond) {
			break
		}
		err = this.execute(stmt.body)
		if err != nil {
			if cf, ok := IsControlFlow(err); ok {
				if cf.IsBreak() {
					break
				}
				if cf.IsContinue() {
					continue
				}
				return nil, err
			}
			return nil, err
		}
	}
	return nil, nil
}

func (this *Interpreter) visitBreakStmt(stmt *Break) (interface{}, error) {
	return nil, NewBreakFlow()
}

func (this *Interpreter) visitContinueStmt(stmt *Continue) (interface{}, error) {
	return nil, NewContinueFlow()
}

func (this *Interpreter) visitAssignExpr(expr *Assign) (interface{}, error) {
	value, err := this.evaluate(expr.value)
	if err != nil {
		return nil, err
	}

	distance, ok := this.locals[expr]
	if ok {
		this.environment.AssignAt(distance, expr.name, value)
	} else {
		globals.Assign(expr.name, value)
	}
	return value, nil
}

func (this *Interpreter) visitArraySetExpr(expr *ArraySet) (interface{}, error) {
	val, err := this.evaluate(expr.left)
	if err != nil {
		return nil, err
	}
	array, ok := val.(LoxIterator)
	if !ok {
		return nil, NewRuntimeError(expr.name, "can only set array.")
	}
	if expr.index == nil {
		v, err := this.evaluate(expr.value)
		if err != nil {
			return nil, err
		}
		array.Add(v)
	} else {
		idxVal, err := this.evaluate(expr.index)
		if err != nil {
			return nil, err
		}
		index := idxVal.(float64)
		v, err := this.evaluate(expr.value)
		if err != nil {
			return nil, err
		}
		if err := array.Set(int(index), v); err != nil {
			return nil, NewRuntimeError(expr.name, err.Error())
		}
	}
	return nil, nil
}

func (this *Interpreter) evaluate(expr Expr) (interface{}, error) {
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

func (this *Interpreter) checkNumberOperand(operator *Token, operand interface{}) error {
	if _, ok := operand.(float64); ok {
		return nil
	}
	return NewRuntimeError(operator, "operand must be a number.")
}

func (this *Interpreter) checkNumberOperands(operator *Token, left, right interface{}) error {
	_, ok1 := left.(float64)
	_, ok2 := right.(float64)
	if ok1 && ok2 {
		return nil
	}
	return NewRuntimeError(operator, "operands must be numbers.")
}

func (this *Interpreter) checkVariable(operator *Token, right interface{}, message string) error {
	if _, ok := right.(*Variable); ok {
		return nil
	}
	return NewRuntimeError(operator, message)
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
