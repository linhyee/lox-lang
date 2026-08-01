package glox

func (p *Parser) declaration() {
	if p.match(TokenExport) {
		p.exportDeclaration()
	} else if p.match(TokenClass) {
		p.classDeclaration(false)
	} else if p.match(TokenFun) {
		p.funDeclaration(false)
	} else if p.match(TokenVar) {
		p.varDeclaration(false, false)
	} else if p.match(TokenConst) {
		p.varDeclaration(true, false)
	} else {
		p.statement()
	}

	if p.panicMode {
		p.synchronize()
	}
}

func (p *Parser) exportDeclaration() {
	if p.compiler.scopeDepth != 0 {
		p.error("can only export from top-level")
		return
	}
	if p.match(TokenVar) {
		p.varDeclaration(false, true)
		return
	}
	if p.match(TokenConst) {
		p.varDeclaration(true, true)
		return
	}
	if p.match(TokenFun) {
		p.funDeclaration(true)
		return
	}
	if p.match(TokenClass) {
		p.classDeclaration(true)
		return
	}
	if p.match(TokenLeftBrace) {
		for !p.check(TokenRightBrace) && !p.check(TokenEOF) {
			p.consume(TokenIdentifier, "expect exported variable name")
			p.module.Export(p.previous.Lexeme)
			if !p.match(TokenComma) {
				break
			}
		}
		p.consume(TokenRightBrace, "expect '}' after export list")
		_ = p.match(TokenSemicolon)
		return
	}
	p.error("expect 'var', 'const', 'fun', 'class', or '{' after 'export'")
}

func (p *Parser) classDeclaration(exported bool) {
	nameConstant, name := p.parseVariable("expect class name", false)
	className := p.previous
	p.emitBytes(byte(OpClass), nameConstant)
	p.defineVariable(nameConstant, false)
	if exported {
		p.module.Export(name)
	}

	enclosingClass := p.class
	p.class = &ClassContext{Enclosing: enclosingClass}

	if p.match(TokenLess) {
		p.consume(TokenIdentifier, "expect superclass name")
		superName := p.previous.Lexeme
		if superName == name {
			p.error("a class can't inherit from itself")
		}
		p.namedVariable(p.previous, false)
		p.beginScope()
		p.addLocal("super", false)
		p.markInitialized()
		p.namedVariable(className, false)
		p.emitOp(OpInherit)
		p.class.HasSuperclass = true
	}

	p.namedVariable(className, false)
	p.consume(TokenLeftBrace, "expect '{' before class body")
	for !p.check(TokenRightBrace) && !p.check(TokenEOF) {
		p.method()
	}
	p.consume(TokenRightBrace, "expect '}' after class body")
	p.emitOp(OpPop)

	if p.class.HasSuperclass {
		p.endScope()
	}
	p.class = enclosingClass
}

func (p *Parser) method() {
	p.consume(TokenIdentifier, "expect method name")
	name := p.previous.Lexeme
	constant := p.makeConstant(name)
	functionType := FunctionMethod
	if name == "init" {
		functionType = FunctionInitializer
	}
	p.function(functionType, name)
	p.emitBytes(byte(OpMethod), constant)
}

func (p *Parser) funDeclaration(exported bool) {
	global, name := p.parseVariable("expect function name", false)
	p.markInitialized()
	p.function(FunctionFunction, name)
	p.defineVariable(global, false)
	if exported {
		p.module.Export(name)
	}
}

func (p *Parser) function(functionType FunctionType, name string) {
	p.initCompiler(functionType, name)
	p.beginScope()
	p.consume(TokenLeftParen, "expect '(' after function name")
	if !p.check(TokenRightParen) {
		for {
			p.compiler.function.Arity++
			if p.compiler.function.Arity > 255 {
				p.errorAtCurrent("can't have more than 255 parameters")
			}
			p.parseVariable("expect parameter name", false)
			p.defineVariable(0, false)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	p.consume(TokenRightParen, "expect ')' after parameters")
	p.consume(TokenLeftBrace, "expect '{' before function body")
	p.block()
	upvalues := append([]UpvalueSpec(nil), p.compiler.upvalues...)
	function := p.endCompiler()
	constant := p.makeConstant(function)
	p.emitBytes(byte(OpClosure), constant)
	for _, upvalue := range upvalues {
		if upvalue.IsLocal {
			p.emitByte(1)
		} else {
			p.emitByte(0)
		}
		p.emitByte(upvalue.Index)
		if upvalue.IsConst {
			p.emitByte(1)
		} else {
			p.emitByte(0)
		}
	}
}

func (p *Parser) varDeclaration(isConst, exported bool) {
	for {
		global, name := p.parseVariable("expect variable name", isConst)
		if p.match(TokenEqual) {
			p.expression()
		} else if isConst {
			p.error("const declaration requires an initializer")
			p.emitOp(OpNil)
		} else {
			p.emitOp(OpNil)
		}
		p.defineVariable(global, isConst)
		if exported {
			p.module.Export(name)
		}
		if !p.match(TokenComma) {
			break
		}
	}
	p.consume(TokenSemicolon, "expect ';' after variable declaration")
}

func (p *Parser) statement() {
	if p.match(TokenPrint) {
		p.printStatement()
	} else if p.match(TokenFor) {
		p.forStatement()
	} else if p.match(TokenIf) {
		p.ifStatement()
	} else if p.match(TokenReturn) {
		p.returnStatement()
	} else if p.match(TokenWhile) {
		p.whileStatement()
	} else if p.match(TokenBreak) {
		p.breakStatement()
	} else if p.match(TokenContinue) {
		p.continueStatement()
	} else if p.match(TokenLeftBrace) {
		p.beginScope()
		p.block()
		p.endScope()
	} else {
		p.expressionStatement()
	}
}

func (p *Parser) block() {
	for !p.check(TokenRightBrace) && !p.check(TokenEOF) {
		p.declaration()
	}
	p.consume(TokenRightBrace, "expect '}' after block")
}

func (p *Parser) printStatement() {
	p.expression()
	p.consume(TokenSemicolon, "expect ';' after value")
	p.emitOp(OpPrint)
}

func (p *Parser) expressionStatement() {
	p.expression()
	p.consume(TokenSemicolon, "expect ';' after expression")
	p.emitOp(OpPop)
}

func (p *Parser) ifStatement() {
	p.consume(TokenLeftParen, "expect '(' after 'if'")
	p.expression()
	p.consume(TokenRightParen, "expect ')' after condition")

	thenJump := p.emitJump(OpJumpIfFalse)
	p.emitOp(OpPop)
	p.statement()
	elseJump := p.emitJump(OpJump)

	p.patchJump(thenJump)
	p.emitOp(OpPop)
	if p.match(TokenElse) {
		p.statement()
	}
	p.patchJump(elseJump)
}

func (p *Parser) whileStatement() {
	loopStart := len(p.currentChunk().Code)
	p.consume(TokenLeftParen, "expect '(' after 'while'")
	p.expression()
	p.consume(TokenRightParen, "expect ')' after condition")
	exitJump := p.emitJump(OpJumpIfFalse)
	p.emitOp(OpPop)

	loop := &LoopContext{Start: loopStart, ScopeDepth: p.compiler.scopeDepth}
	p.loops = append(p.loops, loop)
	p.statement()
	p.loops = p.loops[:len(p.loops)-1]

	p.emitLoop(loopStart)
	p.patchJump(exitJump)
	p.emitOp(OpPop)
	for _, jump := range loop.Breaks {
		p.patchJump(jump)
	}
}

func (p *Parser) forStatement() {
	p.beginScope()
	p.consume(TokenLeftParen, "expect '(' after 'for'")
	if p.match(TokenSemicolon) {
	} else if p.match(TokenVar) {
		p.varDeclaration(false, false)
	} else if p.match(TokenConst) {
		p.varDeclaration(true, false)
	} else {
		p.expressionStatement()
	}

	loopStart := len(p.currentChunk().Code)
	exitJump := -1
	if !p.match(TokenSemicolon) {
		p.expression()
		p.consume(TokenSemicolon, "expect ';' after loop condition")
		exitJump = p.emitJump(OpJumpIfFalse)
		p.emitOp(OpPop)
	}

	bodyJump := -1
	if !p.match(TokenRightParen) {
		bodyJump = p.emitJump(OpJump)
		incrementStart := len(p.currentChunk().Code)
		p.expression()
		p.emitOp(OpPop)
		p.consume(TokenRightParen, "expect ')' after for clauses")
		p.emitLoop(loopStart)
		loopStart = incrementStart
		p.patchJump(bodyJump)
	}

	loop := &LoopContext{Start: loopStart, ScopeDepth: p.compiler.scopeDepth}
	p.loops = append(p.loops, loop)
	p.statement()
	p.loops = p.loops[:len(p.loops)-1]

	p.emitLoop(loopStart)
	if exitJump != -1 {
		p.patchJump(exitJump)
		p.emitOp(OpPop)
	}
	for _, jump := range loop.Breaks {
		p.patchJump(jump)
	}
	p.endScope()
}

func (p *Parser) returnStatement() {
	if p.compiler.function.Type == FunctionScript {
		p.error("can't return from top-level code")
	}
	if p.match(TokenSemicolon) {
		p.emitReturn()
		return
	}
	if p.compiler.function.Type == FunctionInitializer {
		p.error("can't return a value from an initializer")
	}
	p.expression()
	p.consume(TokenSemicolon, "expect ';' after return value")
	p.emitOp(OpReturn)
}

func (p *Parser) breakStatement() {
	if len(p.loops) == 0 {
		p.error("can't use 'break' outside of a loop")
	}
	p.consume(TokenSemicolon, "expect ';' after 'break'")
	if len(p.loops) == 0 {
		return
	}
	loop := p.loops[len(p.loops)-1]
	p.emitPopLocals(loop.ScopeDepth)
	loop.Breaks = append(loop.Breaks, p.emitJump(OpJump))
}

func (p *Parser) continueStatement() {
	if len(p.loops) == 0 {
		p.error("can't use 'continue' outside of a loop")
	}
	p.consume(TokenSemicolon, "expect ';' after 'continue'")
	if len(p.loops) == 0 {
		return
	}
	loop := p.loops[len(p.loops)-1]
	p.emitPopLocals(loop.ScopeDepth)
	p.emitLoop(loop.Start)
}

func (p *Parser) synchronize() {
	p.panicMode = false
	for !p.check(TokenEOF) {
		if p.previous.Type == TokenSemicolon {
			return
		}
		switch p.current.Type {
		case TokenClass, TokenFun, TokenVar, TokenConst, TokenFor, TokenIf, TokenWhile, TokenPrint, TokenReturn, TokenExport:
			return
		}
		p.advance()
	}
}
