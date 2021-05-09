#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "common.h"
#include "compiler.h"
#include "memory.h"
#include "scanner.h"
#include "utf8.h"

#ifdef DEBUG_PRINT_CODE
#include "debug.h"
#endif

typedef struct {
  Token current;
  Token previous;
  bool hadError;
  bool panicMode;
} Parser;

// precedence levels in order from lowest to highest.
typedef enum {
  PREC_NONE,
  PREC_ASSIGNMENT, // =
  PREC_OR,         // or
  PREC_AND,        // and
  PREC_EQUALITY,   // == !=
  PREC_COMPARISON, // < > <= >=
  PREC_TERM,       // + -
  PREC_FACTOR,     // * /
  PREC_UNARY,      // ! -
  PREC_CALL,       // . () []
  PREC_PRIMARY
} Precedence;

typedef void (*ParseFn)(bool canAssign);

typedef struct {
  ParseFn prefix;
  ParseFn infix;
  Precedence precedence;
} ParseRule;

// local variable
typedef struct {
  Token name;
  int depth;
  bool isCaptured;
} Local;

// upvalue
typedef struct {
  uint8_t index;
  bool isLocal;
} Upvalue;

typedef enum {
  TYPE_FUNCTION,
  TYPE_INITIALIZER,
  TYPE_METHOD,
  TYPE_SCRIPT
} FunctionType;

// local variables compiler-struct < calls and funtions enclosing-field
typedef struct Compiler {
  // 使用链表管理函数嵌套调用的问题
  struct Compiler* enclosing;
  ObjFunction* function;
  FunctionType type;

  Local locals[UINT8_COUNT];
  int localCount;
  Upvalue upvalues[UINT8_COUNT];
  int scopeDepth;
} Compiler;

typedef struct ClassCompiler {
  struct ClassCompiler* enclosing;
  bool hasSuperclass;
  Token name;
} ClassCompiler;

Parser parser;

// local variables current-compiler
Compiler* current = NULL;

// current-class
ClassCompiler* currentClass = NULL;

// loop position for continue jump to
int innermostLoopStart = -1;
// the scope of the variables declared inside the loop. 
// when it execute continue, must pop the value on the stack
int innermostLoopScopeDepth = 0;

// like aboves
int innermostBreakScopeStart = -1;
int innermostBreakScopeDepth = 0;
// how many breaks what loop or switch holds 
int* innermostBreakJumps = NULL;
int innermostBreakJumpCount = 0;

// dumping chunks for debuging when compiling has done
Chunk* compilingChunk;

static Chunk* currentChunk() {
  return &current->function->chunk;
}

static void errorAt(Token* token, const char* message) {
  if (parser.panicMode) return;
  parser.panicMode = true;

  fprintf(stderr, "[line %d] Error", token->line);

  if (token->type == TOKEN_EOF) {
    fprintf(stderr, " at end");
  } else if (token->type == TOKEN_ERROR) {
    // nothing
  } else {
    fprintf(stderr, " at '%.*s'", token->length, token->start);
  }

  fprintf(stderr, ": %s\n", message);
  parser.hadError = true;
}

static void error(const char* message) {
  errorAt(&parser.previous, message);
}

static void errorAtCurrent(const char* message) {
  errorAt(&parser.current, message);
}

static void advance() {
  parser.previous = parser.current;

  for (;;) {
    parser.current = scanToken();
    if (parser.current.type != TOKEN_ERROR) break;

    errorAtCurrent(parser.current.start);
  }
}

static void consume(TokenType type, const char* message) {
  if (parser.current.type == type) {
    advance();
    return;
  }
  errorAtCurrent(message) ;
}

static bool check(TokenType type) {
  return parser.current.type == type;
}

static bool match(TokenType type) {
  if (!check(type)) return false;
  advance();
  return true;
}

static void emitByte(uint8_t byte) {
  writeChunk(currentChunk(), byte, parser.previous.line);
}

static void emitBytes(uint8_t byte1, uint8_t byte2) {
  emitByte(byte1);
  emitByte(byte2);
}

static void emitLoop(int loopStart) {
  emitByte(OP_LOOP);

  int offset = currentChunk()->count - loopStart + 2;
  if (offset > UINT16_MAX) error("loop body too large.");

  emitByte((offset >> 8) & 0xff);
  emitByte(offset & 0xff);
}

static int emitJump(uint8_t instruction) {
  emitByte(instruction);
  emitByte(0xff);
  emitByte(0xff);
  return currentChunk()->count - 2; //返回后面patch要写入的位置
}

static void emitReturn() {
  if (current->type == TYPE_INITIALIZER) { 
    emitBytes(OP_GET_LOCAL, 0);
  } else {
    emitByte(OP_NIL);
  }
  emitByte(OP_RETURN);
}

static uint8_t makeConstant(Value value) {
  int constant = addConstant(currentChunk(), value);
  if (constant > UINT8_MAX) {
    error("too many constants in one chunk");
    return 0;
  }

  return (uint8_t)constant;
}

static void emitConstant(Value value) {
  emitBytes(OP_CONSTANT, makeConstant(value));
}

static void patchJump(int offset) {
  // -2 to adjust for the bytecode for the jump offset itself.
  int jump = currentChunk()->count - offset - 2;

  if (jump > UINT16_MAX) {
    error("too much code to jump over.");
  }

  currentChunk()->code[offset] = (jump >> 8) & 0xff;
  currentChunk()->code[offset + 1] = jump & 0xff;
}

static void initCompiler(Compiler* compiler, FunctionType type) {
  compiler->enclosing = current;
  compiler->function = NULL;
  compiler->type = type;
  compiler->localCount = 0;
  compiler->scopeDepth = 0;
  compiler->function = newFunction();
  current = compiler;

  if (type != TYPE_SCRIPT) {
    // 函数名字是贯穿运行时的,所以在堆上分配
    current->function->name = copyString(parser.previous.start,
      parser.previous.length);
  }

  Local* local = &current->locals[current->localCount++];
  local->depth = 0;
  local->isCaptured = false;
  if (type != TYPE_FUNCTION) { 
    local->name.start = "this";
    local->name.length = 4;
  } else {
    local->name.start = "";
    local->name.length = 0;
  }
}

static ObjFunction* endCompiler() {
  emitReturn();
  ObjFunction* function = current->function;

#ifdef DEBUG_PRINT_CODE
  if (!parser.hadError) {
    disassembleChunk(currentChunk(), function->name != NULL
      ? function->name->chars : "<script>");
  }
#endif

  // 将控制权交给上一个函数
  current = current->enclosing;
  return function;
}

static void beginScope() {
  current->scopeDepth++;
}

static void endScope() {
  current->scopeDepth--;

  // 当块结束时候,将块的局部变量全部弹出
  while (current->localCount > 0 &&
    current->locals[current->localCount - 1].depth > 
      current->scopeDepth) {
    if (current->locals[current->localCount - 1].isCaptured) {
      emitByte(OP_CLOSE_UPVALUE);
    } else {
      emitByte(OP_POP);
    }
    current->localCount--;
  }
}

// compiling Expressions forward-declarations
static void expression();
static void statement();
static void declaration();
static ParseRule* getRule(TokenType type);
static void parsePrecedence(Precedence precedence);

static uint8_t identifierConstant(Token* name) {
  return makeConstant(
      OBJ_VAL(copyString(name->start, name->length)));
}

static bool identifiersEqual(Token* a, Token* b) {
  if (a->length != b->length) return false;
  return memcmp(a->start, b->start, a->length) == 0;
}

static int resolveLocal(Compiler* compiler, Token* name) {
  for (int i= compiler->localCount - 1; i >= 0; i--) {
    Local* local = &compiler->locals[i];
    if (identifiersEqual(name, &local->name)) {
      if (local->depth == -1) {
        error("can't read local variable in its own initializer.");
      }
      return i;
    }
  } 
  return -1;
}

static int addUpvalue(Compiler* compiler, uint8_t index, bool isLocal) {
  int upvalueCount = compiler->function->upvalueCount;

  // 可能存在多次引用闭包变量
  for (int i = 0; i < upvalueCount; i++) {
    Upvalue* upvalue = &compiler->upvalues[i];
    if (upvalue->index == index && upvalue->isLocal == isLocal) {
      return i;
    }
  }

  if (upvalueCount == UINT8_COUNT) {
    error("too many closure variables in function.");
    return 0;
  }

  compiler->upvalues[upvalueCount].isLocal = isLocal;
  compiler->upvalues[upvalueCount].index = index;
  return compiler->function->upvalueCount++;
}

static int resolveUpvalue(Compiler* compiler, Token* name) {
  if (compiler->enclosing == NULL) return -1;

  // 在前一个enclosing环境查找
  int local = resolveLocal(compiler->enclosing, name);
  if (local != -1) {
    // 将变量标记为被闭包捕足
    compiler->enclosing->locals[local].isCaptured = true;
    return addUpvalue(compiler, (uint8_t)local, true);
  }
  // 前一个环境没有找到,则往前前一个环境寻找即使前一个环境没有引用该闭包变量,
  // 如果找到了, 并将闭包变量位置递归返回, 保存到当前一个环境当中.
  int upvalue = resolveUpvalue(compiler->enclosing, name);
  if (upvalue != -1) {
    return addUpvalue(compiler, (uint8_t) upvalue, false); // false表明没有在enclosing中找到
  }

  return -1;
}

static void addLocal(Token name) {
  if (current->localCount == UINT8_COUNT) {
    error("too many local variables in function.");
    return;
  }

  Local* local = &current->locals[current->localCount++];
  local->name = name;
  local->depth = -1;
  local->isCaptured = false;
}

static void declareVariable() {
  if (current->scopeDepth == 0) return;

  Token* name = &parser.previous;
  for (int i = current->localCount - 1; i >= 0; i--) {
    Local* local = &current->locals[i];
    if (local->depth != -1 && local->depth < current->scopeDepth) {
      break;
    }

    if (identifiersEqual(name, &local->name)) {
      error("already variable with this name in this scope.");
    }
  }

  addLocal(*name);
}

static uint8_t parseVariable(const char* errorMessage) {
  consume(TOKEN_IDENTIFIER, errorMessage);
  declareVariable();
  // 如是是本地变量, 需将变量的名称填充到常表中, 返回虚拟索引
  if (current->scopeDepth > 0) return 0;

  return identifierConstant(&parser.previous);
}

static void markInitialized() {
  if (current->scopeDepth == 0) return; // ensures function can access it's owner name
  current->locals[current->localCount - 1].depth = current->scopeDepth;
}

static void defineVariable(uint8_t global) {
  if (current->scopeDepth > 0) {
    markInitialized();
    return;
  }
  emitBytes(OP_DEFINE_GLOBAL, global);
}

static uint8_t argumentList() {
  uint8_t argCount = 0;
  if (!check(TOKEN_RIGHT_PAREN)) {
    do {
      expression();
      if (argCount == 255) {
        error("can't have more than 255 arguments.");
      }
      argCount++;
    } while (match(TOKEN_COMMA));
  }
  consume(TOKEN_RIGHT_PAREN, "expect ')' after arguments.");
  return argCount;
}

static void and_(bool canAssign) {
  int endJump = emitJump(OP_JUMP_IF_FALSE);

  emitByte(OP_POP);
  parsePrecedence(PREC_AND);

  patchJump(endJump);
}

static void binary(bool canAssign) {
  // remember the operator.
  TokenType operatorType = parser.previous.type;

  // compile the right operand.
  ParseRule* rule = getRule(operatorType); 
  parsePrecedence((Precedence)(rule->precedence + 1));

  // emit the operator instruction
  switch (operatorType) {
  case TOKEN_BANG_EQUAL:    emitBytes(OP_EQUAL, OP_NOT); break;
  case TOKEN_EQUAL_EQUAL:   emitByte(OP_EQUAL); break;
  case TOKEN_GREATER:       emitByte(OP_GREATER); break;
  case TOKEN_GREATER_EQUAL: emitBytes(OP_LESS, OP_NOT); break;
  case TOKEN_LESS:          emitByte(OP_LESS); break;
  case TOKEN_LESS_EQUAL:    emitBytes(OP_GREATER, OP_NOT); break;
  case TOKEN_PLUS:          emitByte(OP_ADD); break;
  case TOKEN_MINUS:         emitByte(OP_SUBTRACT); break; 
  case TOKEN_STAR:          emitByte(OP_MULTIPLY); break;
  case TOKEN_SLASH:         emitByte(OP_DIVIDE); break;
  default:                  return; // unreachable.
  }
}

static void subscript(bool canAssign) {
  bool hasExpr = true;
  if (!check(TOKEN_RIGHT_BRACKET)) {
    expression();
  } else {
    hasExpr = false;
  }
  consume(TOKEN_RIGHT_BRACKET, "expect ']' after subscript.");

  if (!hasExpr && check(TOKEN_LEFT_BRACKET)) {
    error("expect index expression at previous's '[]'.");
  }

  if (canAssign && match(TOKEN_EQUAL)) {
    expression();
    if (hasExpr) {
      emitByte(OP_SET_INDEX);
    } else {
      emitByte(OP_SHIFT_INDEX);
    }
  } else {
    if (!hasExpr) {
      error("expect index expression at '[]'.");
    } else {
      emitByte(OP_GET_INDEX);
    }
  }
}

static void call(bool canAssign) {
  uint8_t argCount = argumentList();
  emitBytes(OP_CALL, argCount);
}

static void dot(bool canAssign) {
  consume(TOKEN_IDENTIFIER, "expect property name after '.'.");
  uint8_t name = identifierConstant(&parser.previous);

  if (canAssign && match(TOKEN_EQUAL)) { 
    expression();
    emitBytes(OP_SET_PROPERTY, name);
  } else if (match(TOKEN_LEFT_PAREN)) { 
    uint8_t argCount = argumentList();
    emitBytes(OP_INVOKE, name);
    emitByte(argCount);
  } else { 
    emitBytes(OP_GET_PROPERTY, name);
  }
}

static void literal(bool canAssign) {
  switch (parser.previous.type) {
  case TOKEN_FALSE: emitByte(OP_FAlSE); break;
  case TOKEN_NIL:   emitByte(OP_NIL); break;
  case TOKEN_TRUE:  emitByte(OP_TRUE); break;
  default:          return; // unreachable
  }
}

static void grouping(bool canAssign) {
  expression();
  consume(TOKEN_RIGHT_PAREN, "expect ')' after expression.");
}

// a list literal
static void list(bool canAssign) {
  int length = 0;
  do {
    // stopif we hit the end of the list.
    if (check(TOKEN_RIGHT_BRACKET)) break;

    // the element.
    expression();
    length++;
    if (length > 255) { //因为指令栈的限制
      error("the list constant can not have more than 255 elements.");
      return;
    }
  } while (match(TOKEN_COMMA));

  consume(TOKEN_RIGHT_BRACKET, "expect ']' after list elements.");
  emitBytes(OP_LIST, length);
}

static void number(bool canAssign) {
  double value = strtod(parser.previous.start, NULL);
  emitConstant(NUMBER_VAL(value));
}

static void or_(bool canAssign) {
  int elseJump = emitJump(OP_JUMP_IF_FALSE);
  int endJump = emitJump(OP_JUMP);

  patchJump(elseJump);
  emitByte(OP_POP);

  parsePrecedence(PREC_OR);
  patchJump(endJump);
}

static void string(bool canAssign) {
  int len = parser.previous.length - 2; // -2: leading and trailing quotes
  const char* istr = parser.previous.start + 1;
  char* ostr =ALLOCATE(char, len * sizeof(char) + 1);
  int i = 0;
  int o = 0;
 
  while (i < len) {
    // look 5 ahead?
    if (i < len - 5) {
      if (istr[i] == '\\' && istr[i+1] == 'u') {
        // \uFFFF
        utf8_int32_t codepoint = (utf8_int32_t) strtol((char*)istr + i + 2, NULL, 16);
        i += 6;
        int size = (int) utf8codepointsize(codepoint);
        utf8catcodepoint(&ostr[o], codepoint, size);
        o += size;
        continue;
      }
    }
    // look 3 ahead?
    if (i < len - 3) {
      if (istr[i]== '\\' && istr[i+1] == 'x') {
        // \xFF
        int ascii = (int) strtol((char*) istr + i + 2, NULL, 16);
        ostr[o] = (char) ascii;
        i += 4;
        o += 1;
        continue;
      }
    }
    // look 1 ahead?
    if (i < len - 1 && istr[i] == '\\') {
      switch (istr[i+1]) {
      case '\\': ostr[o] = '\\';   i += 2; o +=1; continue;
      case '"':  ostr[o] = '"';    i += 2; o +=1; continue;
      case '\'': ostr[o] = '\'';   i += 2; o +=1; continue;
      case 'a':  ostr[o] = '\a';   i += 2; o +=1; continue;
      case 'b':  ostr[o] = '\b';   i += 2; o +=1; continue;
      case 'e':  ostr[o] = '\x1B'; i += 2; o +=1; continue;
      case 'n':  ostr[o] = '\n';   i += 2; o +=1; continue;
      case 'r':  ostr[o] = '\r';   i += 2; o +=1; continue;
      case 't':  ostr[o] = '\t';   i += 2; o +=1; continue;
      case '?':  ostr[o] = '?';    i += 2; o +=1; continue;
      }
    }
    // copy char as it is
    ostr[o] = istr[i];
    i++;
    o++;
  }
  ostr[o] = '\0';

  emitConstant(OBJ_VAL(copyString(ostr, o)));
#ifdef DEBUG_TRACE_MEMORY_VERBOSE
  printf("compiler:string() FREE(char, %p) // temp buffer\n", (void*)ostr);
#endif
  FREE(char, ostr);
}

static void namedVariable(Token name, bool canAssign) {
  uint8_t getOp, setOp;
  int arg = resolveLocal(current, &name);
  if (arg != -1) { //先查找本地变量
    getOp = OP_GET_LOCAL;
    setOp = OP_SET_LOCAL;
  } else if ((arg = resolveUpvalue(current, &name)) != -1) { //再确认是否是闭包变量
    getOp = OP_GET_UPVALUE;
    setOp = OP_SET_UPVALUE;
  } else { //最后就是全局变量
    arg = identifierConstant(&name);
    getOp = OP_GET_GLOBAL;
    setOp = OP_SET_GLOBAL;
  }

  if (!canAssign) {
    emitBytes(getOp, (uint8_t)arg);
    return;
  }

  if (match(TOKEN_EQUAL)) {
    expression();
    emitBytes(setOp, (uint8_t)arg);
  } else if (match(TOKEN_PLUS_PLUS)) {
    emitBytes(getOp, (uint8_t)arg);
    emitByte(OP_INC);
    emitBytes(setOp, (uint8_t)arg);
    emitByte(OP_DEC);
  } else if (match(TOKEN_MINUS_MINUS)) {
    emitBytes(getOp, (uint8_t)arg);
    emitByte(OP_DEC);
    emitBytes(setOp, (uint8_t)arg);
    emitByte(OP_INC);
  } else {
    emitBytes(getOp, (uint8_t)arg);
  }
}

static void variable(bool canAssign) {
  namedVariable(parser.previous, canAssign);
}

static Token syntheticToken(const char* text) {
  Token token;
  token.start = text;
  token.length = (int)strlen(text);
  return token;
}

static void super_(bool canAssign) {
  if (currentClass == NULL) {
    error("can't use 'super' outside of a class.");
  } else if (!currentClass->hasSuperclass) {
    error("can't use 'super' in a class with no superclass.");
  }

  consume(TOKEN_DOT, "expect '.' after 'super'.");
  consume(TOKEN_IDENTIFIER, "expect superclass method name.");
  uint8_t name = identifierConstant(&parser.previous);

  namedVariable(syntheticToken("this"), false);

  if (match(TOKEN_LEFT_PAREN)) { 
    uint8_t argCount = argumentList();
    namedVariable(syntheticToken("super"), false);
    emitBytes(OP_SUPER_INVOKE, name);
    emitByte(argCount);
  } else {
    namedVariable(syntheticToken("super"), false);
    emitBytes(OP_GET_SUPER, name);
  }
}

static void this_(bool canAssign) {
  if (currentClass == NULL) {
    error("can't use 'this' outside of a class.");
    return;
  }
  variable(false);
}

static void unary(bool canAssign) {
  TokenType operatoType = parser.previous.type;

  // compile the operand.
  parsePrecedence(PREC_UNARY);

  // emit the operator instruction.
  switch (operatoType) {
  case TOKEN_BANG:  emitByte(OP_NOT);    break;
  case TOKEN_MINUS: emitByte(OP_NEGATE); break;
  default:          return;// unreachable.
  }
}

ParseRule rules [] = {
  [TOKEN_LEFT_PAREN]    = {grouping, call, PREC_CALL},
  [TOKEN_RIGHT_PAREN]   = {NULL, NULL, PREC_NONE},
  [TOKEN_LEFT_BRACKET]  = {list, subscript, PREC_CALL},
  [TOKEN_RIGHT_BRACKET] = {NULL, NULL, PREC_NONE},
  [TOKEN_LEFT_BRACE]    = {NULL, NULL, PREC_NONE},
  [TOKEN_RIGHT_BRACE]   = {NULL, NULL, PREC_NONE},
  [TOKEN_COMMA]         = {NULL, NULL, PREC_NONE},
  [TOKEN_DOT]           = {NULL, dot, PREC_CALL},
  [TOKEN_MINUS]         = {unary, binary, PREC_TERM},
  [TOKEN_PLUS]          = {NULL, binary, PREC_TERM},
  [TOKEN_SEMICOLON]     = {NULL, NULL, PREC_NONE},
  [TOKEN_SLASH]         = {NULL, binary, PREC_FACTOR},
  [TOKEN_STAR]          = {NULL, binary, PREC_FACTOR},
  [TOKEN_BANG]          = {unary, NULL, PREC_NONE},
  [TOKEN_BANG_EQUAL]    = {NULL, binary, PREC_EQUALITY},
  [TOKEN_EQUAL]         = {NULL, NULL, PREC_NONE},
  [TOKEN_EQUAL_EQUAL]   = {NULL, binary, PREC_EQUALITY},
  [TOKEN_GREATER]       = {NULL, binary, PREC_COMPARISON},
  [TOKEN_GREATER_EQUAL] = {NULL, binary, PREC_COMPARISON},
  [TOKEN_LESS]          = {NULL, binary, PREC_COMPARISON},
  [TOKEN_LESS_EQUAL]    = {NULL, binary, PREC_COMPARISON},
  [TOKEN_IDENTIFIER]    = {variable, NULL, PREC_NONE},
  [TOKEN_STRING]        = {string, NULL, PREC_NONE},
  [TOKEN_NUMBER]        = {number, NULL, PREC_NONE},
  [TOKEN_AND]           = {NULL, and_, PREC_AND},
  [TOKEN_CLASS]         = {NULL, NULL, PREC_NONE},
  [TOKEN_ELSE]          = {NULL, NULL, PREC_NONE},
  [TOKEN_FALSE]         = {literal, NULL, PREC_NONE},
  [TOKEN_FOR]           = {NULL, NULL, PREC_NONE},
  [TOKEN_FUN]           = {NULL, NULL, PREC_NONE},
  [TOKEN_IF]            = {NULL, NULL, PREC_NONE},
  [TOKEN_NIL]           = {literal, NULL, PREC_NONE},
  [TOKEN_OR]            = {NULL, or_, PREC_OR},
  [TOKEN_PRINT]         = {NULL, NULL, PREC_NONE},
  [TOKEN_RETURN]        = {NULL, NULL, PREC_NONE},
  [TOKEN_SUPER]         = {super_, NULL, PREC_NONE},
  [TOKEN_THIS]          = {this_, NULL, PREC_NONE},
  [TOKEN_TRUE]          = {literal, NULL, PREC_NONE},
  [TOKEN_VAR]           = {NULL, NULL, PREC_NONE},
  [TOKEN_WHILE]         = {NULL, NULL, PREC_NONE},
  [TOKEN_BREAK]         = {NULL, NULL, PREC_NONE},
  [TOKEN_CONTINUE]      = {NULL, NULL, PREC_NONE},
  [TOKEN_SWITCH]        = {NULL, NULL, PREC_NONE},
  [TOKEN_CASE]          = {NULL, NULL, PREC_NONE},
  [TOKEN_DEFAULT]       = {NULL, NULL, PREC_NONE},
  [TOKEN_ERROR]         = {NULL, NULL, PREC_NONE},
  [TOKEN_EOF]           = {NULL, NULL, PREC_NONE},
};

// compiling
static void parsePrecedence(Precedence precedence) {
  advance();
  ParseFn prefixRule = getRule(parser.previous.type)->prefix;
  if (prefixRule == NULL) {
    error("expect expression.");
    return;
  }

  bool canAssign = precedence <= PREC_ASSIGNMENT;
  prefixRule(canAssign);

  while (precedence <= getRule(parser.current.type)->precedence) {
    advance();
    ParseFn infixRule = getRule(parser.previous.type)->infix;
    infixRule(canAssign);
  }

  if (canAssign && match(TOKEN_EQUAL)) {
    error("invalid assignment target.");
  }
}

static ParseRule* getRule(TokenType type) {
  return &rules[type];
}

static void expression() {
  parsePrecedence(PREC_ASSIGNMENT);
}

static void block() {
  while (!check(TOKEN_RIGHT_BRACE) && !check(TOKEN_EOF)) {
    declaration();
  }
  consume(TOKEN_RIGHT_BRACE, "expect '}' after block.");
}

static void function(FunctionType type) { 
  Compiler compiler;
  initCompiler(&compiler, type);
  beginScope();
  
  // compiler the parameter list.
  consume(TOKEN_LEFT_PAREN, "expect '(' after function name.");
  // handle parameters.
  if (!check(TOKEN_RIGHT_PAREN)) {
    do { 
      current->function->arity++;
      if (current->function->arity > 255) {
        errorAtCurrent("can't have more than 255 parameters.");
      }

      uint8_t paramConstant = parseVariable(
        "expect parameter name.");
      defineVariable(paramConstant);
    } while(match(TOKEN_COMMA));
  }
  consume(TOKEN_RIGHT_PAREN, "expect ')' after parameters.");

  // the body.
  consume(TOKEN_LEFT_BRACE, "expect '{' before function body.");
  block();

  // create the function object.
  ObjFunction* function = endCompiler();
  emitBytes(OP_CLOSURE, makeConstant(OBJ_VAL(function)));

  for (int i = 0; i < function->upvalueCount; i++) {
    emitByte(compiler.upvalues[i].isLocal ? 1 : 0);
    emitByte(compiler.upvalues[i].index);
  }
}

static void method() {
  consume(TOKEN_IDENTIFIER, "expect method name.");
  uint8_t constant = identifierConstant(&parser.previous);

  FunctionType type = TYPE_METHOD;
  if (parser.previous.length == 4 &&
      memcmp(parser.previous.start, "init", 4) == 0) {
    type = TYPE_INITIALIZER;
  }
  function(type);
  emitBytes(OP_METHOD, constant);
}

static void classDeclaration() {
  consume(TOKEN_IDENTIFIER, "expect class name.");
  // 类有可能会在局部作用域声明
  Token className = parser.previous; // capture the name of the class
  uint8_t nameConstant = identifierConstant(&parser.previous);
  declareVariable();

  emitBytes(OP_CLASS, nameConstant);
  defineVariable(nameConstant);

  ClassCompiler classCompiler;
  classCompiler.name = parser.previous;
  classCompiler.hasSuperclass = false;
  classCompiler.enclosing = currentClass;
  currentClass = &classCompiler;

  if (match(TOKEN_LESS)) {
    consume(TOKEN_IDENTIFIER, "expect superclass name.");
    variable(false);

    if (identifiersEqual(&className, &parser.previous)) {
      error("a class can't inherit from itselt.");
    }

    beginScope();
    addLocal(syntheticToken("super"));
    defineVariable(0);

    namedVariable(className, false);
    emitByte(OP_INHERIT);
    classCompiler.hasSuperclass = true;
  }

  namedVariable(className, false); //将类型的名称加载到堆栈去,以便绑定方法时,能找到类型
  consume(TOKEN_LEFT_BRACE, "expect '{' before class body.");
  while (!check(TOKEN_RIGHT_BRACE) && !check(TOKEN_EOF)) {
    method();
  }
  
  consume(TOKEN_RIGHT_BRACE, "expect '}' after class body.");
  emitByte(OP_POP);
  
  if (classCompiler.hasSuperclass) {
    endScope();
  }
  currentClass = currentClass->enclosing;
}

static void funDeclaration() {
  uint8_t global = parseVariable("expect function name.");
  markInitialized();
  function(TYPE_FUNCTION);
  defineVariable(global);
}

static void varDeclaration() {
  uint8_t global;
decl:
  global = parseVariable(parser.previous.type == TOKEN_COMMA ? 
    "expect ';' after declaration." : "expect variable name." );

  if (match(TOKEN_EQUAL)) {
    expression();
  } else {
    emitByte(OP_NIL);
  }

  defineVariable(global);

  if (match(TOKEN_COMMA)) {
    goto decl;
  }
  consume(TOKEN_SEMICOLON, "expect ';' after variable declaration.");
}

static void expressionStatement() {
  expression();
  consume(TOKEN_SEMICOLON, "expect ';' after expression.");
  emitByte(OP_POP);
}

static void forStatement() {
  // if a for statement declares a variable, 
  // that variable should be scoped to loop body
  beginScope();

  consume(TOKEN_LEFT_PAREN, "expect '(' after 'for'.");
  if (match(TOKEN_SEMICOLON)) {
    // no initializer
  } else if (match(TOKEN_VAR)) {
    varDeclaration();
  } else {
    expressionStatement();
  }
  
  // save points
  int surroundingLoopStart = innermostLoopStart;
  int surroundingLoopScopeDepth = innermostLoopScopeDepth;
  // starting point
  innermostLoopStart = currentChunk()->count;
  innermostLoopScopeDepth = current->scopeDepth;

  // for 'break'
  int surroundingBreakScopeStart = innermostBreakScopeStart;
  int surroundingBreakScopeDepth = innermostBreakScopeDepth;
  int* surroundingBreakJumps = innermostBreakJumps;
  int surroundingBreakJumpCount = innermostBreakJumpCount;

  innermostBreakScopeStart = currentChunk()->count;
  innermostBreakScopeDepth = current->scopeDepth;
  innermostBreakJumps = ALLOCATE(int, MAX_BREAKS_PER_SCOPE);
  innermostBreakJumpCount = 0;

  int loopStart = currentChunk()->count;

  int exitJump = -1;
  if (!match(TOKEN_SEMICOLON)) {
    expression();
    consume(TOKEN_SEMICOLON, "expect ';' after loop condition.");

    // jump out of the loop if the condition is false.
    exitJump = emitJump(OP_JUMP_IF_FALSE);
    emitByte(OP_POP); // condition
  }

  if (!match(TOKEN_RIGHT_PAREN)) { 
    int bodyJump = emitJump(OP_JUMP);

    int incrementStart = currentChunk()->count;
    expression();
    emitByte(OP_POP);
    consume(TOKEN_RIGHT_PAREN, "expect ')' after for clauses.");

    emitLoop(loopStart);
    loopStart = incrementStart;
    patchJump(bodyJump);
  }

  statement();
  emitLoop(loopStart);

  if (exitJump != -1) {
    patchJump(exitJump);
    emitByte(OP_POP); // condition.
  } 

  // restore points (for continue)
  innermostLoopStart = surroundingLoopStart;
  innermostLoopScopeDepth = surroundingLoopScopeDepth;

  // patch break jump
  for (int i = 0; i < innermostBreakJumpCount; i++) {
    patchJump(innermostBreakJumps[i]);
  }
  
#ifdef DEBUG_TRACE_MEMORY_VERBOSE
  printf("compiler:forStatement() FREE(int, %p) // innermostBreakJumps\n",
    innermostBreakJumps);
#endif
  FREE(int, innermostBreakJumps);

  innermostBreakScopeStart = surroundingBreakScopeStart;
  innermostBreakScopeDepth = surroundingBreakScopeDepth;
  innermostBreakJumps = surroundingBreakJumps;
  innermostBreakJumpCount = surroundingBreakJumpCount;

  endScope();
}

static void breakStatement() {
  if (innermostBreakScopeStart == -1) {
    error("can't use 'break' outside of a loop or switch.");
  }
  consume(TOKEN_SEMICOLON, "expect ';' after 'continue'.");

  for (int i = current->localCount - 1;
       i >= 0 && current->locals[i].depth > innermostBreakScopeDepth;
       i--) {
    emitByte(OP_POP);
  }

  innermostBreakJumps[innermostBreakJumpCount++] = emitJump(OP_JUMP);
}

static void continueStatement() {
  if (innermostLoopStart == -1) {
    error("can't use 'continue' outside of a loop.");
  }
  consume(TOKEN_SEMICOLON, "expect ';' after 'continue'.");

  // discard any locals created inside the loop
  for (int i = current->localCount - 1;
       i >= 0 && current->locals[i].depth > innermostLoopScopeDepth;
       i--) {
    emitByte(OP_POP);
  }
  
  // jump to top of current innermost loop.
  emitLoop(innermostLoopStart);
}

static void ifStatement() {
  consume(TOKEN_LEFT_PAREN, "expect '(' after 'if'.");
  expression();
  consume(TOKEN_RIGHT_PAREN, "expect ')' after condition.");

  int thenJump = emitJump(OP_JUMP_IF_FALSE);
  emitByte(OP_POP); //将判断表达式的值弹出
  statement();

  int elseJump = emitJump(OP_JUMP); // elseJump结束边界 

  patchJump(thenJump); 
  emitByte(OP_POP);

  if (match(TOKEN_ELSE)) statement(); 
  patchJump(elseJump);
}

static void printStatement() {
  expression();
  consume(TOKEN_SEMICOLON, "expect ';' after value.");
  emitByte(OP_PRINT);
}

static void returnStatement() {
  if (current->type == TYPE_SCRIPT) {
    error("can't return from top-level code.");
  }
  if (match(TOKEN_SEMICOLON)) {
    emitReturn();
  } else {
    if (current->type == TYPE_INITIALIZER) {
      error("can't return a value from an initializer.");
    }
    expression();
    consume(TOKEN_SEMICOLON, "expect ';' after return value.");
    emitByte(OP_RETURN);
  }
}

static void switchStatement() {
  beginScope();
  
  consume(TOKEN_LEFT_PAREN, "expect '(' after'switch'.");
  expression();
  consume(TOKEN_RIGHT_PAREN, "expect ')' after value.");
  consume(TOKEN_LEFT_BRACE, "expect '{' before switch cases.");

  int state = 0; // 0: before all cases, 1: before default, 2:after default.

  // for 'break'
  int surroundingBreakScopeStart = innermostBreakScopeStart;
  int surroundingBreakScopeDepth = innermostBreakScopeDepth;
  int* surroundingBreakJumps = innermostBreakJumps;
  int surroundingBreakJumpCount = innermostBreakJumpCount;

  innermostBreakScopeStart = currentChunk()->count;
  innermostBreakScopeDepth = current->scopeDepth;
  innermostBreakJumps = ALLOCATE(int, MAX_BREAKS_PER_SCOPE);
  innermostBreakJumpCount = 0;

  int previousCaseSkip = -1;
  int fallThrough = -1;

  while (!match(TOKEN_RIGHT_BRACE) && !check(TOKEN_EOF)) {
    if (match(TOKEN_CASE) || match(TOKEN_DEFAULT)) {
      TokenType caseType = parser.previous.type;

      if(state == 2) {
        error("can't have cases after the default case.");
      }
      
      if (state == 1) {
        // if control reashes this point, it means the previous case matched
        // but did not break. this means we need a fallthrough jump.
        fallThrough = emitJump(OP_JUMP);
        
        // if that case didn't match, patch its condition to jump to jump here.
        patchJump(previousCaseSkip);
        emitByte(OP_POP);
      }

      if (caseType == TOKEN_CASE) {
        state = 1;
        // case, evaluate expression then compare
        emitByte(OP_DUP);
        expression();

        consume(TOKEN_COLON, "expect ':' after case value.");

        emitByte(OP_EQUAL);
        previousCaseSkip = emitJump(OP_JUMP_IF_FALSE);

        // pop the comparison result
        emitByte(OP_POP);
      } else {
        // defualt
        state =2;
        consume(TOKEN_COLON, "expect ':' after default.");
        previousCaseSkip = -1;
      }

      // both case and default must patch fallThrough if there is one
      if (fallThrough != -1) {
        patchJump(fallThrough);
        fallThrough = -1;
      }
    } else {
      // not case or default so this a statement inside the current case.
      if (state == 0) {
        error("cant't have statements before any case.");
      }
      statement();
    }
  }

  // if we ended without a default case, patch its condition jump.
  if (state == 1) {
    if (previousCaseSkip != -1) {
      patchJump(previousCaseSkip);
      emitByte(OP_POP);
      previousCaseSkip = -1;
    }
  }

  // patch all break jumps
  for (int i = 0; i < innermostBreakJumpCount; i++) {
    patchJump(innermostBreakJumps[i]);
  }

  // for 'break'
#ifdef DEBUG_TRACE_MEMORY_VERBOSE
  printf("compiler:switchStatement() FREE(int, %p) // innermostBreakJumps\n",
    (void*) innermostBreakJumps);
#endif
  FREE(int, innermostBreakJumps);

  innermostBreakScopeStart = surroundingBreakScopeStart;
  innermostBreakScopeDepth = surroundingBreakScopeDepth;
  innermostBreakJumps = surroundingBreakJumps;
  innermostBreakJumpCount = surroundingBreakJumpCount;

  emitByte(OP_POP); // the switch value.
  endScope();
}

static void whileStatement() {
  // save points
  int surroundingLoopStart = innermostLoopStart;
  int surroundingLoopScopeDepth = innermostLoopScopeDepth;
  // starting point
  innermostLoopStart = currentChunk()->count;
  innermostLoopScopeDepth = current->scopeDepth;

  // for 'break'
  int surroundingBreakScopeStart = innermostBreakScopeStart;
  int surroundingBreakScopeDepth = innermostBreakScopeDepth;
  int* surroundingBreakJumps = innermostBreakJumps;
  int surroundingBreakJumpCount = innermostBreakJumpCount;

  innermostBreakScopeStart = currentChunk()->count;
  innermostBreakScopeDepth = current->scopeDepth;
  innermostBreakJumps = ALLOCATE(int, MAX_BREAKS_PER_SCOPE);
  innermostBreakJumpCount = 0;

  int loopStart = currentChunk()->count;

  consume(TOKEN_LEFT_PAREN, "expect '(' after 'while'.");
  expression();
  consume(TOKEN_RIGHT_PAREN, "expect ')' after condition.");

  int exitJump = emitJump(OP_JUMP_IF_FALSE);

  emitByte(OP_POP);
  statement();

  emitLoop(loopStart);

  patchJump(exitJump);
  emitByte(OP_POP); //无论是否跳转, 判断条件的值最后会留在栈上

  // restore points (for continue)
  innermostLoopStart = surroundingLoopStart;
  innermostLoopScopeDepth = surroundingLoopScopeDepth;

  // patch break jump
  for (int i = 0; i < innermostBreakJumpCount; i++) {
    patchJump(innermostBreakJumps[i]);
  }
  
#ifdef DEBUG_TRACE_MEMORY_VERBOSE
  printf("compiler:forStatement() FREE(int, %p) // innermostBreakJumps\n",
    innermostBreakJumps);
#endif
  FREE(int, innermostBreakJumps);

  innermostBreakScopeStart = surroundingBreakScopeStart;
  innermostBreakScopeDepth = surroundingBreakScopeDepth;
  innermostBreakJumps = surroundingBreakJumps;
  innermostBreakJumpCount = surroundingBreakJumpCount;
}

static void synchronize() {
  parser.panicMode = false;

  while (parser.current.type != TOKEN_EOF) {
    if (parser.previous.type == TOKEN_SEMICOLON) return;

    switch (parser.current.type) {
    case TOKEN_CLASS:
    case TOKEN_FUN:
    case TOKEN_VAR:
    case TOKEN_FOR:
    case TOKEN_IF:
    case TOKEN_WHILE:
    case TOKEN_PRINT:
    case TOKEN_RETURN:
      return;
    default:
      // do nothing
      ;
    }
    advance();
  }
}

static void declaration() {
  if (match(TOKEN_CLASS)) { 
    classDeclaration();
  } else if(match(TOKEN_FUN)) {
    funDeclaration();
  } else if (match(TOKEN_VAR)) {
    varDeclaration();
  }  else {
    statement();
  }
  if (parser.panicMode) synchronize();
}

static void statement() {
  if (match(TOKEN_PRINT)) {
    printStatement();
  } else if (match(TOKEN_BREAK)) { 
    breakStatement();
  }else if (match(TOKEN_CONTINUE)) { 
    continueStatement();
  } else if (match(TOKEN_FOR)) { 
    forStatement();
  } else if (match(TOKEN_IF)) {
    ifStatement();
  } else if (match(TOKEN_SWITCH)) {
    switchStatement();
  } else if (match(TOKEN_RETURN)) {
    returnStatement();
  } else if (match(TOKEN_WHILE)) {
    whileStatement();
  } else if (match(TOKEN_LEFT_BRACE)) {
    beginScope();
    block();
    endScope();
  } else {
    // 解析表达式语句, eg: `a=2;`, `print a;`, `1;`等
    expressionStatement();
  }
}

ObjFunction* compile(const char* source) {
  initScanner(source);
  Compiler compiler;
  initCompiler(&compiler, TYPE_SCRIPT);

  parser.hadError = false;
  parser.panicMode = false;

  advance();

  while (!match(TOKEN_EOF)) {
    declaration();
  } 

  ObjFunction* function = endCompiler();
  return parser.hadError ? NULL : function;
}

void markCompilerRoots() {
  Compiler* compiler = current;
  while (compiler != NULL) {
    markObject((Obj*)compiler->function);
    compiler = compiler->enclosing;
  }
}

void printTokens(const char* source) {
  initScanner(source);
  int line = -1;
  for (;;) {
    Token token = scanToken();
    if (token.line != line) {
      printf("%4d ", token.line);
      line = token.line;
    } else {
      printf("   | ");
    }
    printf("%2d '%.*s'\n", token.type, token.length, token.start);

    if (token.type == TOKEN_EOF) {
      break;
    }
  }
}
