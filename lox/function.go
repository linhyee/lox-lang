package lox

func NewLoxFunction(decl *Function, closure *Environment, isInitializer bool) LoxCallable {
	return &LoxFunction{declaration: decl, closure: closure, isInitializer: isInitializer}
}

func NewLoxLambda(lambda *Lambda, closure *Environment, isInitializer bool) LoxCallable {
	return &LoxFunction{
		declaration:   NewFunction(nil, lambda.params, lambda.body),
		closure:       closure,
		isInitializer: isInitializer,
	}
}

type LoxFunction struct {
	declaration   *Function
	closure       *Environment
	isInitializer bool
}

func (this *LoxFunction) Bind(instance *LoxInstance) LoxCallable {
	environment := NewEnvironment(this.closure)
	environment.Define("this", instance)
	return NewLoxFunction(this.declaration, environment, this.isInitializer)
}

func (this *LoxFunction) Arity() int {
	return len(this.declaration.params)
}

func (this *LoxFunction) Call(interpreter *Interpreter, arguments []interface{}) (interface{}, error) {
	env := NewEnvironment(this.closure)
	for i := 0; i < len(this.declaration.params); i++ {
		env.Define(this.declaration.params[i].Lexeme, arguments[i])
	}
	err := interpreter.executeBlock(this.declaration.body, env)
	if err != nil {
		if cf, ok := IsControlFlow(err); ok && cf.IsReturn() {
			value := cf.Value()
			if this.isInitializer {
				return this.closure.GetAt(0, "this"), nil
			}
			return value, nil
		}
		return nil, err
	}
	if this.isInitializer {
		return this.closure.GetAt(0, "this"), nil
	}
	return nil, nil
}

func (this LoxFunction) String() string {
	if this.declaration.name != nil {
		return "<fn " + this.declaration.name.Lexeme + ">"
	}
	return "<fn closure>"
}
