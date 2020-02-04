package smart; import "runtime/debug"; import "strings"; func configurationInitFile() (string, string) { source := `# confinit
project ~ (-nodock -final)
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
-include:[((TARGET)) (closure)] $(TARGET).$(LANG).include [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).$(LANG).out
-symbol:[((TARGET SYMBOL)) (closure)] $(TARGET).symbol($(SYMBOL)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-function:[((TARGET FUNCTION)) (closure)] $(TARGET).function($(FUNCTION)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-type:[((TARGET TYPE)) (closure)] $(TARGET).type($(TYPE)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-library:[((TARGET LIBRARY FUNCTION)) (closure)] $(TARGET).function($(FUNCTION)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -l$(LIBRARY) -o $(OUTDIR)/$(TARGET).out
-struct-member:[((TARGET STRUCT MEMBER)) (closure)] $(TARGET).structmember($(STRUCT),$(MEMBER)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-sizeof:[((TARGET TYPE)) (closure)] $(TARGET).sizeof($(TYPE)) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out
-compiles:[((TARGET)) (closure)] $(TARGET).$(LANG) [($(SHELL) -l=$(OUTDIR)/$(TARGET).$(LANG).log) (check -a status=0)]
	@$(CC) -v -Wl,-v -x$(LANG) $(CFLAGS) $(LDFLAGS) $(_FLAGS_) $< $(LOADLIBES) $(_LOADLIBES_) $(LIBS) $(_LIBS_) -o $(OUTDIR)/$(TARGET).out

%.c.include:[(closure) (plain c) (update-file -sp)]
	$(_INCLUDES_)
	#ifdef __CLASSIC_C__
	int main() { return 0; }
	#else
	int main(void) { return 0; }
	#endif
	
%.c++.include:[(closure) (plain c++) (update-file -sp)]
	$(_INCLUDES_)
	int main() { return 0; }
	
%.symbol:[((SYMBOL)) (closure) (plain text) (update-file -sp)]
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
	
%.type:[((TYPE)) (closure) (plain text) (update-file -sp)]
	$(_INCLUDES_)
	int main(int argc, char** argv)
	{
	  (void)argv;
	  (void)argc;
	  $(TYPE) var;
	  return 0;
	}
	
%.variable:[((VARIABLE)) (closure) (plain text) (update-file -sp)]
	$(_INCLUDES_)
	extern int $(VARIABLE)
	#ifdef __CLASSIC_C__
	int main()
	#else
	int main(int argc, char** argv)
	#endif
	{ (void)argv; return $(VARIABLE); }
	
%.function:[((FUNCTION)) (closure) (plain text) (update-file -sp)]
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
	
%.structmember:[((STRUCT MEMBER)) (closure) (plain text) (update-file -sp)]
	$(_INCLUDES_)
	int main() { (void)sizeof((($(STRUCT) *)0)->$(MEMBER)); return 0; }
	
%.sizeof:[((TYPE)) (closure) (plain text) (update-file -sp)]
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
pthreads.c:[(closure) (plain c) (update-file -sp)]
	#include <pthread.h>
	void* routine(void* args) { return args; }
	int main(void) {
	  pthread_t t;
	  pthread_create(\&t, routine, 0);
	  pthread_join(t, 0);
	  return 0;
	}
	
%.c:[(closure) (plain c) (update-file -sp)]
	$(_VALUE_)
	
%.c++:[(closure) (plain c++) (update-file -sp)]
	$(_VALUE_)
	
`
        var num int
        var filename string
        var lines = strings.Split(string(debug.Stack()), "\n")
        for _, line := range lines {
                if !strings.HasPrefix(line, "\t") { continue }
                if i := strings.Index(line, ":"); num == 1 && i > 0 {
                        filename = line[1:i]
                        break
                }
                num += 1
        }
        return source, filename
}
