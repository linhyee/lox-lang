#include <stdio.h>
#include <string.h>
#include "value.h"
#include "memory.h"
#include "object.h"

#define GROW_UP_ARRAY(array) do { \
  if (array->capacity < array->count + 1) { \
    int oldCapacity = array->capacity; \
    array->capacity = GROW_CAPACITY(oldCapacity); \
    array->values = GROW_ARRAY(Value, array->values, oldCapacity, array->capacity); \
  } \
} while(0);

void initValueArray(ValueArray* array) {
  array->values = NULL;
  array->capacity = 0;
  array->count = 0;
}

void writeValueArray(ValueArray* array, Value value) {
  GROW_UP_ARRAY(array);

  array->values[array->count] = value;
  array->count++;
}

int insertValueArray(ValueArray* array, int index, Value value) {
  if (index < 0 || index >= array->count) {
    return -1;
  }
  GROW_UP_ARRAY(array);
  memmove(array->values + index + 1, array->values + index, (array->count - index) * sizeof(Value));
  array->values[index] = value;
  array->count++;
  return 0;
}

int removeValueArray(ValueArray* array, int index, Value* out) {
  if (index < 0 || index >= array->count) {
    return -1;
  }
  *out = array->values[index];
  memmove(array->values + index, array->values + index + 1, (array->count - index - 1) *sizeof(Value));
  array->count--;
  return 0;
}

int findInValueArray(ValueArray* array, Value value) {
  int i;
  for (i = 0; i < array->count; ++i) {
    if (valuesEqual(value, array->values[i])) {
      return i;
    }
  }
  return -1;
}

void freeValueArray(ValueArray* array) {
  FREE_ARRAY(Value, array->values, array->capacity);
  initValueArray(array);
}

void printValue(Value value) {
#ifdef NAN_BOXING
  if (IS_BOOL(value)) {
    printf(AS_BOOL(value) ? "true" : "false");
  } else if (IS_NIL(value)) {
    printf("nil");
  } else if (IS_NUMBER(value)) {
    printf("%g", AS_NUMBER(value));
  } else if (IS_OBJ(value)) {
    printObject(value);
  }
#else
  switch (value.type) {
  case VAL_BOOL:
    printf(AS_BOOL(value) ? "true" : "false");
    break;
  case VAL_NIL: printf("nil"); break;
  case VAL_NUMBER: printf("%g", AS_NUMBER(value)); break;
  case VAL_OBJ: printObject(value); break;
  }
#endif
}

bool valuesEqual(Value a, Value b) {
#ifdef NAN_BOXING
  if (IS_NUMBER(a) && IS_NUMBER(b)) {
    return AS_NUMBER(a) == AS_NUMBER(b);
  }
  return a == b;
#else
  if (a.type != b.type ) return false;

  switch (a.type) {
  case VAL_BOOL:    return AS_BOOL(a) == AS_BOOL(b);
  case VAL_NIL:     return true;
  case VAL_NUMBER:  return AS_NUMBER(a) == AS_NUMBER(b);
  case VAL_OBJ:     return AS_OBJ(a) == AS_OBJ(b);
  default:          return false; // unreachable.
  }
#endif
}
