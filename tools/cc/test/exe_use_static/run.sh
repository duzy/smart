if ! smart -V > temp.txt ; then
    echo "$BASH_SOURCE:$LINENO: failed building 'gcc/exe'"
fi
