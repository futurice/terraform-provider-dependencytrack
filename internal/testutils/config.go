// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package testutils

import "strings"

func ComposeConfigs(configs ...string) string {
	return strings.Join(configs, "\n")
}
