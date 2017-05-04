#!/bin/bash
set -e

out="./out"
exe=""
smart="$(dir $PWD)/smart"

[[ -f ../smart.go && -f ../main/main.go && -f ../build.sh ]] || {
    echo $BASH_SOURCE:$LINENO "not in test subdir"
    exit -1
}

needs_build() {
    [[ -f $smart ]] || return 1
    T=`stat -c %Y $smart`
    for g in ../*.go; do
        [[ "$T" -lt "`stat -c %Y $g`" ]] && return 1
    done
    return 0
}

enter() {
    cd $1 && echo "smart: Entering directory \`$1'"
}

leave() {
    cd - > /dev/null && echo "smart: Leaving directory \`$1'"
}

needs_build || {
    enter ..
    ./build.sh
    leave ..
}

run() {
    local D=$1
    local f
    for f in $D/* $D/.* ; do
        local e=""
        case $f in
            */.|*/..|*/out|*/src|*/res|./run.sh)
                #echo "$BASH_SOURCE:$LINENO:info: ignore $f"
                ;;
            */.smart)
                #echo "$BASH_SOURCE:$LINENO:info: $*/.smart"
                ;;
            */run.sh)
                #echo "$BASH_SOURCE:$LINENO:info: $*/run.sh"
                enter $*
                rm -rf out
                if !(. run.sh); then
                    e="$BASH_SOURCE:$LINENO: failed '$f'"
                fi
                if [[ "x${e}x" == "xx" ]]; then
                    if [[ -f temp.txt && -f expect.txt ]]; then
                        if ! diff=$(diff temp.txt expect.txt); then
                            echo ".smart:1: unexpected 'smart' output"
                            echo "========== DIFF begin ========== ($*)"
                            echo "$diff"
                            echo "========== DIFF end ============ ($*)"
                        fi
                    fi
                    if [[ -f temp.txt && ! -f check.sh ]]; then
                        rm -f temp.txt
                    fi
                fi
                leave $*
                if [[ "x${e}x" != "xx" ]]; then
                    #echo "----------"
                    echo $e
                fi
                ;;
            *)
                if [[ -d $f ]]; then
                    #echo "$BASH_SOURCE:$LINENO:info: $f"
                    run $f
                fi
                ;;
        esac
    done
}

check() {
    local D=$1
    local f
    for f in $D/* ; do
        local e=""
        case $f in
            */check.sh)
                enter $*
                (. check.sh) || e="$BASH_SOURCE:$LINENO: failed '$f'"
                if [[ "x${e}x" != "xx" ]]; then
                    if [[ -f temp.txt ]]; then
                        echo "========== smart output begins ========== ($*)"
                        cat temp.txt
                        echo "========== smart output ends ============ ($*)"
                    fi
                fi
                rm -f temp.txt
                leave $*
                if [[ "x${e}x" != "xx" ]]; then
                    #echo "----------"
                    echo $e
                fi
                ;;
            *)
                [[ -d $f ]] && {
                    check $f || continue
                }
                ;;
        esac
    done
}

checkdir() {
    local L=$1
    local D=$2
    if ! [[ -d $D ]]; then
        echo "$L: $D not found"
        return 1
    fi
}

checkfile() {
    local L=$1
    local F=$2
    if ! [[ -f $F ]]; then
        echo "$L: $F not found"
        return 1
    fi
}

start() {
    local D=$1
    PATH="$(dirname $PWD):/store/open/android-ndk-r7:${PATH##*/smart-build/bin}"
    echo "$BASH_SOURCE:$LINENO:info: =================================================="
    echo "$BASH_SOURCE:$LINENO:info: RUN test cases..."
    run $D
    echo "$BASH_SOURCE:$LINENO:info: =================================================="
    echo "$BASH_SOURCE:$LINENO:info: CHECK test cases..."
    check $D
}

#start .
#start ./gcc/exe
#start ./android-ndk/shared
start ./grammar/misc
