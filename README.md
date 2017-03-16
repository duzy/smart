# Smart (Simpler Make ART) (Drafting)

**Smart** is a [Semi-Functional Scripting Language]() designed to perform
recursive tasks easily. It's written in [Go](http://golang.org) programming
language.

[![GoDoc](https://godoc.org/github.com/duzy/smart/build?status.svg)](http://godoc.org/github.com/duzy/smart/build)

## Overview

The language is inspired by [GNU make](https://www.gnu.org/software/make/).
It's having same similar syntax as `makefile`, but a `smart` program is highly
modularized and multi-dialect (extensiable). In a `makefile`, there's only a global
namespace, macros defined can later be referenced by any other macros or rules. 
In `smart`, symbols are contained in a module, and the major modules are projects. 
A project is designed to be executed in order to update outdated targets, a module
is to do more specific tasks and supposed to be **used** by a project.

A smart module is declared with the keyword `module` or `project`. A module can be
imported or used by any other module using keywords `import` or `use`. Symbols and
rules defined in a module can only be accessed within the module scope.

## History

The ideas of the `smart` language is originated from the old [smart-make](https://github.com/duzy/smart-make)
project, which is written in `makefile` to ease building projects of complex hierachy.
The rational of `smart-make` is very similar to [the build system of Android OS](https://android.googlesource.com/platform/build/+/master).

The goal of `smart` is to be great successor of makefile doing jobs like `smart-make`
and [the Android OS build system](https://android.googlesource.com/platform/build/+/master).

## Quick Example

```makefile
project example

LINK = g++
COMPILE = g++ -c
LIBS =

## "pthread" is a predefiend module, using it will append values
## of symbols like CFLAGS, LDFLAGS, LIBS, etc.
use "pthread"

# The default rule, using `shell` dialect to interpret the recipes.
foo:[shell]: foo.o
	$(LINK) -o $@ $^ $(LIBS)

# The second `shell` rule to compile the source.
foo.o:[shell]: foo.cpp
	$(COMPILE) -o $@ $<
```

Why
===

Build projects complex hierachy the easy way!
