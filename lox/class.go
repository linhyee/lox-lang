package lox

func NewLoxClass(name string, superclass *LoxClass, methods map[string]LoxCallable) *LoxClass {
	return &LoxClass{name: name, superclass: superclass, methods: methods}
}

type LoxClass struct {
	name       string
	methods    map[string]LoxCallable
	superclass *LoxClass
}

func (this *LoxClass) findMethod(name string) LoxCallable {
	if method, ok := this.methods[name]; ok && method != nil {
		return method
	}
	if this.superclass != nil {
		return this.superclass.findMethod(name)
	}
	return nil
}

func (this *LoxClass) Call(interpreter *Interpreter, arguments []interface{}) interface{} {
	instance := NewLoxInstance(this)
	if initializer := this.findMethod("init"); initializer != nil {
		initializer.(*LoxFunction).Bind(instance).Call(interpreter, arguments)
	}
	return instance
}

func (this *LoxClass) Arity() int {
	initializer := this.findMethod("init")
	if initializer == nil {
		return 0
	}
	return initializer.Arity()
}

func (this LoxClass) String() string {
	return this.name
}
