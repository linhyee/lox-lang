package glox

type Precedence int

const (
	PrecNone Precedence = iota
	PrecAssignment
	PrecTernary
	PrecOr
	PrecAnd
	PrecEquality
	PrecComparison
	PrecTerm
	PrecFactor
	PrecUnary
	PrecCall
	PrecPrimary
)

type parseFn func(*Parser, bool)

type parseRule struct {
	prefix     parseFn
	infix      parseFn
	precedence Precedence
}

type Local struct {
	Name       string
	Depth      int
	IsCaptured bool
	IsConst    bool
}

type LoopContext struct {
	Start      int
	ScopeDepth int
	Breaks     []int
}

type ClassContext struct {
	Enclosing     *ClassContext
	HasSuperclass bool
}

type Compiler struct {
	enclosing  *Compiler
	function   *Function
	locals     []Local
	upvalues   []UpvalueSpec
	scopeDepth int
}

type Parser struct {
	scanner   *Scanner
	current   Token
	previous  Token
	panicMode bool

	diagnostics *Diagnostics
	module      *Module
	compiler    *Compiler
	class       *ClassContext
	loops       []*LoopContext
	globalConst map[string]bool
}

func Compile(source, path string, module *Module, diagnostics *Diagnostics) (*Function, error) {
	if diagnostics == nil {
		diagnostics = NewDiagnostics(nil)
	}
	parser := &Parser{
		scanner:     NewScanner(source),
		diagnostics: diagnostics,
		module:      module,
		globalConst: map[string]bool{},
	}
	parser.initCompiler(FunctionScript, "")
	parser.advance()
	for !parser.match(TokenEOF) {
		parser.declaration()
	}
	function := parser.endCompiler()
	if diagnostics.HadError() {
		return nil, diagnostics.Err()
	}
	return function, nil
}

func (p *Parser) initCompiler(functionType FunctionType, name string) {
	compiler := &Compiler{
		enclosing: p.compiler,
		function:  &Function{Name: name, Type: functionType},
		locals:    make([]Local, 0, 256),
		upvalues:  make([]UpvalueSpec, 0, 256),
	}
	localName := ""
	if functionType == FunctionMethod || functionType == FunctionInitializer {
		localName = "this"
	}
	compiler.locals = append(compiler.locals, Local{Name: localName, Depth: 0})
	p.compiler = compiler
}

func (p *Parser) endCompiler() *Function {
	p.emitReturn()
	function := p.compiler.function
	function.UpvalueCount = len(p.compiler.upvalues)
	p.compiler = p.compiler.enclosing
	return function
}

func (p *Parser) currentChunk() *Chunk {
	return &p.compiler.function.Chunk
}

func (p *Parser) advance() {
	p.previous = p.current
	for {
		p.current = p.scanner.ScanToken()
		if p.current.Type != TokenError {
			break
		}
		p.errorAtCurrent(p.current.Lexeme)
	}
}

func (p *Parser) consume(typ TokenType, message string) {
	if p.current.Type == typ {
		p.advance()
		return
	}
	p.errorAtCurrent(message)
}

func (p *Parser) check(typ TokenType) bool {
	return p.current.Type == typ
}

func (p *Parser) match(typ TokenType) bool {
	if !p.check(typ) {
		return false
	}
	p.advance()
	return true
}

func (p *Parser) errorAtCurrent(message string) {
	p.errorAt(p.current, message)
}

func (p *Parser) error(message string) {
	p.errorAt(p.previous, message)
}

func (p *Parser) errorAt(token Token, message string) {
	if p.panicMode {
		return
	}
	p.panicMode = true
	p.diagnostics.ErrorAt(token, message)
}

func (p *Parser) emitByte(b byte) {
	p.currentChunk().Write(b, p.previous.Line)
}

func (p *Parser) emitBytes(a, b byte) {
	p.emitByte(a)
	p.emitByte(b)
}

func (p *Parser) emitOp(op OpCode) {
	p.emitByte(byte(op))
}

func (p *Parser) emitReturn() {
	if p.compiler.function.Type == FunctionInitializer {
		p.emitBytes(byte(OpGetLocal), 0)
	} else {
		p.emitOp(OpNil)
	}
	p.emitOp(OpReturn)
}

func (p *Parser) makeConstant(value Value) byte {
	constant := p.currentChunk().AddConstant(value)
	if constant > 255 {
		p.error("too many constants in one chunk")
		return 0
	}
	return byte(constant)
}

func (p *Parser) emitConstant(value Value) {
	p.emitBytes(byte(OpConstant), p.makeConstant(value))
}

func (p *Parser) emitJump(op OpCode) int {
	p.emitOp(op)
	p.emitByte(0xff)
	p.emitByte(0xff)
	return len(p.currentChunk().Code) - 2
}

func (p *Parser) patchJump(offset int) {
	jump := len(p.currentChunk().Code) - offset - 2
	if jump > 0xffff {
		p.error("too much code to jump over")
		return
	}
	p.currentChunk().Code[offset] = byte((jump >> 8) & 0xff)
	p.currentChunk().Code[offset+1] = byte(jump & 0xff)
}

func (p *Parser) emitLoop(loopStart int) {
	p.emitOp(OpLoop)
	offset := len(p.currentChunk().Code) - loopStart + 2
	if offset > 0xffff {
		p.error("loop body too large")
		return
	}
	p.emitByte(byte((offset >> 8) & 0xff))
	p.emitByte(byte(offset & 0xff))
}

func (p *Parser) beginScope() {
	p.compiler.scopeDepth++
}

func (p *Parser) endScope() {
	p.compiler.scopeDepth--
	for len(p.compiler.locals) > 0 && p.compiler.locals[len(p.compiler.locals)-1].Depth > p.compiler.scopeDepth {
		local := &p.compiler.locals[len(p.compiler.locals)-1]
		if local.IsCaptured {
			p.emitOp(OpCloseUpvalue)
		} else {
			p.emitOp(OpPop)
		}
		p.compiler.locals = p.compiler.locals[:len(p.compiler.locals)-1]
	}
}

func (p *Parser) emitPopLocals(targetDepth int) {
	for i := len(p.compiler.locals) - 1; i >= 0; i-- {
		local := p.compiler.locals[i]
		if local.Depth <= targetDepth {
			break
		}
		if local.IsCaptured {
			p.emitOp(OpCloseUpvalue)
		} else {
			p.emitOp(OpPop)
		}
	}
}
