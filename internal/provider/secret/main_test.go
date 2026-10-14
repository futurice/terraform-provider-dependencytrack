// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package secret_test

import (
	"os"
	"testing"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testDependencyTrack *testutils.TestDependencyTrack

func TestMain(m *testing.M) {
	if os.Getenv(resource.EnvTfAcc) != "" {
		var cleanup func()
		testDependencyTrack, cleanup = testutils.InitTestDependencyTrack()
		defer cleanup()
	}

	m.Run()
}
