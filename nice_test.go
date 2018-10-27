// Copyright (C) 2018 Ramesh Vyaghrapuri. All rights reserved.
// Use of this source code is governed by a MIT-style license
// that can be found in the LICENSE file.

package nice_test

import (
	"errors"
	"github.com/tvastar/nice"
	"github.com/tvastar/nice/json"
	"reflect"
	"testing"
)

func TestNiceJSON(t *testing.T) {
	encoded, err := json.Encode([]interface{}{
		map[string]interface{}{"hello": []interface{}{1, 2.5, "boo"}},
		"hello| world()",
		[]interface{}(nil),
	})
	if err != nil {
		t.Fatal("Unexpected encode error", err)
	}
	decoded, err := json.Decode(encoded)
	if err != nil {
		t.Fatal("Decode failed", err, string(encoded))
	}
	expected := []interface{}{
		map[string]interface{}{"hello": []interface{}{1.0, 2.5, "boo"}},
		"hello| world()",
		[]interface{}(nil),
	}
	if !reflect.DeepEqual(decoded, expected) {
		t.Error("Unexpected decoded value", decoded, string(encoded))
	}
}

func TestRecursion(t *testing.T) {
	r := func(name []byte) nice.Handler {
		if string(name) == "x" {
			return func(_ nice.Resolver, args []byte) (interface{}, error) {
				return json.Resolve([]byte("json:string")), nil
			}
		}
		return json.Resolve(name)
	}

	encoded := "((x)|hello)"
	expected := "hello"
	v, err := nice.Eval(nice.Resolver(r).Recurse, []byte(encoded))
	if v != expected || err != nil {
		t.Error("Recursion error", v, err)
	}
}

func TestMissingFunction(t *testing.T) {
	err := errors.New("some error")
	r := func(_ []byte) nice.Handler {
		return nice.ErrorHandler(err)
	}

	encoded := "((\\x))"
	v, err2 := nice.Eval(nice.Resolver(r).Recurse, []byte(encoded))
	if v != nil || err2 != err {
		t.Error("Missing function error", v, err2)
	}
}

func TestNotAFunction(t *testing.T) {
	encoded := "((json:string|))"
	v, err := json.Decode([]byte(encoded))
	if v != nil || err != nice.Error("nice: not a function") {
		t.Error("Not a function error", v, err)
	}
}

func TestMissingClose(t *testing.T) {
	encoded := "(x"
	v, err := json.Decode([]byte(encoded))
	if v != nil || err != nice.Error("nice: missing )") {
		t.Error("Not a function error", v, err)
	}
}

func TestMismatched(t *testing.T) {
	encoded := "(x))"
	v, err := json.Decode([]byte(encoded))
	if v != nil || err != nice.Error("nice: mismatched )") {
		t.Error("Unexpected", v, err)
	}

	encoded = "((x)"
	v, err = json.Decode([]byte(encoded))
	if v != nil || err != nice.Error("nice: mismatched (") {
		t.Error("Unexpected", v, err)
	}

	encoded = "(json:string|(x)))"
	v, err = json.Decode([]byte(encoded))
	if v != nil || err != nice.Error("nice: mismatched )") {
		t.Error("Unexpected", v, err)
	}

	encoded = "(json:string|()"
	v, err = json.Decode([]byte(encoded))
	if v != nil || err.Error() != "nice: mismatched (" {
		t.Error("Unexpected", v, err)
	}
}

func TestArgsError(t *testing.T) {
	encoded := "(json:array|(x))"
	v, err := json.Decode([]byte(encoded))
	if err.Error() != "json: unknown type: x" {
		t.Error("Args error", v, err)
	}

	encoded = "(json:array|(x)|)"
	v, err = json.Decode([]byte(encoded))
	if err.Error() != "json: unknown type: x" {
		t.Error("Args error", v, err)
	}
}
