package smart; const configurationInitFile = `project ~ (-nodock -final)
OUTDIR := &(CTD)/.configure

files (
    (*.c.include *.c++.include *.symbol *.variable *.function \
     *.structmember *.sizeof *.type *.c *.c++ *.log) => $(OUTDIR)
)

SHELL := shell -s
CC := gcc
CFLAGS :=
LDFLAGS :=
LOADLIBES :=
LIBS :=
LANG := c++
_INCLUDES_ :=
_FLAGS_ :=
_VALUE_ :=
_LIBS_ :=
_LOADLIBES_ :=

# -l=$(or &(outobj),&(CTD))/.configure/$(TARGET).$(LANG).log
# -o $(or &(outobj),&(CTD))/.configure/$(TARGET).$(LANG).out
-include:[((TARGET)) (unclose) (cd -s &/)] $(TARGET).$(LANG).include [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).$(LANG).out
-symbol:[((TARGET SYMBOL)) (unclose) (cd -s &/)] $(TARGET).symbol($(SYMBOL)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-function:[((TARGET FUNCTION)) (unclose) (cd -s &/)] $(TARGET).function($(FUNCTION)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-type:[((TARGET TYPE)) (unclose) (cd -s &/)] $(TARGET).type($(TYPE)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-library:[((TARGET LIBRARY FUNCTION)) (unclose) (cd -s &/)] $(TARGET).function($(FUNCTION)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -l$(LIBRARY) -o $(OUTDIR)/$(TARGET).out
-struct-member:[((TARGET STRUCT MEMBER)) (unclose) (cd -s &/)] $(TARGET).structmember($(STRUCT),$(MEMBER)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-sizeof:[((TARGET TYPE)) (unclose) (cd -s &/)] $(TARGET).sizeof($(TYPE)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-compiles:[((TARGET)) (unclose) (cd -s &/)] $(TARGET).$(LANG) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out

%.c.include:[(unclose) (cd -s &/) (plain c) (update-file -sp)]
	$(_INCLUDES_)
	#ifdef __CLASSIC_C__
	int main() { return 0; }
	#else
	int main(void) { return 0; }
	#endif
	
%.c++.include:[(unclose) (cd -s &/) (plain c++) (update-file -sp)]
	$(_INCLUDES_)
	int main() { return 0; }
	
%.symbol:[((SYMBOL)) (unclose) (cd -s &/) (plain text) (update-file -sp)]
	$(_INCLUDES_)
	int main(int argc, char** argv)
	{
	  (void)argv;
	#ifndef $(SYMBOL)
	  return ((int*)(\&$(SYMBOL)))[argc];
	#else
	  (void)argc;
	  return 0;
	#endif
	}
	
%.type:[((TYPE)) (unclose) (cd -s &/) (plain text) (update-file -sp)]
	$(_INCLUDES_)
	int main(int argc, char** argv)
	{
	  (void)argv;
	  (void)argc;
	  $(TYPE) var;
	  return 0;
	}
	
%.variable:[((VARIABLE)) (unclose) (cd -s &/) (plain text) (update-file -sp)]
	$(_INCLUDES_)
	extern int $(VARIABLE)
	#ifdef __CLASSIC_C__
	int main()
	#else
	int main(int argc, char** argv)
	#endif
	{ (void)argv; return $(VARIABLE); }
	
%.function:[((FUNCTION)) (unclose) (cd -s &/) (plain text) (update-file -sp)]
	$(_INCLUDES_)
	#ifdef __cplusplus
	extern "C"
	#endif
	char $(FUNCTION)(void);
	#ifdef __CLASSIC_C__
	int main()
	#else
	int main(int ac, char* av[])
	#endif
	{ $(FUNCTION)(); return 0; }
	
%.structmember:[((STRUCT MEMBER)) (unclose) (cd -s &/) (plain text) (update-file -sp)]
	$(_INCLUDES_)
	int main() { (void)sizeof((($(STRUCT) *)0)->$(MEMBER)); return 0; }
	
%.sizeof:[((TYPE)) (unclose) (cd -s &/) (plain text) (update-file -sp)]
	#undef ARCH
	#if defined(__i386)
	#   define ARCH "__i386"
	#elif defined(__x86_64)
	#   define ARCH "__x86_64"
	#elif defined(__ppc__)
	#   define ARCH "__ppc__"
	#elif defined(__ppc64__)
	#   define ARCH "__ppc64__"
	#elif defined(__aarch64__)
	#   define ARCH "__aarch64__"
	#elif defined(__ARM_ARCH_7A__)
	#   define ARCH "__ARM_ARCH_7A__"
	#elif defined(__ARM_ARCH_7S__)
	#   define ARCH "__ARM_ARCH_7S__"
	#endif
	#define SIZE (sizeof($(TYPE)))
	#ifdef __CLASSIC_C__
	int main(argc, argv) int argc; char *argv[];
	#else
	int main(int argc, char *argv[])
	#endif
	{ (void)argv; return SIZE; }
	
#$(OUTDIR)/pthreads.c
pthreads.c:[(unclose) (cd -s &/) (plain c) (update-file -sp)]
	#include <pthread.h>
	void* routine(void* args) { return args; }
	int main(void) {
	  pthread_t t;
	  pthread_create(\&t, routine, 0);
	  pthread_join(t, 0);
	  return 0;
	}
	
%.c:[(unclose) (cd -s &/) (plain c) (update-file -sp)]
	$(_VALUE_)
	
%.c++:[(unclose) (cd -s &/) (plain c++) (update-file -sp)]
	$(_VALUE_)
	
`
