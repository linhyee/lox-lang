package lox

import "errors"

type ControlFlowKind int

const (
	CF_BREAK ControlFlowKind = iota
	CF_CONTINUE
	CF_RETURN
)

type ControlFlow struct {
	kind  ControlFlowKind
	value interface{}
}

func NewBreakFlow() *ControlFlow {
	return &ControlFlow{kind: CF_BREAK}
}

func NewContinueFlow() *ControlFlow {
	return &ControlFlow{kind: CF_CONTINUE}
}

func NewReturnFlow(value interface{}) *ControlFlow {
	return &ControlFlow{kind: CF_RETURN, value: value}
}

func (cf *ControlFlow) Error() string {
	switch cf.kind {
	case CF_BREAK:
		return "break"
	case CF_CONTINUE:
		return "continue"
	case CF_RETURN:
		return "return"
	default:
		return "unknown control flow"
	}
}

func (cf *ControlFlow) IsBreak() bool {
	return cf.kind == CF_BREAK
}

func (cf *ControlFlow) IsContinue() bool {
	return cf.kind == CF_CONTINUE
}

func (cf *ControlFlow) IsReturn() bool {
	return cf.kind == CF_RETURN
}

func (cf *ControlFlow) Value() interface{} {
	return cf.value
}

func IsControlFlow(err error) (*ControlFlow, bool) {
	var cf *ControlFlow
	if errors.As(err, &cf) {
		return cf, true
	}
	return nil, false
}
