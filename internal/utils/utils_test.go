package utils_test

import (
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/google/uuid"
	"testing"
)

func TestParseUUID_basic(t *testing.T) {
	testUUID := uuid.MustParse("8ffb30fb-77e6-4886-9f32-ff142f9bf90b")
	result, diags := utils.ParseUUID(testUUID.String())

	if result != testUUID {
		t.Errorf("Parsed UUID [%s] is different than expected [%s]", result, testUUID)
	}

	if diags.HasError() {
		t.Errorf("Unexpected error: %v", diags)
	}
}

func TestParseUUID_invalid(t *testing.T) {
	_, diags := utils.ParseUUID("not-an-UUID")

	if !diags.HasError() {
		t.Errorf("Error expected, but received none")
	}
}
