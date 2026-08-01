package glox

func (p *Parser) expression() {
	p.parsePrecedence(PrecAssignment)
}

func (p *Parser) parsePrecedence(precedence Precedence) {
	p.advance()
	prefix := rules[p.previous.Type].prefix
	if prefix == nil {
		p.error("expect expression")
		return
	}
	canAssign := precedence <= PrecAssignment
	prefix(p, canAssign)

	for precedence <= rules[p.current.Type].precedence {
		p.advance()
		infix := rules[p.previous.Type].infix
		infix(p, canAssign)
	}

	if canAssign && p.match(TokenEqual) {
		p.error("invalid assignment target")
	}
}

func grouping(p *Parser, canAssign bool) {
	p.expression()
	p.consume(TokenRightParen, "expect ')' after expression")
}

func number(p *Parser, canAssign bool) {
	p.emitConstant(p.previous.Literal)
}

func string_(p *Parser, canAssign bool) {
	p.emitConstant(p.previous.Literal)
}

func literal(p *Parser, canAssign bool) {
	switch p.previous.Type {
	case TokenFalse:
		p.emitOp(OpFalse)
	case TokenTrue:
		p.emitOp(OpTrue)
	case TokenNil:
		p.emitOp(OpNil)
	}
}

func variable(p *Parser, canAssign bool) {
	p.namedVariable(p.previous, canAssign)
}

func this_(p *Parser, canAssign bool) {
	if p.class == nil {
		p.error("can't use 'this' outside of a class")
		return
	}
	p.namedVariable(p.previous, false)
}

func super_(p *Parser, canAssign bool) {
	if p.class == nil {
		p.error("can't use 'super' outside of a class")
	} else if !p.class.HasSuperclass {
		p.error("can't use 'super' in a class with no superclass")
	}
	p.consume(TokenDot, "expect '.' after 'super'")
	p.consume(TokenIdentifier, "expect superclass method name")
	method := p.identifierConstant(p.previous.Lexeme)
	thisToken := Token{Type: TokenIdentifier, Lexeme: "this", Line: p.previous.Line}
	superToken := Token{Type: TokenIdentifier, Lexeme: "super", Line: p.previous.Line}
	p.namedVariable(thisToken, false)
	if p.match(TokenLeftParen) {
		argCount := p.argumentList()
		p.namedVariable(superToken, false)
		p.emitBytes(byte(OpSuperInvoke), method)
		p.emitByte(argCount)
		return
	}
	p.namedVariable(superToken, false)
	p.emitBytes(byte(OpGetSuper), method)
}

func unary(p *Parser, canAssign bool) {
	operatorType := p.previous.Type
	if operatorType == TokenPlusPlus || operatorType == TokenMinusMinus {
		p.consume(TokenIdentifier, "prefix increment target must be a variable")
		name := p.previous
		getOp := OpGetGlobal
		setOp := OpSetGlobal
		arg := p.identifierConstant(name.Lexeme)
		local := p.resolveLocal(p.compiler, name.Lexeme)
		upvalue := -1
		if local != -1 {
			getOp, setOp, arg = OpGetLocal, OpSetLocal, byte(local)
		} else if upvalue = p.resolveUpvalue(p.compiler, name.Lexeme); upvalue != -1 {
			getOp, setOp, arg = OpGetUpvalue, OpSetUpvalue, byte(upvalue)
		}
		if p.variableIsConst(name, local, upvalue) {
			p.error("cannot assign to const '" + name.Lexeme + "'")
		}
		p.emitBytes(byte(getOp), arg)
		p.emitConstant(float64(1))
		if operatorType == TokenPlusPlus {
			p.emitOp(OpAdd)
		} else {
			p.emitOp(OpSubtract)
		}
		p.emitBytes(byte(setOp), arg)
		return
	}

	p.parsePrecedence(PrecUnary)
	switch operatorType {
	case TokenBang:
		p.emitOp(OpNot)
	case TokenMinus:
		p.emitOp(OpNegate)
	}
}

func binary(p *Parser, canAssign bool) {
	operatorType := p.previous.Type
	rule := rules[operatorType]
	p.parsePrecedence(rule.precedence + 1)
	switch operatorType {
	case TokenBangEqual:
		p.emitOp(OpEqual)
		p.emitOp(OpNot)
	case TokenEqualEqual:
		p.emitOp(OpEqual)
	case TokenGreater:
		p.emitOp(OpGreater)
	case TokenGreaterEqual:
		p.emitOp(OpLess)
		p.emitOp(OpNot)
	case TokenLess:
		p.emitOp(OpLess)
	case TokenLessEqual:
		p.emitOp(OpGreater)
		p.emitOp(OpNot)
	case TokenPlus:
		p.emitOp(OpAdd)
	case TokenMinus:
		p.emitOp(OpSubtract)
	case TokenStar:
		p.emitOp(OpMultiply)
	case TokenSlash:
		p.emitOp(OpDivide)
	}
}

func and_(p *Parser, canAssign bool) {
	endJump := p.emitJump(OpJumpIfFalse)
	p.emitOp(OpPop)
	p.parsePrecedence(PrecAnd)
	p.patchJump(endJump)
}

func or_(p *Parser, canAssign bool) {
	elseJump := p.emitJump(OpJumpIfFalse)
	endJump := p.emitJump(OpJump)
	p.patchJump(elseJump)
	p.emitOp(OpPop)
	p.parsePrecedence(PrecOr)
	p.patchJump(endJump)
}

func ternary(p *Parser, canAssign bool) {
	thenJump := p.emitJump(OpJumpIfFalse)
	p.emitOp(OpPop)
	p.expression()
	elseJump := p.emitJump(OpJump)
	p.consume(TokenColon, "expect ':' after then branch")
	p.patchJump(thenJump)
	p.emitOp(OpPop)
	p.parsePrecedence(PrecTernary)
	p.patchJump(elseJump)
}

func call(p *Parser, canAssign bool) {
	argCount := p.argumentList()
	p.emitBytes(byte(OpCall), argCount)
}

func (p *Parser) argumentList() byte {
	argCount := 0
	if !p.check(TokenRightParen) {
		for {
			p.expression()
			if argCount == 255 {
				p.error("can't have more than 255 arguments")
			}
			argCount++
			if !p.match(TokenComma) {
				break
			}
		}
	}
	p.consume(TokenRightParen, "expect ')' after arguments")
	return byte(argCount)
}

func dot(p *Parser, canAssign bool) {
	p.consume(TokenIdentifier, "expect property name after '.'")
	name := p.identifierConstant(p.previous.Lexeme)
	if canAssign && p.match(TokenEqual) {
		p.expression()
		p.emitBytes(byte(OpSetProperty), name)
	} else if p.match(TokenLeftParen) {
		argCount := p.argumentList()
		p.emitBytes(byte(OpInvoke), name)
		p.emitByte(argCount)
	} else {
		p.emitBytes(byte(OpGetProperty), name)
	}
}

func subscript(p *Parser, canAssign bool) {
	p.expression()
	p.consume(TokenRightBracket, "expect ']' after index")
	if canAssign && p.match(TokenEqual) {
		p.expression()
		p.emitOp(OpSetIndex)
	} else {
		p.emitOp(OpGetIndex)
	}
}

func list(p *Parser, canAssign bool) {
	count := 0
	if !p.check(TokenRightBracket) {
		for {
			p.expression()
			count++
			if count > 255 {
				p.error("list literal can't have more than 255 elements")
			}
			if !p.match(TokenComma) {
				break
			}
			if p.check(TokenRightBracket) {
				break
			}
		}
	}
	p.consume(TokenRightBracket, "expect ']' after list elements")
	p.emitBytes(byte(OpList), byte(count))
}

func map_(p *Parser, canAssign bool) {
	p.emitOp(OpMap)
	if !p.check(TokenRightBrace) {
		for {
			switch {
			case p.match(TokenIdentifier):
				p.emitConstant(p.previous.Lexeme)
			case p.match(TokenString):
				p.emitConstant(p.previous.Literal)
			case p.match(TokenLeftBracket):
				p.expression()
				p.consume(TokenRightBracket, "expect ']' after computed map key")
			default:
				p.error("expect map key")
			}
			p.consume(TokenColon, "expect ':' after map key")
			p.expression()
			p.emitOp(OpMapSet)
			if !p.match(TokenComma) {
				break
			}
			if p.check(TokenRightBrace) {
				break
			}
		}
	}
	p.consume(TokenRightBrace, "expect '}' after map literal")
}

func lambda(p *Parser, canAssign bool) {
	if p.check(TokenIdentifier) {
		p.error("function declarations are only allowed as statements")
		return
	}
	p.function(FunctionLambda, "lambda")
}

var rules = map[TokenType]parseRule{}

func init() {
	rules = map[TokenType]parseRule{
		TokenLeftParen:    {prefix: grouping, infix: call, precedence: PrecCall},
		TokenRightParen:   {},
		TokenLeftBrace:    {prefix: map_},
		TokenRightBrace:   {},
		TokenLeftBracket:  {prefix: list, infix: subscript, precedence: PrecCall},
		TokenRightBracket: {},
		TokenComma:        {},
		TokenDot:          {infix: dot, precedence: PrecCall},
		TokenMinus:        {prefix: unary, infix: binary, precedence: PrecTerm},
		TokenPlus:         {infix: binary, precedence: PrecTerm},
		TokenSemicolon:    {},
		TokenSlash:        {infix: binary, precedence: PrecFactor},
		TokenStar:         {infix: binary, precedence: PrecFactor},
		TokenBang:         {prefix: unary},
		TokenBangEqual:    {infix: binary, precedence: PrecEquality},
		TokenEqual:        {},
		TokenEqualEqual:   {infix: binary, precedence: PrecEquality},
		TokenGreater:      {infix: binary, precedence: PrecComparison},
		TokenGreaterEqual: {infix: binary, precedence: PrecComparison},
		TokenLess:         {infix: binary, precedence: PrecComparison},
		TokenLessEqual:    {infix: binary, precedence: PrecComparison},
		TokenPlusPlus:     {prefix: unary},
		TokenMinusMinus:   {prefix: unary},
		TokenIdentifier:   {prefix: variable},
		TokenString:       {prefix: string_},
		TokenNumber:       {prefix: number},
		TokenQuestion:     {infix: ternary, precedence: PrecTernary},
		TokenColon:        {},
		TokenAnd:          {infix: and_, precedence: PrecAnd},
		TokenClass:        {},
		TokenConst:        {},
		TokenElse:         {},
		TokenExport:       {},
		TokenFalse:        {prefix: literal},
		TokenFun:          {prefix: lambda},
		TokenFor:          {},
		TokenIf:           {},
		TokenNil:          {prefix: literal},
		TokenOr:           {infix: or_, precedence: PrecOr},
		TokenPrint:        {},
		TokenReturn:       {},
		TokenSuper:        {prefix: super_},
		TokenThis:         {prefix: this_},
		TokenTrue:         {prefix: literal},
		TokenVar:          {},
		TokenWhile:        {},
		TokenBreak:        {},
		TokenContinue:     {},
		TokenError:        {},
		TokenEOF:          {},
	}
}
