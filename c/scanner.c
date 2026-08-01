#include <stdio.h>

#include "utf8.h"
#include "common.h"
#include "scanner.h"

typedef utf8_int32_t rune;

Scanner scanner;

void initScanner(const char* source) {
  scanner.start = source;
  scanner.current = source;
  scanner.line = 1;
}

static bool isAlpha(rune c) {
  return (c >= 'a' && c <= 'z') || (c >= 'A' && c <='Z') || c == '_';
}

static bool isDigit(rune c) {
  return c >= '0' && c <= '9';
}

static bool isEol(rune c) {
  return c == '\n' || c == '\r';
}

static bool isAtEnd() {
  return *scanner.current == '\0';
}

static rune advance() {
  rune rune;
  scanner.current = (const char*)utf8codepoint(scanner.current, &rune);
  return rune;
}

static rune peek() {
  const char* p = scanner.current;
  rune rune;
  utf8codepoint(p, &rune);
  return rune;
}

static rune peekNext() {
  if (isAtEnd()) return '\0';

  rune rune;
  const char* p = scanner.current;
  utf8codepoint(p + utf8codepointcalcsize(p), &rune);

  return rune;
}

static bool match(rune expected) {
  if (isAtEnd()) return false;

  const char* p= scanner.current;
  rune rune;
  p = utf8codepoint(p, &rune);

  if (rune != expected) return false;

  scanner.current = p;
  return true;
}

static Token makeToken(TokenType_ type) {
  Token token;
  token.type = type;
  token.start = scanner.start;
  token.length = (int) (scanner.current - scanner.start);
  token.line = scanner.line;

  return token;
}

static Token errorToken(const char* message) {
  Token token;
  token.type = TOKEN_ERROR;
  token.start = message;
  token.length = (int)utf8size(message) - 1;
  token.line = scanner.line; 

  return token;
}

static void skipWhitespace() {
  for (;;) {
    rune c = peek();
    switch (c) {
    case ' ':
    case '\r':
    case '\t':
      advance();
      break;

    case'\n':
      scanner.line++;
      advance();
      break;

    case '/':
      if (peekNext() == '/') {
        // a comments goes until the end of the line.
        while (peek() != '\n' && !isAtEnd()) advance();
      } else if (peekNext() == '*') {
        // a block comment goes until '*/'
        advance(); // slash
        advance(); // star

        // scan for */ while ignoring everything else
        while ((peek() != '*' || peekNext() != '/') && !isAtEnd()) {
          if (isEol(peek())) scanner.line++;
          advance();
        }

        advance(); // star
        advance(); // slash
      } else {
        return;
      }
      break;

    default:
      return;
    }
  }
}

static TokenType_ checkKeyword(int start, int length, 
    const char* rest, TokenType_ type) {
  if (scanner.current - scanner.start == start + length &&
    utf8ncmp(scanner.start + start, rest, length) == 0) {
    return type;
  }
  return TOKEN_IDENTIFIER;
}

static TokenType_ identifierType() {
  const char* p = scanner.start;
  rune c;
  p = utf8codepoint(p, &c);
  switch (c) {
  case 'a': 
    if (scanner.current - scanner.start > 1) {
      utf8codepoint(p, &c);
      switch (c) {
      case 'n': return checkKeyword(2, 1, "d", TOKEN_AND);
      case 's': return checkKeyword(2, 0, "", TOKEN_AS);
      }
    }
    break;
  case 'b': return checkKeyword(1, 4, "reak", TOKEN_BREAK);
  case 'd': return checkKeyword(1, 6, "efault", TOKEN_DEFAULT);
  case 'c':
    if (scanner.current - scanner.start > 1) {
      utf8codepoint(p, &c);
      switch (c) {
      case 'l': return checkKeyword(2, 3, "ass", TOKEN_CLASS); 
      case 'o': return checkKeyword(2, 6, "ntinue", TOKEN_CONTINUE);
      case 'a': return checkKeyword(2, 2, "se", TOKEN_CASE);
      } 
    }
    break;
  case 'e': 
    if (scanner.current - scanner.start > 1) {
      utf8codepoint(p, &c);
      switch (c) {
      case 'l': return checkKeyword(2, 2, "se", TOKEN_ELSE);
      case 'x': return checkKeyword(2, 4, "port", TOKEN_EXPORT);
      }
    }
    break;
  case 'f':
    if (scanner.current - scanner.start > 1) {
      utf8codepoint(p, &c);
      switch (c) {
      case 'a': return checkKeyword(2, 3, "lse", TOKEN_FALSE);
      case 'o': return checkKeyword(2, 1, "r", TOKEN_FOR);
      case 'u': return checkKeyword(2, 1, "n", TOKEN_FUN);
      }
    }
    break;
  case 'i': 
    if (scanner.current - scanner.start > 1) {
      utf8codepoint(p, &c);
      rune second = c;
      if (second == 'f' && scanner.current - scanner.start == 2) {
        return TOKEN_IF;
      }
      if (second == 'm' && scanner.current - scanner.start == 6) {
        return checkKeyword(2, 4, "port", TOKEN_IMPORT);
      }
    }
    return TOKEN_IDENTIFIER;
  case 'n': return checkKeyword(1, 2, "il", TOKEN_NIL);
  case 'o': return checkKeyword(1, 1, "r", TOKEN_OR);
  case 'p': return checkKeyword(1, 4, "rint", TOKEN_PRINT);
  case 'r': return checkKeyword(1, 5, "eturn", TOKEN_RETURN);
  case 's':
    if ( scanner.current - scanner.start > 1) {
      utf8codepoint(p, &c);
      switch (c) {
      case 'w':
        return checkKeyword(2, 4, "itch", TOKEN_SWITCH);
      case 'u':
        return checkKeyword(2, 3, "per", TOKEN_SUPER);
      }
    }
  case 't':
    if (scanner.current - scanner.start > 1) {
      utf8codepoint(p, &c);
      switch (c) {
      case 'h': return checkKeyword(2, 2, "is", TOKEN_THIS);
      case 'r': return checkKeyword(2, 2, "ue", TOKEN_TRUE);
      }
    }
    break;
  case 'v': return checkKeyword(1, 2, "ar", TOKEN_VAR);
  case 'w': return checkKeyword(1, 4, "hile", TOKEN_WHILE);
  }

  return TOKEN_IDENTIFIER;
}

static Token identifier() {
  while (isAlpha(peek()) || isDigit(peek())) advance();
  return makeToken(identifierType());
}

static Token number() {
  while (isDigit(peek())) advance();


  // look for a fractional part.
  if (peek() == '.' && isDigit(peekNext())) {
    // consume the '.'.
    advance();

    while (isDigit(peek())) advance();
  }

  return makeToken(TOKEN_NUMBER);
}

static Token string() {
  while (peek() != '"' && !isAtEnd()) {
    if (isEol(peek())) scanner.line++;
    if (peek() == '\\' && peekNext() == '\\') {
      // double backslashes will be interpreted as a single backslash later
      // skip them so they don't interfere with the next test
      advance();
      advance();
      continue;
    } 
    if (peek() == '\\' && peekNext() == '"') {
      // escaped double quotes must not terminate the string
      advance();
      advance();
      continue;
    }
    // as it going on without causing problems
    advance();
  }

  if (isAtEnd()) return errorToken("unterminated string.");

  // the closing quote.
  advance();
  return makeToken(TOKEN_STRING);
}

Token scanToken() {
  skipWhitespace();

  scanner.start = scanner.current;
  if (isAtEnd()) {
    return makeToken(TOKEN_EOF);
  }

  rune c = advance();
  if (isAlpha(c)) return identifier();
  if (isDigit(c)) return number();

  switch (c) {
  case '(': return makeToken(TOKEN_LEFT_PAREN);
  case ')': return makeToken(TOKEN_RIGHT_PAREN);
  case '[': return makeToken(TOKEN_LEFT_BRACKET);
  case ']': return makeToken(TOKEN_RIGHT_BRACKET);
  case '{': return makeToken(TOKEN_LEFT_BRACE);
  case '}': return makeToken(TOKEN_RIGHT_BRACE);
  case ':': return makeToken(TOKEN_COLON);
  case ';': return makeToken(TOKEN_SEMICOLON);
  case ',': return makeToken(TOKEN_COMMA);
  case '.': return makeToken(TOKEN_DOT);
  case '-': return makeToken(match('-') ? TOKEN_MINUS_MINUS   : TOKEN_MINUS);
  case '+': return makeToken(match('+') ? TOKEN_PLUS_PLUS     : TOKEN_PLUS);
  case '/': return makeToken(TOKEN_SLASH);
  case '*': return makeToken(TOKEN_STAR);
  case '!': return makeToken(match('=') ? TOKEN_BANG_EQUAL    : TOKEN_BANG);
  case '=': return makeToken(match('=') ? TOKEN_EQUAL_EQUAL   : TOKEN_EQUAL);
  case '<': return makeToken(match('=') ? TOKEN_LESS_EQUAL    : TOKEN_LESS);
  case '>': return makeToken(match('=') ? TOKEN_GREATER_EQUAL : TOKEN_GREATER);
  case '"': return string();
  }

  return errorToken("unexpected character.");
}
