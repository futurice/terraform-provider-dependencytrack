package testutils

import (
	"context"
	"testing"
)

func CreateTestContext(t *testing.T) context.Context {
	t.Helper()

	// Nothing here for now, but better to have context creation abstracted.
	return context.Background()
}
