// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// IsV5 reports whether the server runs Dependency-Track 5 or later; the client caches the version after the first call.
func IsV5(ctx context.Context, client *dtrack.Client) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	version, err := client.ServerVersion(ctx)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read the Dependency-Track server version, got error: %s", err))
		return false, diags
	}

	major, err := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to parse the Dependency-Track server version %q: %s", version, err))
		return false, diags
	}

	return major >= 5, diags
}

// RequireV5 returns an error diagnostic when the server runs Dependency-Track 4, for resources that only exist on v5.
func RequireV5(ctx context.Context, client *dtrack.Client, resourceType string) diag.Diagnostics {
	v5, diags := IsV5(ctx, client)
	if !diags.HasError() && !v5 {
		diags.AddError("Unsupported Resource", resourceType+" requires Dependency-Track v5 or later.")
	}
	return diags
}
