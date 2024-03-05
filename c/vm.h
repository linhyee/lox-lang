#ifndef clox_vm_h
#define clox_vm_h

#include "object.h"
#include "table.h"

#define FRAMES_MAX 64
// 栈的长度=栈帧数*指令上限
#define STACK_MAX  (FRAMES_MAX * UINT8_COUNT)

// 函数栈帧
typedef struct {
  ObjClosure* closure;
  uint8_t* ip;
  // the slots field points into the VM’s value stack 
  // at the first slot that this function can use
  Value* slots;
} CallFrame;

typedef struct {
  CallFrame frames[FRAMES_MAX];
  int frameCount;

  Value stack[STACK_MAX];
  Value* stackTop;
  Table globals;
  Table strings;
  ObjString* initString;
  ObjUpvalue* openUpvalues;

  size_t bytesAllocated;
  size_t nextGC;

  Obj* objects;
  int grayCount;
  int grayCapacity;
  Obj** grayStack;

  ObjClass* listClass;
} VM;

typedef enum {
  INTERPRET_OK,
  INTERPRET_COMPILE_ERROR,
  INTERPRET_RUNTIME_ERROR
} InterpretResult;

extern VM vm;

void initVM();
void freeVM();
InterpretResult interpret(const char* source);
void push(Value value);
Value pop();

#endif
