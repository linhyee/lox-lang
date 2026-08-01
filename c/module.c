#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
  #include <windows.h>
  #include <direct.h>
  #define MAX_PATH_LEN MAX_PATH
#else
  #include <unistd.h>
  #include <libgen.h>
  #define MAX_PATH_LEN 4096
#endif

#include "common.h"
#include "module.h"
#include "memory.h"
#include "vm.h"
#include "compiler.h"

ModuleLoader moduleLoader;

static int findImportPathIndex(ObjString* path) {
  for (int i = 0; i < moduleLoader.currentImportStack.count; i++) {
    if (valuesEqual(moduleLoader.currentImportStack.values[i], OBJ_VAL(path))) {
      return i;
    }
  }
  return -1;
}

static void printImportCycle(int line, ObjString* pathStr) {
  int start = findImportPathIndex(pathStr);
  fprintf(stderr, "[line %d] error: Circular import detected: ", line);
  if (start >= 0) {
    for (int i = start; i < moduleLoader.currentImportStack.count; i++) {
      fprintf(stderr, "%s -> ", AS_STRING(moduleLoader.currentImportStack.values[i])->chars);
    }
    fprintf(stderr, "%s.\n", pathStr->chars);
    return;
  }

  // Path is currently loading but not in stack (e.g. main module).
  fprintf(stderr, "%s -> ", pathStr->chars);
  for (int i = 0; i < moduleLoader.currentImportStack.count; i++) {
    fprintf(stderr, "%s -> ", AS_STRING(moduleLoader.currentImportStack.values[i])->chars);
  }
  fprintf(stderr, "%s.\n", pathStr->chars);
}

void initModuleLoader() {
  initTable(&moduleLoader.loadedModules);
  initTable(&moduleLoader.resolvedPaths);
  moduleLoader.baseDir = NULL;
  initValueArray(&moduleLoader.currentImportStack);
}

void freeModuleLoader() {
  freeTable(&moduleLoader.loadedModules);
  freeTable(&moduleLoader.resolvedPaths);
  if (moduleLoader.baseDir != NULL) {
    free((void*)moduleLoader.baseDir);
  }
  freeValueArray(&moduleLoader.currentImportStack);
}

void setModuleBaseDir(const char* filePath) {
  if (moduleLoader.baseDir != NULL) {
    free((void*)moduleLoader.baseDir);
    moduleLoader.baseDir = NULL;
  }
  if (filePath == NULL) return;

  char* pathCopy = strdup(filePath);
  char* lastSlash = strrchr(pathCopy, '/');
  char* lastBackslash = strrchr(pathCopy, '\\');
  char* separator = (lastSlash > lastBackslash) ? lastSlash : lastBackslash;

  if (separator != NULL) {
    *separator = '\0';
    moduleLoader.baseDir = strdup(pathCopy);
  } else {
    moduleLoader.baseDir = strdup(".");
  }
  free(pathCopy);
}

char* readFileContent(const char* path) {
  FILE* file = fopen(path, "rb");
  if (file == NULL) return NULL;

  fseek(file, 0L, SEEK_END);
  size_t fileSize = ftell(file);
  rewind(file);

  char* buffer = (char*)malloc(fileSize + 1);
  if (buffer == NULL) {
    fclose(file);
    return NULL;
  }

  size_t bytesRead = fread(buffer, sizeof(char), fileSize, file);
  if (bytesRead < fileSize) {
    free(buffer);
    fclose(file);
    return NULL;
  }

  buffer[bytesRead] = '\0';
  fclose(file);
  return buffer;
}

char* resolveModulePath(const char* modulePath, const char* baseDir) {
  char resolved[MAX_PATH_LEN];
  
#ifdef _WIN32
  // Check for absolute path on Windows (e.g., C:\... or /...)
  if ((modulePath[0] != '\0' && modulePath[1] == ':') || modulePath[0] == '\\' || modulePath[0] == '/') {
    strncpy(resolved, modulePath, MAX_PATH_LEN);
  } else {
    snprintf(resolved, MAX_PATH_LEN, "%s/%s", baseDir ? baseDir : ".", modulePath);
  }
  char* normalized = (char*)malloc(MAX_PATH_LEN);
  if (_fullpath(normalized, resolved, MAX_PATH_LEN) == NULL) {
    free(normalized);
    return NULL;
  }
#else
  // Check for absolute path on Unix
  if (modulePath[0] == '/') {
    strncpy(resolved, modulePath, MAX_PATH_LEN);
  } else {
    snprintf(resolved, MAX_PATH_LEN, "%s/%s", baseDir ? baseDir : ".", modulePath);
  }
  char* normalized = (char*)malloc(MAX_PATH_LEN);
  // Note: realpath requires the file to exist. If it doesn't, it returns NULL.
  if (realpath(resolved, normalized) == NULL) {
    // Fallback to simple normalization if file doesn't exist yet (for testing or future use)
    strncpy(normalized, resolved, MAX_PATH_LEN);
  }
#endif

  // Uniform separators
  for (char* p = normalized; *p != '\0'; p++) {
    if (*p == '\\') *p = '/';
  }

  return normalized;
}

ObjModule* loadModule(const char* modulePath, int line) {
  ObjModule* result = NULL;
  bool success = false;
  bool pathRooted = false;
  bool moduleRooted = false;
  bool importStackPushed = false;
  char* source = NULL;
  char* absolutePath = resolveModulePath(modulePath, moduleLoader.baseDir);
  if (absolutePath == NULL) {
    fprintf(stderr, "[line %d] error: Could not resolve path '%s'.\n", line, modulePath);
    return NULL;
  }

  ObjString* pathStr = copyString(absolutePath, (int)strlen(absolutePath));
  push(OBJ_VAL(pathStr));
  pathRooted = true;

  Value cached;
  if (tableGet(&moduleLoader.loadedModules, pathStr, &cached)) {
    ObjModule* cachedModule = AS_MODULE(cached);
    if (cachedModule->state == MODULE_FAILED) {
      fprintf(stderr, "[line %d] error: Failed to import '%s'.\n", line, absolutePath);
      goto cleanup;
    }
    if (cachedModule->state == MODULE_LOADING || cachedModule->state == MODULE_EXECUTING) {
      printImportCycle(line, pathStr);
      goto cleanup;
    }
    result = cachedModule;
    success = true;
    goto cleanup;
  }

  if (findImportPathIndex(pathStr) != -1) {
    printImportCycle(line, pathStr);
    goto cleanup;
  }
  writeValueArray(&moduleLoader.currentImportStack, OBJ_VAL(pathStr));
  importStackPushed = true;

  ObjModule* module = newModule(pathStr);
  push(OBJ_VAL(module));
  moduleRooted = true;
  module->state = MODULE_LOADING;
  tableSet(&moduleLoader.loadedModules, pathStr, OBJ_VAL(module));

  source = readFileContent(absolutePath);
  if (source == NULL) {
    fprintf(stderr, "[line %d] error: Could not read file '%s'.\n", line, absolutePath);
    module->state = MODULE_FAILED;
    goto cleanup;
  }

  const char* savedBaseDir = moduleLoader.baseDir;
  moduleLoader.baseDir = NULL;
  setModuleBaseDir(absolutePath);

  Scanner savedScanner = scanner;
  Parser savedParser = parser;

  ObjFunction* function = compileModule(source, module);

  scanner = savedScanner;
  parser = savedParser;
  if (moduleLoader.baseDir) free((void*)moduleLoader.baseDir);
  moduleLoader.baseDir = savedBaseDir;
  free(source);
  source = NULL;

  if (function == NULL) {
    module->state = MODULE_FAILED;
    goto cleanup;
  }

  push(OBJ_VAL(function));
  ObjClosure* closure = newClosure(function);
  closure->module = module;
  closure->env = &module->env;
  module->closure = closure;
  pop();

  module->state = MODULE_READY;
  result = module;
  success = true;

cleanup:
  if (source != NULL) free(source);
  if (moduleRooted) pop();
  if (importStackPushed) popValueArray(&moduleLoader.currentImportStack);
  if (pathRooted) pop();
  free(absolutePath);
  return success ? result : NULL;
}
