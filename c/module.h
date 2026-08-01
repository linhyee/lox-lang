#ifndef clox_module_h
#define clox_module_h

#include "object.h"
#include "table.h"
#include "value.h"

/*
 * ModuleLoader tracks loaded modules, resolves import paths, and
 * keeps the current import chain for cycle detection.
 */
typedef struct {
  // Canonical absolute path -> ObjModule*
  Table loadedModules;
  // Reserved for normalized path memoization.
  Table resolvedPaths;
  // Directory used for resolving relative import paths.
  const char* baseDir;
  // Active import stack for cycle diagnostics.
  ValueArray currentImportStack;
} ModuleLoader;

extern ModuleLoader moduleLoader;

// Initialize module loader state.
void initModuleLoader();
// Free all module loader resources.
void freeModuleLoader();
// Update the current base directory from a source file path.
void setModuleBaseDir(const char* filePath);
// Load/compile a module or return it from cache.
ObjModule* loadModule(const char* modulePath, int line);
// Resolve modulePath against baseDir into a canonical absolute path.
char* resolveModulePath(const char* modulePath, const char* baseDir);
// Read an entire file into memory; caller owns the returned buffer.
char* readFileContent(const char* path);

#endif
