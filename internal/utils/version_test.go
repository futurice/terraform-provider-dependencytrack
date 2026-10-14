// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package utils

import "testing"

func TestIsV5Version(t *testing.T) {
	tests := []struct {
		version string
		want    bool
		wantErr bool
	}{
		{"4.14.5", false, false},
		{"5.2.0", true, false},
		{"6.0.0-SNAPSHOT", true, false},
		{"latest", false, true},
		{"", false, true},
	}

	for _, tt := range tests {
		got, err := IsV5Version(tt.version)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("IsV5Version(%q) = %t, %v, want %t, error %t", tt.version, got, err, tt.want, tt.wantErr)
		}
	}
}
