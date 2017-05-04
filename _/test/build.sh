#!/bin/bash
#set -e -b -x
set -e -b

OFILES=
GOFILES="smart.go parse.go builtins.go gcc.go clang.go android-ndk.go android-sdk.go"
for i in $GOFILES; do
    O="${i/%.go/.6}"
    OFILES="$OFILES $O"
#    go tool 6g -o $O $i
done

go tool 6g -o _go_.6 $GOFILES

[[ -f smart.a ]] && rm -f smart.a
go tool pack gr smart.a _go_.6

[[ -f smart ]] && rm -f smart
go tool 6g -I . -o main.6 main/main.go
go tool 6l -L . -o smart main.6

test -f smart && rm -f $OFILES main.6

#go test -file smart_test.go .
