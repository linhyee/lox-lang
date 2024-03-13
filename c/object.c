#include <stdio.h>
#include <string.h>

#include "memory.h"
#include "object.h"
#include "table.h"
#include "value.h"
#include "vm.h"
#include "utf8.h"

#define ALLOCATE_OBJ(type, objectType) \
  (type*)allocateObject(sizeof(type), objectType)

static Obj* allocateObject(size_t size, ObjType type) {
  Obj* object = (Obj*)reallocate(NULL, 0, size);
  object->type = type;
  object->isMarked = false;

  object->next = vm.objects;
  vm.objects = object;

#ifdef DEBUG_LOG_GC
  char buf[128] = {0};
  objTypeName(type, buf);
  printf("%p allocate %zu for %s\n", (void*)object, size, buf);
#endif
  return object;
}

ObjBoundMethod* newBoundMethod(Value receiver, 
    Obj* method) {
  ObjBoundMethod* bound = 
    ALLOCATE_OBJ(ObjBoundMethod, OBJ_BOUND_METHOD);
  bound->receiver = receiver;
  bound->method = method;
  return bound;
}

ObjClass* newClass(ObjString* name) {
  ObjClass* klass = ALLOCATE_OBJ(ObjClass, OBJ_CLASS);
  klass->name = name;
  initTable(&klass->methods);
  return klass;
}

ObjClosure* newClosure(ObjFunction* function) {
  ObjUpvalue** upvalues = ALLOCATE(ObjUpvalue*,
   function->upvalueCount);
  for (int i = 0; i < function->upvalueCount; i++) {
    upvalues[i] = NULL;
  } 
  ObjClosure* closure = ALLOCATE_OBJ(ObjClosure, OBJ_CLOSURE);
  closure->function = function;
  closure->upvalues = upvalues;
  closure->upvalueCount = function->upvalueCount;
  return closure;
}

ObjFunction* newFunction() {
  ObjFunction* function = ALLOCATE_OBJ(ObjFunction, OBJ_FUNCTION);

  function->arity = 0;
  function->upvalueCount = 0;
  function->name = NULL;
  initChunk(&function->chunk);
  return function;
}

ObjInstance* newInstance(ObjClass* klass) {
  ObjInstance* instance = ALLOCATE_OBJ(ObjInstance, OBJ_INSTANCE);
  instance->klass = klass;
  initTable(&instance->fields);
  return instance;
}

ObjNative* newNative(NativeFn function, int arity) {
  ObjNative* native = ALLOCATE_OBJ(ObjNative, OBJ_NATIVE);
  native->function = function;
  native->arity = arity;
  return native;
}

static ObjString* allocateString(char* chars, int length, uint32_t hash) {
  ObjString* string = ALLOCATE_OBJ(ObjString, OBJ_STRING);
  string->length = length;
  string->chars = chars;
  string->hash = hash;

  push(OBJ_VAL(string)); // for collector
  tableSet(&vm.strings, string, NIL_VAL);
  pop();

  return string;
}

static uint32_t hashString(const char* key, int length) { 
  uint32_t hash = 2166136261u;
  for (int i = 0; i < length; i++) {
    hash ^= (uint8_t)key[i];
    hash *= 16777619;
  }
  return hash;
}

ObjString* takeString(char* chars, int length) {
  uint32_t hash = hashString(chars, length);
  ObjString* interned = tableFindString(&vm.strings, chars, length, hash);

  if (interned != NULL) {
    FREE_ARRAY(char, chars, length + 1);
    return interned;
  }
  return allocateString(chars, length, hash);
}

ObjString* copyString(const char* chars, int length) {
  uint32_t hash = hashString(chars, length);
  ObjString* interned = tableFindString(&vm.strings, chars, length, hash);
  if (interned != NULL) return interned;

  char* heapChars = ALLOCATE(char, length + 1);
  memcpy(heapChars, chars, length);
  heapChars[length] = '\0';

  return allocateString(heapChars, length, hash);
}

ObjList* newList() {
  ObjList* list = ALLOCATE_OBJ(ObjList, OBJ_LIST);
  initValueArray(&list->array);
  return list;
}

void copyList(ObjList* list, Value* values, int length) { 
  if (length <= 0) {
    return;
  }
  list->array.values = ALLOCATE(Value, length);
  list->array.count = length;
  list->array.capacity = length;
  memcpy(list->array.values, values, length * sizeof(Value));
}

ObjMap* newMap() {
  ObjMap* map = ALLOCATE_OBJ(ObjMap, OBJ_MAP);
  initTable(&map->table);
  return map;
}

ObjUpvalue* newUpvalue(Value* slot) {
  ObjUpvalue* upvalue = ALLOCATE_OBJ(ObjUpvalue, OBJ_UPVALUE);
  upvalue->location = slot;
  upvalue->closed = NIL_VAL;
  upvalue->next = NULL;
  return upvalue;
}

static void printList(ObjList* list) {
  printf("[");
  for (int i=0; i < list->array.count; i++) {
    printValue(list->array.values[i]);
    if (i != list->array.count - 1) {
      printf(",");
    }
  }
  printf("]");
}

static void printMap(ObjMap* map) {
  printf("{");
  int first = 1;
  for (int i=0; i < map->table.capacity; i++) {
    Entry *entry = &map->table.entries[i];
    if (entry->key == NULL) {
      continue;
    }
    if (first) {
      first = 0;
    } else {
      printf(", ");
    }
    printf("%.*s", entry->key->length, entry->key->chars);
    printf(": ");
    printValue(entry->value);
  }
  printf("}");
}

static void printFunction(ObjFunction* function) {
  if (function->name == NULL) {
    printf("<script>");
    return;
  }
  printf("<fn %s>", function->name->chars);
}

void printObject(Value value) {
  switch (OBJ_TYPE(value)) {
  case OBJ_BOUND_METHOD:
    printObject(OBJ_VAL(AS_BOUND_METHOD(value)->method));
    break;
  case OBJ_CLASS:
    printf("%s", AS_CLASS(value)->name->chars);
    break;
  case OBJ_CLOSURE:
    printFunction(AS_CLOSURE(value)->function);
    break;
  case OBJ_FUNCTION:
    printFunction(AS_FUNCTION(value));
    break;
  case OBJ_INSTANCE:
    printf("<%s instance>", 
      AS_INSTANCE(value)->klass->name->chars);
    break;
  case OBJ_NATIVE:
    printf("<native fn>");
    break;
  case OBJ_STRING:
    printf("%s", AS_CSTRING(value));
    break;
  case OBJ_LIST:
    printList(AS_LIST(value));
    break;
  case OBJ_MAP:
    printMap(AS_MAP(value));
    break;
  case OBJ_UPVALUE:
    printf("upvalue");
    break;
  }
}

void objTypeName(ObjType type, char* out) {
  switch (type) {
  case OBJ_BOUND_METHOD:
    strcpy(out, "bound_method");
    break;
  case OBJ_CLASS:
    strcpy(out, "class");
    break;
  case OBJ_CLOSURE:
    strcpy(out, "closure");
    break;
  case OBJ_FUNCTION:
    strcpy(out, "function");
    break;
  case OBJ_INSTANCE:
    strcpy(out, "instance");
    break;
  case OBJ_NATIVE:
    strcpy(out, "native-fn");
    break;
  case OBJ_STRING:
    strcpy(out, "string");
    break;
  case OBJ_LIST:
    strcpy(out, "list");
    break;
  case OBJ_MAP:
    strcpy(out, "map");
    break;
  case OBJ_UPVALUE:
    strcpy(out, "upvalue");
    break;
  }
}
