#ifndef clox_compiler_h
#define clox_compiler_h

#include "object.h"
#include "vm.h"
#include "scanner.h"

typedef struct {
  Token current;
  Token previous;
  bool hadError;
  bool panicMode;
} Parser;

extern Parser parser;

ObjFunction* compileModule(const char* source, struct ObjModule* module);
void markCompilerRoots();
void setCompilerSource(const char* filePath);

#endif

