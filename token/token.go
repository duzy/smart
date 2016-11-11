//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  The implementation of smart/token package is highly referencing to go/token.
//  
package token

import (
        "strconv"
)

type Token int

const (
        // Special tokens.
	ILLEGAL Token = iota
	EOF
	COMMENT
        
	literal_beg
	// Identifiers and basic type literals
	// (these tokens stand for classes of literals)
	STRING // "abc"
	literal_end
        
	operator_beg
        ASSIGN // =
        QUE_ASSIGN // ?=
        SCO_ASSIGN // :=
        DCO_ASSIGN // ::=
        EXC_ASSIGN // !=     exclamation
        ADD_ASSIGN // +=
	operator_end

	keyword_beg
        MODULE
        COMMIT
        USE
        EXPORT
	keyword_end
)

