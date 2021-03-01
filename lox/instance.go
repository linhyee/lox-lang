package lox

func NewLoxInstance(class *LoxClass) *LoxInstance {
	return &LoxInstance{class: class, fields: map[string]interface{}{}}
}

type LoxInstance struct {
	class  *LoxClass
	fields map[string]interface{}
}

func (this *LoxInstance) Get(name *Token) interface{} {
	if value, ok := this.fields[name.Lexeme]; ok {
		return value
	}
	if method := this.class.findMethod(name.Lexeme); method != nil {
		return method.(*LoxFunction).Bind(this)
	}
	panic(NewRuntimeError(name, "undefined property '"+name.Lexeme+"'."))
}

func (this *LoxInstance) Set(name *Token, value interface{}) {
	this.fields[name.Lexeme] = value
}

func (this LoxInstance) String() string {
	return this.class.name + " instance"
}
