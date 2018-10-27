// Copyright (C) 2018 Ramesh Vyaghrapuri. All rights reserved.
// Use of this source code is governed by a MIT-style license
// that can be found in the LICENSE file.

// Package json implements a json-like codec for nice.
//
// The Encode function encodes generic JSON types into the nice
// format. This includes map[string]interface{}, []interface{},
// string, int, float and nil.  It errors out for all other types.
//
// The Resolver resolves all the types produced by the Encode function
// (though it only decodes all numbers to float64). The Decode
// function provides a simple wrapper using the provided Resolver.
package json

import (
	"bytes"
	"errors"
	"github.com/tvastar/nice"
	"io"
	"strconv"
)

// Encode encodes the provided JSON-like value into a byte sequence
func Encode(v interface{}) ([]byte, error) {
	w := &bytes.Buffer{}
	if err := EncodeTo(w, v); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// Decode converts the byte sequence into a  JSON-like value
func Decode(b []byte) (interface{}, error) {
	return nice.Eval(nice.Resolver(Resolve).Recurse, b)
}

// Resolve resolves all the type names implemented by this package.
func Resolve(name []byte) nice.Handler {
	switch string(name) {
	case "json:null":
		return evalNull
	case "json:string":
		return evalString
	case "json:number":
		return evalNumber
	case "json:array":
		return evalArray
	case "json:map":
		return evalMap
	}
	return nice.Handler(func(_ nice.Resolver, _ []byte) (interface{}, error) {
		return nil, errors.New("json: unknown type: " + string(name))
	})
}

func evalNull(r nice.Resolver, args []byte) (interface{}, error) {
	return nil, nil
}

func evalNumber(r nice.Resolver, args []byte) (interface{}, error) {
	v, err := evalString(r, args)
	if err == nil {
		return strconv.ParseFloat(v.(string), 64)
	}
	return nil, err
}

func evalString(r nice.Resolver, args []byte) (interface{}, error) {
	values, err := nice.EvalArgs(r, args)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, errors.New("json: incorrect number of args")
	}
	arg, ok := values[0].(nice.Raw)
	if !ok {
		return nil, errors.New("json: incorrect type")
	}

	return string(nice.Unescape([]byte(arg))), nil
}

func evalArray(r nice.Resolver, args []byte) (interface{}, error) {
	return nice.EvalArgs(r, args)
}

func evalMap(r nice.Resolver, args []byte) (interface{}, error) {
	v, err := nice.EvalArgs(r, args)
	if err != nil {
		return nil, err
	}

	if len(v)%2 != 0 {
		return nil, errors.New("json:map expects even number of args")
	}
	result := map[string]interface{}{}
	for kk := 0; kk < len(v); kk += 2 {
		key, ok := v[0].(nice.Raw)
		if !ok {
			return nil, errors.New("json:map allows string keys only")
		}
		result[string(nice.Unescape([]byte(key)))] = v[1]
	}
	return result, nil
}

func call(w io.Writer, name string, args ...[]byte) {
	must(w.Write([]byte{'('}))
	must(w.Write([]byte(name)))
	if len(args) > 0 {
		for _, arg := range args {
			must(w.Write([]byte{'|'}))
			must(w.Write(arg))
		}
	}
	must(w.Write([]byte{')'}))
}

// EncodeTo encodes the provided JSON-like value writing the output
// into the writer
func EncodeTo(w io.Writer, v interface{}) error {
	switch v := v.(type) {
	case nil:
		call(w, "json:null")
	case int:
		arg := strconv.FormatInt(int64(v), 10)
		call(w, "json:number", []byte(arg))
	case int32:
		arg := strconv.FormatInt(int64(v), 10)
		call(w, "json:number", []byte(arg))
	case int64:
		arg := strconv.FormatInt(v, 10)
		call(w, "json:number", []byte(arg))
	case float32:
		arg := strconv.FormatFloat(float64(v), 'E', -1, 64)
		call(w, "json:number", []byte(arg))
	case float64:
		arg := strconv.FormatFloat(v, 'E', -1, 64)
		call(w, "json:number", []byte(arg))
	case string:
		arg := nice.Escape([]byte(v))
		call(w, "json:string", arg)
	case []interface{}:
		must(w.Write([]byte("(json:array")))
		for _, elt := range v {
			must(w.Write([]byte{'|'}))
			if err := EncodeTo(w, elt); err != nil {
				return err
			}
		}
		must(w.Write([]byte{')'}))
	case map[string]interface{}:
		must(w.Write([]byte("(json:map")))
		for k, elt := range v {
			must(w.Write([]byte{'|'}))
			must(w.Write(nice.Escape([]byte(k))))
			must(w.Write([]byte{'|'}))
			if err := EncodeTo(w, elt); err != nil {
				return err
			}
		}
		must(w.Write([]byte{')'}))
	default:
		return errors.New("json: unknown type")
	}
	return nil
}

func must(_ int, err error) {}
