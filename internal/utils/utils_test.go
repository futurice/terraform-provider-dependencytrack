package utils_test

import (
	"context"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"reflect"
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

func TestTFStringSetToStringSlice_basic(t *testing.T) {
	ctx := context.Background()

	testStrings := []string{"a", "b"}
	testSet, _ := types.SetValueFrom(ctx, types.StringType, testStrings)

	result, diags := utils.TFStringSetToStringSlice(ctx, testSet)

	if !reflect.DeepEqual(result, testStrings) {
		t.Errorf("Parsed strings [%v] are different than expected [%v]", result, testStrings)
	}

	if diags.HasError() {
		t.Errorf("Unexpected error: %v", diags)
	}
}

func TestTFStringSetToStringSlice_empty(t *testing.T) {
	ctx := context.Background()

	var testStrings []string
	testSet, _ := types.SetValueFrom(ctx, types.StringType, testStrings)

	result, diags := utils.TFStringSetToStringSlice(ctx, testSet)

	if !reflect.DeepEqual(result, testStrings) {
		t.Errorf("Parsed strings [%v] are different than expected [%v]", result, testStrings)
	}

	if diags.HasError() {
		t.Errorf("Unexpected error: %v", diags)
	}
}

func TestTFStringSetToStringSlice_unknown(t *testing.T) {
	ctx := context.Background()

	testSet := types.SetUnknown(types.StringType)

	result, diags := utils.TFStringSetToStringSlice(ctx, testSet)

	if result != nil {
		t.Errorf("Expected nil, got strings [%v]", result)
	}

	if diags.HasError() {
		t.Errorf("Unexpected error: %v", diags)
	}
}

func TestTFStringSetToStringSlice_null(t *testing.T) {
	ctx := context.Background()

	testSet := types.SetNull(types.StringType)

	result, diags := utils.TFStringSetToStringSlice(ctx, testSet)

	if result != nil {
		t.Errorf("Expected nil, got strings [%v]", result)
	}

	if diags.HasError() {
		t.Errorf("Unexpected error: %v", diags)
	}
}
