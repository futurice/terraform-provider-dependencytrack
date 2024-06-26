package testutils

import "strings"

func ComposeConfigs(configs ...string) string {
	return strings.Join(configs, "\n")
}
