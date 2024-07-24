package utils

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
