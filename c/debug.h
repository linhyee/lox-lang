#ifndef clox_debug_h
#define clox_debug_h

#include <stdint.h>

#define DEBUG_TRACE_EXECUTION

#include "chunk.h"

void disassembleChunk(Chunk* chunk, const char* name);
int disassembleInstruction(Chunk* chunk, int offset);

#undef DEBUG_TRACE_EXECUTION

#endif
