// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package utils

import "testing"

func TestJSONEqual(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{`{"a":1,"b":"x"}`, `{"b": "x", "a": 1}`, true},
		{`{"a":1}`, `{"a":2}`, false},
		{`{"a":1}`, `{"a":1,"b":2}`, false},
		{`{}`, ``, false},
		{`not json`, `not json`, false},
	}

	for _, tt := range tests {
		if got := JSONEqual(tt.a, tt.b); got != tt.want {
			t.Errorf("JSONEqual(%q, %q) = %t, want %t", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCompactJSON(t *testing.T) {
	if got := CompactJSON(`{"b": 1, "a": [1, 2]}`); got != `{"b":1,"a":[1,2]}` {
		t.Errorf("CompactJSON kept whitespace: %s", got)
	}
	if got := CompactJSON(`not json`); got != `not json` {
		t.Errorf("CompactJSON changed invalid JSON: %s", got)
	}
}
