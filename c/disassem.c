#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "common.h"
#include "chunk.h"
#include "debug.h"
#include "vm.h"
#include "compiler.h"
#include "module.h"
#include "object.h"

static char* readFile(const char* path) {
  FILE* file = fopen(path, "rb");
  if (file == NULL) {
    fprintf(stderr, "error: could not open file \"%s\".\n", path);
    return NULL;
  }

  fseek(file, 0L, SEEK_END);
  size_t fileSize = ftell(file);
  rewind(file);

  char* buffer = (char*)malloc(fileSize + 1);
  if (buffer == NULL) {
    fprintf(stderr, "error: not enough memory to read \"%s\".\n", path);
    fclose(file);
    return NULL;
  }

  size_t bytesRead = fread(buffer, sizeof(char), fileSize, file);
  if (bytesRead < fileSize) {
    fprintf(stderr, "error: could not read file \"%s\".\n", path);
    free(buffer);
    fclose(file);
    return NULL;
  }

  buffer[bytesRead] = '\0';
  fclose(file);
  return buffer;
}

static void disassembleFunction(ObjFunction* function, int depth) {
  if (function == NULL) return;
  
  char indent[256] = "";
  for (int i = 0; i < depth * 2 && i < 250; i++) {
    indent[i] = ' ';
  }
  indent[depth * 2] = '\0';
  
  printf("%s== %s ==\n", indent, 
         function->name ? function->name->chars : "<script>");
  disassembleChunk(&function->chunk, NULL);
  printf("\n");
  
  // Disassemble nested functions (stored in constants)
  for (int i = 0; i < function->chunk.constants.count; i++) {
    Value constant = function->chunk.constants.values[i];
    if (IS_OBJ(constant) && OBJ_TYPE(constant) == OBJ_FUNCTION) {
      ObjFunction* nested = AS_FUNCTION(constant);
      disassembleFunction(nested, depth + 1);
    }
  }
}

static void printUsage(const char* progName) {
  fprintf(stderr, "usage: %s <lox-file>\n", progName);
  fprintf(stderr, "  Disassemble compiled bytecode from a .lox file\n");
}

int main(int argc, const char* argv[]) {
  if (argc != 2) {
    printUsage(argv[0]);
    return 1;
  }

  const char* filePath = argv[1];
  
  initVM();
  initModuleLoader();
  setCompilerSource(filePath);
  
  char* source = readFile(filePath);
  if (source == NULL) {
    freeModuleLoader();
    freeVM();
    return 1;
  }

  // Create a temporary module for compilation
  ObjString* pathStr = copyString(filePath, (int)strlen(filePath));
  ObjModule* module = newModule(pathStr);
  
  ObjFunction* function = compileModule(source, module);
  
  if (function == NULL) {
    fprintf(stderr, "error: compilation failed\n");
    free(source);
    freeModuleLoader();
    freeVM();
    return 1;
  }
  
  printf("=== Bytecode Disassembly: %s ===\n\n", filePath);
  disassembleFunction(function, 0);
  
  free(source);
  freeModuleLoader();
  freeVM();
  return 0;
}
