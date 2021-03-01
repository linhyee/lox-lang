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
	closure       *Environment //闭包环境
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

func (this *LoxFunction) Call(interpreter *Interpreter, arguments []interface{}) (value interface{}) {
	env := NewEnvironment(this.closure) //parent 指向创造函数的环境
	for i := 0; i < len(this.declaration.params); i++ {
		env.Define(this.declaration.params[i].Lexeme, arguments[i])
	}
	defer func(this *LoxFunction) {
		if r := recover(); r != nil {
			if v, ok := r.(*returnExp); ok && v != nil {
				value = v.value
				if this.isInitializer {
					value = this.closure.GetAt(0, "this")
				}
			} else {
				panic(r) //往上传播恐慌
			}
		}
		if this.isInitializer {
			value = this.closure.GetAt(0, "this")
		}
	}(this)
	interpreter.executeBlock(this.declaration.body, env)
	return
}

func (this LoxFunction) String() string {
	if this.declaration.name != nil {
		return "<fn " + this.declaration.name.Lexeme + ">"
	}
	return "<fn closure>"
}
