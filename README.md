# Smart (Simpler Making ART) (BETA)

NOTE: The `smart` project has temporarily gone private and should reopen again in the future.

## Overview

The language is inspired by [GNU make](https://www.gnu.org/software/make/).
It has a similar syntax to `Makefile`, but a `smart` program is highly
modularized, multi-dialect (extensible) and data-typed. In a `Makefile`, there's 
only a global namespace, macros defined can later be referenced by any other macros
or rules. In `smart`, symbols are contained in a module, and the major modules are
projects. A project is designed to be executed in order to update outdated targets,
a module is to do more specific tasks and supposed to be **used** by a project.

A `smart` module is declared with the keyword `module` or `project`. A module can be
imported or used by any other module using keywords `import` or `use`. Symbols and
rules defined in a module can only be accessed within the module scope.

The `smart` language has some basic data types, this is another important difference
comparing to macros in a makefile.

## Quick Start

### Install `smart` utility

We use `go` to install the `smart` command line utility directly from GitHub like this:

```shell
$ go get github.com/extbit/smart/cmd
$ $GOPATH/bin/smart -help
```

### Write `smart` scripts

The `smart` command will look for file `build.smart` in the working directory to
start building. For example of doing this:

```shell
$ cd $GOPATH/src/github.com/extbit/smart/examples/hello
$ $GOPATH/bin/smart run
Hello World!
```

It should build the `hello` example and run it, having a 'Hello World!' output.

## Quick Example

```makefile
project example

## "posix/thread" is a predefiend module, allowing users to use pthread
## in the project, it's supposed to append values of symbols like CFLAGS, LDFLAGS,
## LIBS, etc. But at the current version, it affects only the `libs` symbol.
use "posix/thread"

LINK = g++
COMPILE = g++ -c
LIBS =

GREETING = "hello, there"

# The default rule, using `shell` dialect to interpret the recipes.
# Note that the `libs` was introduced by the "posix/thread".
foo:[(shell)]: foo.o
	$(LINK) -o $@ $^ $(libs)

# The second `shell` rule to compile the source.
foo.o:[(shell)]: foo.cpp
	$(COMPILE) -o $@ $<

# The `plain` dialect simply expands the recipes into plain text,
# and the `(as text)` tells that the symbol `text` is being used to
# store the plain text. The `,` starts post-execution of the recipes.
foo.cpp:[(plain) (update-file)]:
	#include <iostream>
	int main(int argc, char** argv) {
	    std::cout <<"$(GREETING)" << std::endl;
	    return 0;
	}

check:[(python) (stdout-equals "okay")]:
	print "not okay"
```

History
=======

The idea of the `smart` language is originated from the old [smart-make](https://github.com/duzy/smart-make)
project, which is written in pure `Makefile` to ease building projects having a complex hierarchy.
The rationale of `smart-make` is very similar to the [Android build system](https://android.googlesource.com/platform/build/+/master).

The goal of `smart` is to supersede `make` utility (especially in the scenario of modularization and hierarchical building), following the rationale of `smart-make`
and the [Android build system](https://android.googlesource.com/platform/build/+/master).

Why
===

Build projects with complex hierarchies the easy way!
