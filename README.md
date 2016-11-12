# Smart (Simpler Make ART) (Drafting)

**Smart** is a [Semi-Functional Scripting Language]() designed to perform
recursive tasks easily. It's written in [Go](http://golang.org) programming
language.

[![GoDoc](https://godoc.org/github.com/duzy/smart/build?status.svg)](http://godoc.org/github.com/duzy/smart/build)

Support: [https://www.bountysource.com/teams/smart] -- ([Salt It Now!](https://salt.bountysource.com/checkout/amount?team=smart))

## Overview

The language is inspired by [GNU make](https://www.gnu.org/software/make/).
It's having almost the same syntax as makefile, similar but different features.
It could do the same job as [GNU make](https://www.gnu.org/software/make/) but not
limited to. **Smart** is supposed to be used more likely in a programming manner
of a [semi-functional]() paradigm by comparing to 
[functional progarmming](https://en.wikipedia.org/wiki/Functional_programming).

## Quick Example

```makefile
# The starting rule, using `:!:` to mark it as phony.
start:!: foo

# Declare a module `foo`.
module foo, gcc

me.sources := foo.c
me.export.includes := -I$(me.dir)
me.export.libdirs := -L$(me.dir)
me.export.libs := -lfoo

$(me.dir)/libfoo.a: $(me.dir)/foo.o
	@ar crs $@ $^

$(me.dir)/foo.o: $(me.dir)/foo.c
	@gcc -c -o $@ $<

commit
```

Why
===

Get tasks of complex dependency done the easy way!
