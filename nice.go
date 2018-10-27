// Copyright (C) 2018 Ramesh Vyaghrapuri. All rights reserved.
// Use of this source code is governed by a MIT-style license
// that can be found in the LICENSE file.

// Package nice is a lisp-like call expression format.
//
// This package defines a very simple lisp-like call expression
// format:
//     expr := <atomic> OR "(" <expr> [ "|" <expr> ]* ")"
//
// The expressions is a list which uses the pipe symbol ("|") to
// separate out the elements of the list.
//
// This is meant to be both a data interchange format as well as a way
// to represent expressions in domain specific languages.
//
// Syntax details
//
// The encoded format of expressions is quite simple with four special
// characters: "(", "|", ")" and "\" with the backslash used only to
// escape the special characters.
//
// The brackets introduce expressions where the arguments are
// separated by the pipe "|" symbol.  When the expression does not
// start with brackets, it is considered as a raw sequence of bytes.
//
// As in Lisp, the first element of the list provides the function
// name to invoke with the rest of the list as arguments. If the first
// byte is not a bracket-open, then the whole sequence is treated as a
// raw sequence of bytes (i.e. a single atomic value).
//
// As with Lisp, if there is a list within the first element of the
// list, that list is expected to be evaluated to find the name of the
// function to run.
//
// Example Encoding:
//
// A call like add(x,y) would be represented by something like:
//        (add|x|y)
//
// where the x and y are recursively encoded the same way.
//
// See https://godoc.org/github.com/tvastar/nice/json for a codec that
// works with json-like types ([]interface{}, map[string]interface{},
// etc).
//
// See https://godoc.org/github.com/tvaster/nice/nicer for a codec
// that marhsals and unmarshals any particular native Golang type.
//
// Issues
//
// Due to the lazy evaluation nature of this package, some parse
// errors are deferred. An example is the following invalid string:
//      (add|()
//
package nice

// Resolver resolves a name to a handler.
//
// Note that the name can be an expression itself -- so
// resolvers would typically begin by calling Eval recursively on the
// name. For convenience,  the Resolve.Recurse function takes care of
// this and os can be used to wrap a non-recursive lookup function
type Resolver func(name []byte) Handler

// Recurse handles the case where the argument is an expression
// itself. In that case, it evaluates the expression and returns that
// if it is a handler. If the returned expression is not a handler, it
// returns an error handler.
func (r Resolver) Recurse(name []byte) Handler {
	if len(name) > 0 && name[0] != '(' {
		return r(name)
	}
	v, err := Eval(r, name)
	if err != nil {
		return ErrorHandler(err)
	}

	if h, ok := v.(Handler); ok {
		return h
	}

	return ErrorHandler(Error("nice: not a function"))
}

// ErrorHandler returns a Handler which always returns the passed in error.
func ErrorHandler(err error) Handler {
	return Handler(func(_ Resolver, _ []byte) (interface{}, error) {
		return nil, err
	})
}

// Handler evaluates a sequence of args.  Use EvalArgs to recursively
// evaluate the args. Note that there is a distinction between an
// absence of args "(x)" vs an empty "(x|)".  The former will have
// args set to nil while the latter will have args set to an empty
// byte slice.
type Handler func(r Resolver, args []byte) (interface{}, error)

// Eval decodes and evaluates the provided UTF8 byte sequence. If the
// provided sequence is not of the form "(..)", it is considered  an
// atomic sequence and a Raw result is returned.
func Eval(r Resolver, s []byte) (interface{}, error) {
	if len(s) == 0 || s[0] != '(' {
		return Raw(s), nil
	}
	if s[len(s)-1] != ')' {
		return nil, Error("nice: missing )")
	}

	nesting := 1
	for kk := 1; kk < len(s)-1; kk++ {
		switch {
		case s[kk] == '\\':
			kk++
		case s[kk] == '|' && nesting == 1:
			return r(s[1:kk])(r, s[kk+1:len(s)-1])
		case s[kk] == '(':
			nesting++
		case s[kk] == ')':
			nesting--
			if nesting == 0 {
				return nil, Error("nice: mismatched )")
			}
		}
	}

	if nesting != 1 {
		return nil, Error("nice: mismatched (")
	}

	return r(s[1:len(s)-1])(r, nil)
}

// EvalArgs takes a pipe-separated sequence orgs (as provied to a
// Handler) and returns a list of values for it.  Note that a nil args
// parameter results in a nil value while an empty args byte slice
// results in a single value (with the atomic empty byte sequence).
func EvalArgs(r Resolver, s []byte) ([]interface{}, error) {
	if s == nil {
		return nil, nil
	}

	nesting, offset := 0, 0
	result := []interface{}(nil)
	for kk := 0; kk < len(s); kk++ {
		switch {
		case s[kk] == '\\':
			kk++
		case s[kk] == '|' && nesting == 0:
			v, err := Eval(r, s[offset:kk])
			if err != nil {
				return nil, err
			}
			result = append(result, v)
			offset = kk + 1
		case s[kk] == '(':
			nesting++
		case s[kk] == ')':
			nesting--
			if nesting < 0 {
				return nil, Error("nice: mismatched )")
			}
		}
	}

	if nesting != 0 {
		return nil, Error("nice: mismatched (")
	}

	v, err := Eval(r, s[offset:])
	if err != nil {
		return nil, err
	}
	return append(result, v), nil
}

// Escape takes a byte sequence and escapes any of the special
// characters in the string
func Escape(s []byte) []byte {
	result := []byte(nil)
	for kk, c := range s {
		special := c == '\\' || c == '(' || c == ')' || c == '|'
		if special && result == nil {
			result = make([]byte, 0, len(s))
			result = append(result, s[:kk]...)
		}
		if special {
			result = append(result, '\\')
		}
		if result != nil {
			result = append(result, c)
		}
	}
	if result != nil {
		return result
	}
	return s
}

// Unescape converts an escaped byte sequence to its unescaped
// form. It is the inverse of Escape
func Unescape(s []byte) []byte {
	result := []byte(nil)
	offset, slash := 0, false
	for kk, c := range s {
		switch {
		case !slash && c == '\\':
		case slash && result == nil:
			result = make([]byte, len(s))
			copy(result[0:len(s)], s[:kk-1])
			offset = kk - 1
			fallthrough
		case result != nil:
			result[offset] = c
			offset++
		}
		slash = !slash && c == '\\'
	}
	if result == nil {
		return s
	}
	return result[:offset]
}

// Error is returned in case of any errors in this package
type Error string

// Error impolements the Error interface
func (p Error) Error() string {
	return string(p)
}

// Raw refers to an unprocessed byte sequence that is returned by Eval
// when the input to it is an "atomic expression" (i.e when the input
// isn't of the form "(...)")
type Raw []byte
