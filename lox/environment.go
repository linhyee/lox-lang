package lox

type Environment struct {
	enclosing *Environment           //作用域
	values    map[string]interface{} //变量存储空间
}

func NewEnvironment(env *Environment) *Environment {
	return &Environment{enclosing: env, values: make(map[string]interface{})}
}

// Define puts value into map
func (this *Environment) Define(name string, value interface{}) {
	this.values[name] = value
}

// Get get value from map
func (this *Environment) Get(name *Token) interface{} {
	if v, ok := this.values[name.Lexeme]; ok {
		return v
	}
	if this.enclosing != nil {
		return this.enclosing.Get(name)
	}
	panic(NewRuntimeError(name, "get undefined variable '"+name.Lexeme+"'."))
}

// GetAt
func (this *Environment) GetAt(distance int, name string) interface{} {
	return this.ancestor(distance).values[name]
}

// ancestor
func (this *Environment) ancestor(distance int) *Environment {
	environment := this
	for i := 0; i < distance; i++ {
		environment = environment.enclosing
	}
	return environment
}

// Assign assign new value to variable
func (this *Environment) Assign(name *Token, value interface{}) {
	if _, ok := this.values[name.Lexeme]; ok {
		this.values[name.Lexeme] = value
		return
	}
	if this.enclosing != nil {
		this.enclosing.Assign(name, value)
		return
	}
	panic(NewRuntimeError(name, "set undefined variable '"+name.Lexeme+"'."))
}

// AssignAt
func (this *Environment) AssignAt(distance int, name *Token, value interface{}) {
	this.ancestor(distance).values[name.Lexeme] = value
}
