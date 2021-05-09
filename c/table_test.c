#include <stdio.h>

#include "table.h"
#include "value.h"
#include "object.h"

#ifdef clox_table_test

int main() {
  Table tb;
  initTable(&tb);

  Value v = OBJ_VAL(copyString("test string", 11));
  bool ok;
  ObjString* a = copyString("a", 1);
  ok = tableSet(&tb, a, v);
  printf("%d %d\n", ok, a->hash);

  ObjString* b = copyString("b", 1);
  ok = tableSet(&tb, b, v);
  printf("%d %d\n", ok, b->hash);

  ObjString* c = copyString("c", 1);
  ok = tableSet(&tb, c, v);
  printf("%d %d\n", ok, c->hash);

  ObjString* d = copyString("d", 1);
  ok = tableSet(&tb, d, v);
  printf("%d %d\n", ok, d->hash);

  ObjString* e = copyString("e", 1);
  ok = tableSet(&tb, e, v);
  printf("%d %d\n", ok, e->hash);

  Value val;
  ok = tableGet(&tb, c, &val);
  printf("%d %p\n", ok, val.as.obj);
  printObject(val);
  printf("\n");

  ok = tableDelete(&tb, d);
  printf("delete, resp=%d\n", ok);

  Table tb1;
  initTable(&tb1);

  tableAddAll(&tb, &tb1);
  ok = tableGet(&tb1, e, &val);
  printObject(val);
  printf("\n");

  freeTable(&tb);
  freeTable(&tb1);
  return 0;
}

#endif
