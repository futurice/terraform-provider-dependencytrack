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

	v5, err := IsV5Version(version)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to parse the Dependency-Track server version: %s", err))
		return false, diags
	}

	return v5, diags
}

// IsV5Version reports whether a version such as "5.2.0" is Dependency-Track 5 or later, by its major number.
func IsV5Version(version string) (bool, error) {
	major, err := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	if err != nil {
		return false, fmt.Errorf("version %q has no numeric major: %w", version, err)
	}

	return major >= 5, nil
}

// RequireV5 returns an error diagnostic when the server runs Dependency-Track 4, for resources that only exist on v5.
func RequireV5(ctx context.Context, client *dtrack.Client, resourceType string) diag.Diagnostics {
	v5, diags := IsV5(ctx, client)
	if !diags.HasError() && !v5 {
		diags.AddError("Unsupported Resource", resourceType+" requires Dependency-Track v5 or later.")
	}
	return diags
}
