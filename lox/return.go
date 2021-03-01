package lox

type returnExp struct {
	runtimeError
	value interface{}
}

func NewReturnExp(value interface{}) *returnExp {
	return &returnExp{runtimeError: runtimeError{token: nil, message: ""}, value: value}
}
