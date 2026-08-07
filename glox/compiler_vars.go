package glox

func (p *Parser) parseVariable(message string, isConst bool) (byte, string) {
	p.consume(TokenIdentifier, message)
	name := p.previous.Lexeme
	p.declareVariable(isConst)
	if p.compiler.scopeDepth > 0 {
		return 0, name
	}
	if isConst {
		p.globalConst[name] = true
	}
	return p.identifierConstant(name), name
}

func (p *Parser) identifierConstant(name string) byte {
	return p.makeConstant(name)
}

func (p *Parser) declareVariable(isConst bool) {
	if p.compiler.scopeDepth == 0 {
		return
	}
	name := p.previous.Lexeme
	for i := len(p.compiler.locals) - 1; i >= 0; i-- {
		local := p.compiler.locals[i]
		if local.Depth != -1 && local.Depth < p.compiler.scopeDepth {
			break
		}
		if local.Name == name {
			p.error("already a variable with this name in this scope")
		}
	}
	p.addLocal(name, isConst)
}

func (p *Parser) addLocal(name string, isConst bool) {
	if len(p.compiler.locals) >= 256 {
		p.error("too many local variables in function")
		return
	}
	p.compiler.locals = append(p.compiler.locals, Local{Name: name, Depth: -1, IsConst: isConst})
}

func (p *Parser) markInitialized() {
	if p.compiler.scopeDepth == 0 {
		return
	}
	p.compiler.locals[len(p.compiler.locals)-1].Depth = p.compiler.scopeDepth
}

func (p *Parser) defineVariable(global byte, isConst bool) {
	if p.compiler.scopeDepth > 0 {
		p.markInitialized()
		return
	}
	if isConst {
		p.emitBytes(byte(OpDefineGlobalConst), global)
	} else {
		p.emitBytes(byte(OpDefineGlobal), global)
	}
}

func (p *Parser) resolveLocal(compiler *Compiler, name string) int {
	for i := len(compiler.locals) - 1; i >= 0; i-- {
		local := compiler.locals[i]
		if local.Name == name {
			if local.Depth == -1 {
				p.error("can't read local variable in its own initializer")
			}
			return i
		}
	}
	return -1
}

func (p *Parser) addUpvalue(compiler *Compiler, index uint8, isLocal, isConst bool) int {
	for i, upvalue := range compiler.upvalues {
		if upvalue.Index == index && upvalue.IsLocal == isLocal {
			return i
		}
	}
	if len(compiler.upvalues) >= 256 {
		p.error("too many closure variables in function")
		return 0
	}
	compiler.upvalues = append(compiler.upvalues, UpvalueSpec{
		Index:   index,
		IsLocal: isLocal,
		IsConst: isConst,
	})
	return len(compiler.upvalues) - 1
}

func (p *Parser) resolveUpvalue(compiler *Compiler, name string) int {
	if compiler.enclosing == nil {
		return -1
	}
	if local := p.resolveLocal(compiler.enclosing, name); local != -1 {
		compiler.enclosing.locals[local].IsCaptured = true
		return p.addUpvalue(compiler, uint8(local), true, compiler.enclosing.locals[local].IsConst)
	}
	if upvalue := p.resolveUpvalue(compiler.enclosing, name); upvalue != -1 {
		return p.addUpvalue(compiler, uint8(upvalue), false, compiler.enclosing.upvalues[upvalue].IsConst)
	}
	return -1
}

func (p *Parser) variableIsConst(name Token, local, upvalue int) bool {
	if local != -1 {
		return p.compiler.locals[local].IsConst
	}
	if upvalue != -1 {
		return p.compiler.upvalues[upvalue].IsConst
	}
	return p.globalConst[name.Lexeme]
}

func (p *Parser) namedVariable(name Token, canAssign bool) {
	getOp := OpGetGlobal
	setOp := OpSetGlobal
	arg := p.identifierConstant(name.Lexeme)

	local := p.resolveLocal(p.compiler, name.Lexeme)
	upvalue := -1
	if local != -1 {
		getOp = OpGetLocal
		setOp = OpSetLocal
		arg = byte(local)
	} else if upvalue = p.resolveUpvalue(p.compiler, name.Lexeme); upvalue != -1 {
		getOp = OpGetUpvalue
		setOp = OpSetUpvalue
		arg = byte(upvalue)
	}

	if canAssign && p.match(TokenEqual) {
		if p.variableIsConst(name, local, upvalue) {
			p.error("cannot assign to const '" + name.Lexeme + "'")
		}
		p.expression()
		p.emitBytes(byte(setOp), arg)
		return
	}
	if canAssign && (p.match(TokenPlusPlus) || p.match(TokenMinusMinus)) {
		operator := p.previous.Type
		if p.variableIsConst(name, local, upvalue) {
			p.error("cannot assign to const '" + name.Lexeme + "'")
		}
		p.emitBytes(byte(getOp), arg)
		p.emitOp(OpDup)
		p.emitConstant(int64(1))
		if operator == TokenPlusPlus {
			p.emitOp(OpAdd)
		} else {
			p.emitOp(OpSubtract)
		}
		p.emitBytes(byte(setOp), arg)
		p.emitOp(OpPop)
		return
	}

	p.emitBytes(byte(getOp), arg)
}
