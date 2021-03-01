package lox

import (
	"bytes"
	"fmt"
)

type Parser struct {
	tokens  []*Token
	current int
	loop    int
}

func NewParser(tokens []*Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
	}
}

func (this *Parser) Parse() []Stmt {
	var statements []Stmt
	for !this.isAtEnd() {
		statements = append(statements, this.declaration()...)
	}
	return statements
}

// declaration 解析文法`声明语句`, var a ='2';
func (this *Parser) declaration() (stmts []Stmt) {
	defer func(parser *Parser) {
		if pe, ok := recover().(parserError); ok {
			_ = pe.Error()
			parser.synchronize()
			stmts = nil
		}
	}(this)
	if this.match(CLASS) {
		stmts = append(stmts, this.classDeclaration())
		return
	}
	if this.match(FUN) {
		stmts = append(stmts, this.function("function"))
		return
	}
	if this.match(VAR) { //匹配`var`关键字
		return this.varDeclarations() //解析声明语句
	}
	// 如果不是声明语句,则是普通语句
	stmts = append(stmts, this.statement())
	return
}

// classDeclaration 解析class声明
func (this *Parser) classDeclaration() Stmt {
	name := this.consume(IDENTIFIER, "expect class name")

	var superclass *Variable = nil
	if this.match(LESS) {
		this.consume(IDENTIFIER, "expect superclass name.")
		superclass = NewVariable(this.previous())
	}

	this.consume(LEFT_BRACE, "expect '{' before class body.")

	var methods []*Function
	for !this.check(RIGHT_BRACE) && !this.isAtEnd() {
		methods = append(methods, this.function("method"))
	}

	this.consume(RIGHT_BRACE, "expect '}' after class body.")
	return NewClass(name, superclass, methods)
}

func (this *Parser) varDeclarations() (stmts []Stmt) {
	stmts = append(stmts, this.varDeclaration(false))
	for this.match(COMMA) {
		stmts = append(stmts, this.varDeclaration(false))
	}
	this.consume(SEMICOLON, "expect ';' after variable declaration.")
	return
}

func (this *Parser) varDeclaration(consume bool) Stmt {
	name := this.consume(IDENTIFIER, "expect variable name.")
	var initializer Expr
	if this.match(EQUAL) { //带初始化表达式
		initializer = this.assignment()
	}
	if consume {
		this.consume(SEMICOLON, "expect ';' after variable declaration.")
	}
	return NewVar(name, initializer)
}

// whileStatement 解析while语句
func (this *Parser) whileStatement() Stmt {
	this.consume(LEFT_PAREN, "expect '(' after 'while'.")
	condition := this.expression()
	this.consume(RIGHT_PAREN, "expect ')' after condition.")

	this.loop++
	defer func(this *Parser) {
		this.loop--
	}(this)
	body := this.statement()

	return NewWhile(condition, body)
}

// breakStatement 解析break语句
func (this *Parser) breakStatement() Stmt {
	if !(this.loop > 0) {
		panic(NewParseError(this.previous(), "break statement must be inside a loop."))
	}
	this.consume(SEMICOLON, "expect ';' after 'break' statement")
	return NewBreak()
}

// continueStatement 解析continue语句
func (this *Parser) continueStatement() Stmt {
	if !(this.loop > 0) {
		panic(NewParseError(this.previous(), "Continue statement must be inside a loop."))
	}
	this.consume(SEMICOLON, "expect ';' after 'continue' statement.")
	return NewContinue()
}

// statement 解析文法`语句`
func (this *Parser) statement() Stmt {
	if this.match(FOR) {
		return this.forStatement()
	}
	if this.match(IF) {
		return this.ifStatement()
	}
	if this.match(PRINT) {
		return this.printStatement()
	}
	if this.match(RETURN) {
		return this.returnStatement()
	}
	if this.match(WHILE) {
		return this.whileStatement()
	}
	if this.match(BREAK) {
		return this.breakStatement()
	}
	if this.match(CONTINUE) {
		return this.continueStatement()
	}
	if this.match(LEFT_BRACE) {
		return NewBlock(this.block())
	}
	return this.expressionStatement()
}

// forStatement 解析for语句
func (this *Parser) forStatement() Stmt {
	this.consume(LEFT_PAREN, "expect '(' after 'for'.")

	//解析初始化表达式
	var initializer Stmt = nil
	if this.match(SEMICOLON) {
		initializer = nil
	} else if this.match(VAR) {
		initializer = this.varDeclaration(true)
	} else {
		initializer = this.expressionStatement()
	}

	//解析条件
	var condition Expr = nil
	if !this.check(SEMICOLON) {
		condition = this.expression()
	}
	this.consume(SEMICOLON, "expect ';' after loop condition.")

	//解析步长
	var increment Expr = nil
	if !this.check(RIGHT_PAREN) {
		increment = this.expression()
	}
	this.consume(RIGHT_PAREN, "expect ')' after for clauses.")

	this.loop++
	defer func(this *Parser) {
		this.loop--
	}(this)

	body := this.statement()

	if increment != nil {
		body = NewBlock([]Stmt{body, NewExpression(increment)})
	}

	if condition == nil {
		condition = NewLiteral(true)
	}
	body = NewWhile(condition, body)

	if initializer != nil {
		body = NewBlock([]Stmt{initializer, body})
	}
	return body
}

// ifStatement 解析if语句
func (this *Parser) ifStatement() Stmt {
	this.consume(LEFT_PAREN, "expect '(' after 'if'.")
	condition := this.expression()
	this.consume(RIGHT_PAREN, "expect ')' after if condition.")

	thenBranch := this.statement()
	var elseBranch Stmt = nil
	if this.match(ELSE) {
		elseBranch = this.statement()
	}
	return NewIf(condition, thenBranch, elseBranch)
}

// printStatement 解析print语名
func (this *Parser) printStatement() Stmt {
	value := this.expression()
	this.consume(SEMICOLON, "expect ';' after value.") //printExpr 语句必须以分号`;`结束
	return NewPrint(value)
}

// returnStatement 解析return 语句
func (this *Parser) returnStatement() Stmt {
	keyword := this.previous()
	var value Expr = nil
	if !this.check(SEMICOLON) {
		value = this.expression()
	}
	this.consume(SEMICOLON, "expect ';' after return value.")
	return NewReturn(keyword, value)
}

// expressionStatement 解析表达式语句
func (this *Parser) expressionStatement() Stmt {
	expr := this.expression()
	this.consume(SEMICOLON, "expect ';' after expression.")
	return NewExpression(expr)
}

// function 解析function声明
func (this *Parser) function(kind string) *Function {
	name := this.consume(IDENTIFIER, "expect "+kind+" name.")
	parameters, body := this.functionBody(kind)
	return NewFunction(name, parameters, body)
}

// lambda 解析lambda函数
func (this *Parser) lambda(kind string) *Lambda {
	parameters, body := this.functionBody(kind)
	return NewLambda(parameters, body)
}

func (this *Parser) functionBody(kind string) ([]*Token, []Stmt) {
	this.consume(LEFT_PAREN, "expect '(' after "+kind+" name.")
	var parameters []*Token
	if !this.check(RIGHT_PAREN) {
		for {
			if len(parameters) >= 255 {
				panic(NewParseError(this.peek(), "can't have more than 255 parameters."))
			}
			parameters = append(parameters, this.consume(IDENTIFIER, "Expect parameter name."))
			if !this.match(COMMA) {
				break
			}
		}
	}
	this.consume(RIGHT_PAREN, "expect ')' after parameters.")
	this.consume(LEFT_BRACE, "expect '{' before "+kind+" body.")
	body := this.block()
	return parameters, body
}

// block 解析{}语句块
func (this *Parser) block() []Stmt {
	var statements []Stmt
	for !this.check(RIGHT_BRACE) && !this.isAtEnd() {
		statements = append(statements, this.declaration()...)
	}
	this.consume(RIGHT_BRACE, "expect '}' after block.")
	return statements
}

// assignment 解析赋值语句
func (this *Parser) assignment() Expr {
	expr := this.ternary()
	if this.match(EQUAL) {
		equals := this.previous()
		value := this.assignment()

		if v, ok := expr.(*Variable); ok {
			name := v.name
			return NewAssign(name, value)
		} else if v, ok := expr.(*Index); ok {
			return NewArraySet(v.left, v.name, v.index, value)
		} else if v, ok := expr.(*Get); ok {
			return NewSet(v.object, v.name, value)
		}
		panic(NewParseError(equals, "invalid assignment target."))
	}
	return expr
}

// ternary 解析三元运算符 ? :
func (this *Parser) ternary() Expr {
	expr := this.or()
	if this.match(QUESTION_MARK) {
		thenBranch := this.expression()
		this.consume(COLON, "expect ':' after then branch of ternary.")
		elseBranch := this.ternary()
		expr = NewTernary(expr, thenBranch, elseBranch)
	}
	return expr
}

// or 解析or运算符
func (this *Parser) or() Expr {
	expr := this.and()

	for this.match(OR) {
		operator := this.previous()
		right := this.and()
		expr = NewLogical(expr, operator, right)
	}
	return expr
}

func (this *Parser) and() Expr {
	expr := this.equality()

	for this.match(AND) {
		operator := this.previous()
		right := this.equality()
		expr = NewLogical(expr, operator, right)
	}
	return expr
}

// expression 解析文法`运算表达式`,比如字面量,一元表达式,二元表达式,括号表达式
func (this *Parser) expression() Expr {
	return this.comma()
}

// expr 解析逗号运算符
func (this *Parser) comma() Expr {
	expr := this.assignment()
	for this.match(COMMA) {
		comma := this.previous()
		right := this.assignment()
		expr = NewBinary(expr, comma, right)
	}
	return expr
}

// equality 解析==,!=后面的的表达式
func (this *Parser) equality() Expr {
	var expr = this.comparison()
	for this.match(BANG_EQUAL, EQUAL_EQUAL) {
		operator := this.previous()
		right := this.comparison()
		expr = NewBinary(expr, operator, right)
	}
	return expr
}

// match 匹配token, 如果匹配成功,游标current向前走一步
func (this *Parser) match(types ...TokenType) bool {
	for _, ty := range types {
		if this.check(ty) {
			this.advance()
			return true
		}
	}
	return false
}

func (this *Parser) matchNext(types ...TokenType) bool {
	for _, ty := range types {
		if this.checkNext(ty) {
			this.advance()
			return true
		}
	}
	return false
}

// check 检测当前游标current指向的token类型是否与ty一致
func (this *Parser) check(ty TokenType) bool {
	if this.isAtEnd() {
		return false
	}
	return this.peek().Type == ty
}

// checkNext 检测当前游标current的下一个token类型是否ty一致
func (this *Parser) checkNext(ty TokenType) bool {
	if this.isAtEnd() {
		return false
	}
	tk := this.tokens[this.current+1]
	if tk.Type == EOF {
		return false
	}
	return tk.Type == ty
}

// advance 游标current往前走一步, 并返回当前token
func (this *Parser) advance() *Token {
	if !this.isAtEnd() {
		this.current++
	}
	return this.previous()
}

// isAtEnd 是否到了Token序列的末端
func (this *Parser) isAtEnd() bool {
	return this.peek().Type == EOF
}

// peek 获取当前游标current指向的token
func (this *Parser) peek() *Token {
	return this.tokens[this.current]
}

// previous 返回当前游标current的前一个token
func (this *Parser) previous() *Token {
	return this.tokens[this.current-1]
}

// comparison 解析比较运算符
func (this *Parser) comparison() Expr {
	var expr = this.term()
	for this.match(GREATER, GREATER_EQUAL, LESS, LESS_EQUAL) {
		operator := this.previous()
		right := this.term()
		expr = NewBinary(expr, operator, right)
	}
	return expr
}

// term 解析加/减法运算符
func (this *Parser) term() Expr {
	var expr = this.factor()
	for this.match(MINUS, PLUS) {
		operator := this.previous()
		right := this.factor()
		expr = NewBinary(expr, operator, right)
	}
	return expr
}

// 解析乘/除法运算符
func (this *Parser) factor() Expr {
	var expr = this.unary()
	for this.match(SLASH, STAR) {
		operator := this.previous()
		right := this.unary()
		expr = NewBinary(expr, operator, right)
	}
	return expr
}

// unary 解析一元运算符
func (this *Parser) unary() Expr {
	if this.match(BANG, MINUS) {
		operator := this.previous()
		right := this.unary()
		return NewUnary(operator, right, false)
	}
	return this.prefix()
}

func (this *Parser) prefix() Expr {
	if this.match(PLUS_PLUS, MINUS_MINUS) {
		operator := this.previous()
		right := this.primary()
		return NewUnary(operator, right, false)
	}
	return this.postfix()
}

// postfix 解析后缀运算符
func (this *Parser) postfix() Expr {
	// 必须预匹配++或--, 因为,如a++, a会先被求值,而不会被相加后,存到变量中
	if this.matchNext(PLUS_PLUS, MINUS_MINUS) {
		operator := this.peek()
		this.current--
		left := this.primary()
		this.advance()
		return NewUnary(operator, left, true)
	}
	return this.call()
}

// call 解析函数调用
func (this *Parser) call() Expr {
	expr := this.primary()
	for true {
		if this.match(LEFT_PAREN) {
			expr = this.finishCall(expr)
		} else if this.match(DOT) {
			name := this.consume(IDENTIFIER, "expect property name after '.'.")
			expr = NewGet(expr, name)
		} else if this.match(LEFT_BRACKET) {
			bracket, index := this.previous(), this.assignment()
			if index == nil && this.checkNext(EQUAL) == false {
				panic(NewParseError(bracket, "expect expression after '['."))
			}
			this.consume(RIGHT_BRACKET, "expect ']' after index expression.")
			expr = NewIndex(expr, bracket, index)
		} else {
			break
		}
	}
	return expr
}

func (this *Parser) finishCall(callee Expr) Expr {
	var arguments []Expr
	if !this.check(RIGHT_PAREN) {
		for {
			if len(arguments) >= 255 {
				panic(NewParseError(this.peek(), "can't have more than 255 arguments."))
			}
			arguments = append(arguments, this.assignment())
			if !this.match(COMMA) {
				break
			}
		}
	}

	paren := this.consume(RIGHT_PAREN, "expect ')'  after arguments.")
	return NewCall(callee, paren, arguments)
}

// arrayLiteral 解析数组字面量
func (this *Parser) arrayLiteral() Expr {
	bracket := this.previous()
	var items []Expr
	for {
		items = append(items, this.assignment())
		if !this.match(COMMA) {
			break
		}
	}
	this.consume(RIGHT_BRACKET, "expected ']' after list items")
	return NewArrayLiteral(bracket, items)
}

// primary 解析字面量
func (this *Parser) primary() Expr {
	if this.match(FALSE) {
		return NewLiteral(false)
	}
	if this.match(TRUE) {
		return NewLiteral(true)
	}
	if this.match(NIL) {
		return NewLiteral(nil)
	}
	if this.match(NUMBER, STRING) {
		return NewLiteral(this.previous().Literal)
	}
	if this.match(SUPER) {
		keyword := this.previous()
		this.consume(DOT, "expect '.' after 'super'")
		method := this.consume(IDENTIFIER, "expect superclass method name.")
		return NewSuper(keyword, method)
	}
	if this.match(THIS) {
		return NewThis(this.previous())
	}
	if this.match(IDENTIFIER) {
		return NewVariable(this.previous())
	}
	if this.match(LEFT_PAREN) {
		expr := this.expression()
		this.consume(RIGHT_PAREN, "expect ')' after expression.")
		return NewGrouping(expr)
	}
	if this.match(LEFT_BRACKET) {
		return this.arrayLiteral()
	}
	if this.match(FUN) && !this.check(IDENTIFIER) {
		return this.lambda("function")
	}
	if this.check(RIGHT_BRACKET) && this.previous().Type == LEFT_BRACKET {
		return nil
	}

	// Error productions
	if this.match(QUESTION_MARK) {
		panic(NewParseError(this.previous(), "missing left-hand condition of ternary operator."))
	}
	if this.match(BANG_EQUAL, EQUAL_EQUAL) {
		panic(NewParseError(this.previous(), "missing left-hand operand."))
	}
	if this.match(GREATER, GREATER_EQUAL, LESS, LESS_EQUAL) {
		panic(NewParseError(this.previous(), "missing  left-hand operand."))
	}
	if this.match(SLASH, STAR) {
		panic(NewParseError(this.previous(), "missing left-hand operand."))
	}
	panic(NewParseError(this.peek(), "expect expression. `"+this.locate()+"`"))
}

// consume 消耗当前等于ty的token,并前进一步
func (this *Parser) consume(ty TokenType, message string) *Token {
	if this.check(ty) {
		return this.advance()
	}

	panic(NewParseError(this.peek(), message+" near `"+this.locate()+"`"))
}

func (this *Parser) locate() string {
	bs := bytes.NewBufferString("")
	for i, tk := range this.tokens[0:this.current] {
		bs.WriteString(tk.Lexeme)
		if i < this.current-1 {
			bs.WriteString(" ")
		}
		if tk.Literal != nil && !(tk.Type == STRING || tk.Type == NUMBER) {
			switch v := tk.Literal.(type) {
			case string:
				bs.WriteString("\"" + v + "\"")
			default:
				bs.WriteString(fmt.Sprintf("%v", v))
			}
		}
	}
	start := 0
	if bs.Len() > 20 {
		start = bs.Len() - 20
	}
	return bs.String()[start:bs.Len()]
}

func (this *Parser) synchronize() {
	this.advance()

	for !this.isAtEnd() {
		if this.previous().Type == SEMICOLON {
			return
		}
		switch this.peek().Type {
		case CLASS, FUN, VAR, FOR, IF, WHILE, PRINT, RETURN:
			return
		}

		this.advance()
	}
}
