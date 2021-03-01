package lox

type Expr interface {
	accept(visitor VisitorExpr) interface{}
}

type VisitorExpr interface {
	visitAssignExpr(expr *Assign) interface{}
	visitBinaryExpr(expr *Binary) interface{}
	visitLambdaExpr(expr *Lambda) interface{}
	visitCallExpr(expr *Call) interface{}
	visitGetExpr(expr *Get) interface{}
	visitIndexExpr(expr *Index) interface{}
	visitArrayLiteralExpr(expr *ArrayLiteral) interface{}
	visitGroupingExpr(expr *Grouping) interface{}
	visitLiteralExpr(expr *Literal) interface{}
	visitLogicalExpr(expr *Logical) interface{}
	visitSetExpr(expr *Set) interface{}
	visitSuperExpr(expr *Super) interface{}
	visitThisExpr(expr *This) interface{}
	visitArraySetExpr(expr *ArraySet) interface{}
	visitUnaryExpr(expr *Unary) interface{}
	visitTernaryExpr(expr *Ternary) interface{}
	visitVariableExpr(expr *Variable) interface{}
}

func NewAssign(name *Token, value Expr) *Assign {
	return &Assign{
		name : name,
		value : value,
	}
}

type Assign struct {
	name *Token
	value Expr
}

func (this *Assign) accept(visitor VisitorExpr) interface{} {
	return visitor.visitAssignExpr(this)
}

func NewBinary(left Expr, operator *Token, right Expr) *Binary {
	return &Binary{
		left : left,
		operator : operator,
		right : right,
	}
}

type Binary struct {
	left Expr
	operator *Token
	right Expr
}

func (this *Binary) accept(visitor VisitorExpr) interface{} {
	return visitor.visitBinaryExpr(this)
}

func NewLambda(params []*Token, body []Stmt) *Lambda {
	return &Lambda{
		params : params,
		body : body,
	}
}

type Lambda struct {
	params []*Token
	body []Stmt
}

func (this *Lambda) accept(visitor VisitorExpr) interface{} {
	return visitor.visitLambdaExpr(this)
}

func NewCall(callee Expr, paren *Token, arguments []Expr) *Call {
	return &Call{
		callee : callee,
		paren : paren,
		arguments : arguments,
	}
}

type Call struct {
	callee Expr
	paren *Token
	arguments []Expr
}

func (this *Call) accept(visitor VisitorExpr) interface{} {
	return visitor.visitCallExpr(this)
}

func NewGet(object Expr, name *Token) *Get {
	return &Get{
		object : object,
		name : name,
	}
}

type Get struct {
	object Expr
	name *Token
}

func (this *Get) accept(visitor VisitorExpr) interface{} {
	return visitor.visitGetExpr(this)
}

func NewIndex(left Expr, name *Token, index Expr) *Index {
	return &Index{
		left : left,
		name : name,
		index : index,
	}
}

type Index struct {
	left Expr
	name *Token
	index Expr
}

func (this *Index) accept(visitor VisitorExpr) interface{} {
	return visitor.visitIndexExpr(this)
}

func NewArrayLiteral(bracket *Token, items []Expr) *ArrayLiteral {
	return &ArrayLiteral{
		bracket : bracket,
		items : items,
	}
}

type ArrayLiteral struct {
	bracket *Token
	items []Expr
}

func (this *ArrayLiteral) accept(visitor VisitorExpr) interface{} {
	return visitor.visitArrayLiteralExpr(this)
}

func NewGrouping(expression Expr) *Grouping {
	return &Grouping{
		expression : expression,
	}
}

type Grouping struct {
	expression Expr
}

func (this *Grouping) accept(visitor VisitorExpr) interface{} {
	return visitor.visitGroupingExpr(this)
}

func NewLiteral(value interface{}) *Literal {
	return &Literal{
		value : value,
	}
}

type Literal struct {
	value interface{}
}

func (this *Literal) accept(visitor VisitorExpr) interface{} {
	return visitor.visitLiteralExpr(this)
}

func NewLogical(left Expr, operator *Token, right Expr) *Logical {
	return &Logical{
		left : left,
		operator : operator,
		right : right,
	}
}

type Logical struct {
	left Expr
	operator *Token
	right Expr
}

func (this *Logical) accept(visitor VisitorExpr) interface{} {
	return visitor.visitLogicalExpr(this)
}

func NewSet(object Expr, name *Token, value Expr) *Set {
	return &Set{
		object : object,
		name : name,
		value : value,
	}
}

type Set struct {
	object Expr
	name *Token
	value Expr
}

func (this *Set) accept(visitor VisitorExpr) interface{} {
	return visitor.visitSetExpr(this)
}

func NewSuper(keyword *Token, method *Token) *Super {
	return &Super{
		keyword : keyword,
		method : method,
	}
}

type Super struct {
	keyword *Token
	method *Token
}

func (this *Super) accept(visitor VisitorExpr) interface{} {
	return visitor.visitSuperExpr(this)
}

func NewThis(keyword *Token) *This {
	return &This{
		keyword : keyword,
	}
}

type This struct {
	keyword *Token
}

func (this *This) accept(visitor VisitorExpr) interface{} {
	return visitor.visitThisExpr(this)
}

func NewArraySet(left Expr, name *Token, index Expr, value Expr) *ArraySet {
	return &ArraySet{
		left : left,
		name : name,
		index : index,
		value : value,
	}
}

type ArraySet struct {
	left Expr
	name *Token
	index Expr
	value Expr
}

func (this *ArraySet) accept(visitor VisitorExpr) interface{} {
	return visitor.visitArraySetExpr(this)
}

func NewUnary(operator *Token, right Expr, postfix bool) *Unary {
	return &Unary{
		operator : operator,
		right : right,
		postfix : postfix,
	}
}

type Unary struct {
	operator *Token
	right Expr
	postfix bool
}

func (this *Unary) accept(visitor VisitorExpr) interface{} {
	return visitor.visitUnaryExpr(this)
}

func NewTernary(expr Expr, thenBranch Expr, elseBranch Expr) *Ternary {
	return &Ternary{
		expr : expr,
		thenBranch : thenBranch,
		elseBranch : elseBranch,
	}
}

type Ternary struct {
	expr Expr
	thenBranch Expr
	elseBranch Expr
}

func (this *Ternary) accept(visitor VisitorExpr) interface{} {
	return visitor.visitTernaryExpr(this)
}

func NewVariable(name *Token) *Variable {
	return &Variable{
		name : name,
	}
}

type Variable struct {
	name *Token
}

func (this *Variable) accept(visitor VisitorExpr) interface{} {
	return visitor.visitVariableExpr(this)
}

