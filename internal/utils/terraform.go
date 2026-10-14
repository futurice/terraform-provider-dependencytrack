// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// ParseAttributeUUID works just as ParseUUID but returns any errors as attribute errors on the specified attribute path.
func ParseAttributeUUID(uuidString string, attributePath string) (uuid.UUID, diag.Diagnostics) {
	var diags diag.Diagnostics

	id, err := uuid.Parse(uuidString)
	if err != nil {
		diags.AddAttributeError(path.Root(attributePath), "Invalid UUID", fmt.Sprintf("Failed to parse string [%s] as UUID: %v", uuidString, err))
		return uuid.UUID{}, diags
	}

	return id, diags
}

// StringValueOrNull returns a null string for an empty value, so optional attributes the server leaves empty stay null.
func StringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
