// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package secret_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const secretResourceName = "dependencytrack_secret.test"

func TestAccSecretResource_basic(t *testing.T) {
	testDependencyTrack.SkipUnlessV5(t)
	ctx := testutils.CreateTestContext(t)

	name := acctest.RandomWithPrefix("test-secret")
	var created dtrack.Secret

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{
				Config: testAccSecretConfig(name, "Some description", "first", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSecret(ctx, name, "Some description", &created),
					resource.TestCheckResourceAttr(secretResourceName, "name", name),
					resource.TestCheckResourceAttr(secretResourceName, "description", "Some description"),
					resource.TestCheckResourceAttr(secretResourceName, "value_wo_version", "1"),
					resource.TestCheckNoResourceAttr(secretResourceName, "value_wo"),
				),
			},
			{
				ResourceName:                         secretResourceName,
				ImportState:                          true,
				ImportStateId:                        name,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				// Write-only values and their version never come back from the server.
				ImportStateVerifyIgnore: []string{"value_wo", "value_wo_version"},
			},
			{
				// A new version with the same description must still send the value.
				Config: testAccSecretConfig(name, "Some description", "second", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSecretUpdatedSince(ctx, name, &created),
					resource.TestCheckResourceAttr(secretResourceName, "value_wo_version", "2"),
				),
			},
			{
				Config: testAccSecretConfig(name, "Other description", "second", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSecret(ctx, name, "Other description", nil),
					resource.TestCheckResourceAttr(secretResourceName, "description", "Other description"),
				),
			},
		},
		CheckDestroy: testAccCheckSecretDoesNotExist(ctx, name),
	})
}

func TestAccSecretResource_invalidName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{
				Config:      testAccSecretConfig("with.dot", "", "value", 1),
				ExpectError: regexp.MustCompile("must be 1 to 64 letters"),
			},
		},
	})
}

func TestAccSecretResource_requiresV5(t *testing.T) {
	if testDependencyTrack.IsV5() {
		t.Skip("checks the error on Dependency-Track v4")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{
				Config:      testAccSecretConfig(acctest.RandomWithPrefix("test-secret"), "", "value", 1),
				ExpectError: regexp.MustCompile("requires Dependency-Track v5"),
			},
		},
	})
}

func testAccSecretConfig(name, description, value string, version int) string {
	descriptionLine := ""
	if description != "" {
		descriptionLine = fmt.Sprintf("description      = %q", description)
	}

	return testDependencyTrack.AddProviderConfiguration(fmt.Sprintf(`
resource "dependencytrack_secret" "test" {
	name             = %[1]q
	%[2]s
	value_wo         = %[3]q
	value_wo_version = %[4]d
}
`, name, descriptionLine, value, version))
}

// testAccCheckSecret checks the secret exists with the description, and records it in out if given.
func testAccCheckSecret(ctx context.Context, name, description string, out *dtrack.Secret) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		secret, err := testDependencyTrack.Client.Secret.Get(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to get secret %s: %w", name, err)
		}
		if secret.Description != description {
			return fmt.Errorf("secret %s has description %q, expected %q", name, secret.Description, description)
		}
		if out != nil {
			*out = secret
		}
		return nil
	}
}

// testAccCheckSecretUpdatedSince checks the server recorded an update after the secret was created.
func testAccCheckSecretUpdatedSince(ctx context.Context, name string, created *dtrack.Secret) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		secret, err := testDependencyTrack.Client.Secret.Get(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to get secret %s: %w", name, err)
		}
		if secret.UpdatedAt == 0 || secret.UpdatedAt == created.UpdatedAt {
			return fmt.Errorf("secret %s was not updated, updated_at is %d", name, secret.UpdatedAt)
		}
		return nil
	}
}

func testAccCheckSecretDoesNotExist(ctx context.Context, name string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		_, err := testDependencyTrack.Client.Secret.Get(ctx, name)
		var apiErr *dtrack.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to get secret %s: %w", name, err)
		}
		return fmt.Errorf("secret %s still exists", name)
	}
}
