#ifndef clox_common_h
#define clox_common_h

#include <stdbool.h> // c99 bool type
#include <stddef.h> // size_t, NULL
#include <stdint.h> // uint8_t

#define NAN_BOXING
#define DEBUG_PRINT_CODE

#define DEBUG_STRESS_GC
#define DEBUG_LOG_GC

#define UINT8_COUNT (UINT8_MAX + 1)

#define MAX_BREAKS_PER_SCOPE 256

#undef DEBUG_PRINT_CODE
#undef DEBUG_STRESS_GC
#undef DEBUG_LOG_GC

#endif
