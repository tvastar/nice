// Copyright (C) 2018 Ramesh Vyaghrapuri. All rights reserved.
// Use of this source code is governed by a MIT-style license
// that can be found in the LICENSE file.

package json_test

import (
	"github.com/tvastar/nice/json"
	"reflect"
	"testing"
)

func TestJSON(t *testing.T) {
	encoded, err := json.Encode([]interface{}{
		map[string]interface{}{"hello": []interface{}{1, 2.5, "boo"}},
		"hello| world()",
		[]interface{}(nil),
		int(1),
		int32(2),
		int64(3),
		float32(1.5),
		float64(1.5),
		nil,
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
		float64(1),
		float64(2),
		float64(3),
		float64(1.5),
		float64(1.5),
		nil,
	}
	if !reflect.DeepEqual(decoded, expected) {
		t.Error("Unexpected decoded value", decoded, string(encoded))
	}
}

func TestErrors(t *testing.T) {
	if _, err := json.Encode([]int{2}); err.Error() != "json: unknown type" {
		t.Error("Unexpected", err)
	}

	encoded := "(json:string|(x))"
	decoded, err := json.Decode([]byte(encoded))
	if err.Error() != "json: unknown type: x" {
		t.Error("Unexpected", decoded, err)
	}

	encoded = "(json:number|23a)"
	decoded, err = json.Decode([]byte(encoded))
	if err.Error() != "strconv.ParseFloat: parsing \"23a\": invalid syntax" {
		t.Error("Unexpected", decoded, err)
	}

	encoded = "(json:number|2|1)"
	decoded, err = json.Decode([]byte(encoded))
	if err.Error() != "json: incorrect number of args" {
		t.Error("Unexpected", decoded, err)
	}

	encoded = "(json:number|(json:number|2))"
	decoded, err = json.Decode([]byte(encoded))
	if err.Error() != "json: incorrect type" {
		t.Error("Unexpected", decoded, err)
	}

	encoded = "(json:map|(x))"
	decoded, err = json.Decode([]byte(encoded))
	if err.Error() != "json: unknown type: x" {
		t.Error("Unexpected", decoded, err)
	}

	encoded = "(json:map|2)"
	decoded, err = json.Decode([]byte(encoded))
	if err.Error() != "json:map expects even number of args" {
		t.Error("Unexpected", decoded, err)
	}

	encoded = "(json:map|(json:number|2)|5)"
	decoded, err = json.Decode([]byte(encoded))
	if err.Error() != "json:map allows string keys only" {
		t.Error("Unexpected", decoded, err)
	}

	_, err = json.Encode([]interface{}{[]int{2}})
	if err.Error() != "json: unknown type" {
		t.Error("Unexpected", err)
	}

	_, err = json.Encode(map[string]interface{}{"x": []int{2}})
	if err.Error() != "json: unknown type" {
		t.Error("Unexpected", err)
	}
}
