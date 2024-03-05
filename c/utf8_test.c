#include <stdio.h>
#include <string.h>
#include "utf8.h"

void *allocate_from_buffer(void *user_data, size_t n) { return user_data; }

#ifdef clox_utf8_test
int main(void) {
  char *s = "yes,我是中国人"; 
  char *p = s;
  printf("%d %s\n", (int) utf8size(s), s);
  utf8_int32_t ch; 
  p = utf8codepoint(p, &ch);
  printf("%s\n", (char*)(&ch));
  p = utf8codepoint(p, &ch);
  printf("%s\n", (char*)(&ch));
  p = utf8codepoint(p, &ch);
  printf("%s\n", (char*)(&ch));
  p = utf8codepoint(p, &ch);
  printf("%s\n", (char*)(&ch));
  p = utf8codepoint(p, &ch);
  printf("%s\n", (char*)(&ch));
  p = utf8codepoint(p, &ch);
  printf("%s\n", (char*)(&ch));
  p = utf8codepoint(p, &ch);
  printf("%s\n", (char*)(&ch));
  
  char user_data[1024]={'0'};
  char *q = p;
  p = utf8codepoint(p, &ch);
  utf8ndup_ex(q, p-q, allocate_from_buffer, user_data);
  printf("%s\n", user_data);
  printf("%s\n", (char*)&ch);

  printf("%s\n", "国");
  return 0;
}
#endif