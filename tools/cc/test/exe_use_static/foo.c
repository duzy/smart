#include <stdio.h>
int main(int argc, char**argv) {
  extern int foo();
  printf("hello: %s (static: %d)\n", argv[0], foo());
  return 0;
}
