// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"bytes"
	"encoding/json"
	"reflect"
)

// JSONEqual reports whether two strings hold the same JSON value, ignoring key order and whitespace.
func JSONEqual(a, b string) bool {
	var av, bv interface{}
	if json.Unmarshal([]byte(a), &av) != nil || json.Unmarshal([]byte(b), &bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

// CompactJSON strips insignificant whitespace, or returns the input unchanged if it isn't valid JSON.
func CompactJSON(s string) string {
	var buf bytes.Buffer
	if json.Compact(&buf, []byte(s)) != nil {
		return s
	}
	return buf.String()
}
