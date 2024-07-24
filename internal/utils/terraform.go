package utils

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ParseUUID parses the UUID value returning a possible error as Diagnostics instead of error.
func ParseUUID(uuidString string) (uuid.UUID, diag.Diagnostics) {
	var diags diag.Diagnostics

	id, err := uuid.Parse(uuidString)
	if err != nil {
		diags.AddError("Incorrect UUID", fmt.Sprintf("Failed to parse string [%s] as UUID: %v", uuidString, err))
		return uuid.UUID{}, diags
	}

	return id, diags
}

// TFStringSetToStringSlice extracts strings from a string SetValue.
func TFStringSetToStringSlice(ctx context.Context, tfSet basetypes.SetValue) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if tfSet.IsUnknown() {
		return nil, diags
	}

	if tfSet.IsNull() {
		return nil, diags
	}

	tfStrings := make([]types.String, 0, len(tfSet.Elements()))
	diags = tfSet.ElementsAs(ctx, &tfStrings, false)
	if diags.HasError() {
		return nil, diags
	}

	strings := make([]string, len(tfStrings))
	for i, tfString := range tfStrings {
		strings[i] = tfString.ValueString()
	}

	return strings, diags
}
