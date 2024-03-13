#ifndef clox_object_h
#define clox_object_h

#include "common.h"
#include "chunk.h"
#include "table.h"
#include "value.h"

#define OBJ_TYPE(value)        (AS_OBJ(value)->type)

#define IS_BOUND_METHOD(value) isObjType(value, OBJ_BOUND_METHOD)
#define IS_CLASS(value)        isObjType(value, OBJ_CLASS)
#define IS_CLOSURE(value)      isObjType(value, OBJ_CLOSURE)
#define IS_FUNCTION(value)     isObjType(value, OBJ_FUNCTION)
#define IS_INSTANCE(value)     isObjType(value, OBJ_INSTANCE)
#define IS_NATIVE(value)       isObjType(value, OBJ_NATIVE)
#define IS_STRING(value)       isObjType(value, OBJ_STRING)
#define IS_LIST(value)         isObjType(value, OBJ_LIST)
#define IS_MAP(value)          isObjType(value, OBJ_MAP)

#define AS_BOUND_METHOD(value) ((ObjBoundMethod*)AS_OBJ(value))
#define AS_CLASS(value)        ((ObjClass*)AS_OBJ(value))
#define AS_CLOSURE(value)      ((ObjClosure*)AS_OBJ(value))
#define AS_FUNCTION(value)     ((ObjFunction*)AS_OBJ(value))
#define AS_INSTANCE(value)     ((ObjInstance*)AS_OBJ(value))
#define AS_NATIVE(value)       (((ObjNative*)AS_OBJ(value))->function)
#define AS_STRING(value)       ((ObjString*)AS_OBJ(value))
#define AS_CSTRING(value)      (((ObjString*)AS_OBJ(value))->chars)
#define AS_LIST(value)         ((ObjList*)AS_OBJ(value))
#define AS_MAP(value)          ((ObjMap*)AS_OBJ(value))

typedef enum {
  OBJ_BOUND_METHOD,
  OBJ_CLASS,
  OBJ_CLOSURE,
  OBJ_FUNCTION,
  OBJ_INSTANCE,
  OBJ_NATIVE,
  OBJ_STRING,
  OBJ_LIST,
  OBJ_MAP,
  OBJ_UPVALUE,
} ObjType;

struct Obj {
  ObjType type;
  bool isMarked;
  struct Obj* next;
};

typedef struct {
  Obj obj;
  int arity;
  int upvalueCount; // 放在ObjFunction里面,因为要在runtime时用到
  Chunk chunk;
  ObjString* name;
} ObjFunction;

// 为了在native里面可以调用runtimeError, 所以设置一个外部errRet用来传送业务错误来终止native调用,
// 原因在于runtimerError里面会重置stack以及frameCount=0, 导致下面代码中frame取值出现段错误.
// case OP_INVOKE: {
//  ...
//  if (!invoke(method, argCount)) {
//    return INTERPRET_RUNTIME_ERROR;
//  }
//  frame = &vm.frames[vm.frameCount - 1];
//  ...
// }
// 当然也可以将返回类型Value改为bool, 在native内部使push操作来作为native的执行结果
typedef Value (*NativeFn)(int argCount, Value* args, int* errRet);

typedef struct {
  Obj obj;
  NativeFn function;
  int arity;
} ObjNative;


struct ObjString {
  Obj obj;
  int length;
  char* chars;
  uint32_t hash;
};

typedef struct {
  Obj obj;
  ValueArray array;
} ObjList;

typedef struct {
  Obj obj;
  Table table;
} ObjMap;

typedef struct ObjUpvalue {
  Obj obj;
  Value* location;
  Value closed;
  struct ObjUpvalue* next;
} ObjUpvalue;


typedef struct {
  Obj obj;
  ObjFunction* function;
  ObjUpvalue** upvalues;
  int upvalueCount;
} ObjClosure;

typedef struct {
  Obj obj;
  ObjString* name;
  Table methods;
} ObjClass;

typedef struct {
  Obj obj;
  ObjClass* klass;
  Table fields;
} ObjInstance;

typedef struct {
  Obj obj;
  Value receiver;
  //ObjClosure* method;
  Obj* method; //使用基类, 因为有methodNative
} ObjBoundMethod;

ObjBoundMethod* newBoundMethod(Value receiver, Obj* method);
ObjClass* newClass(ObjString* name);
ObjClosure* newClosure(ObjFunction* function);
ObjFunction* newFunction();
ObjInstance* newInstance(ObjClass* klass);
ObjNative* newNative(NativeFn function, int arity);
ObjString* takeString(char* chars, int length);
ObjString* copyString(const char* chars, int length);
ObjList* newList();
void copyList(ObjList* list, Value* values, int length);
ObjMap* newMap();
ObjUpvalue* newUpvalue(Value* slot);
void printObject(Value value);
void objTypeName(ObjType type, char* out);

static inline bool isObjType(Value value, ObjType type) {
  return IS_OBJ(value) && AS_OBJ(value)->type == type;
}

#endif
