// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package extensionconfig_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const extensionConfigResourceName = "dependencytrack_extension_config.test"

func TestAccExtensionConfigResource_basic(t *testing.T) {
	testDependencyTrack.SkipUnlessV5(t)
	ctx := testutils.CreateTestContext(t)

	secretName := acctest.RandomWithPrefix("test-trivy-token")
	config := `{"enabled": true, "apiUrl": "http://localhost:8081", "apiToken": "%s", "ignoreUnfixed": false, "scanLibrary": true, "scanOs": true}`
	updatedConfig := `{"enabled": true, "apiUrl": "http://localhost:8081", "apiToken": "%s", "ignoreUnfixed": true, "scanLibrary": true, "scanOs": false}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{
				Config: testAccExtensionConfigWithSecret(secretName, config),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckTrivyConfig(ctx, fmt.Sprintf(config, secretName)),
					resource.TestCheckResourceAttr(extensionConfigResourceName, "id", "vuln-analyzer/trivy"),
					resource.TestCheckResourceAttr(extensionConfigResourceName, "config", fmt.Sprintf(config, secretName)),
				),
			},
			{
				ResourceName:      extensionConfigResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Import has no configured string to keep, so it stores the server's compact JSON.
				ImportStateVerifyIgnore: []string{"config"},
			},
			{
				Config: testAccExtensionConfigWithSecret(secretName, updatedConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckTrivyConfig(ctx, fmt.Sprintf(updatedConfig, secretName)),
					resource.TestCheckResourceAttr(extensionConfigResourceName, "config", fmt.Sprintf(updatedConfig, secretName)),
				),
			},
			{
				// Destroying the resource only forgets it, so the server keeps the last config.
				Config: testAccExtensionConfigSecretOnly(secretName),
				Check:  testAccCheckTrivyConfig(ctx, fmt.Sprintf(updatedConfig, secretName)),
			},
		},
	})
}

func TestAccExtensionConfigResource_missingSecret(t *testing.T) {
	testDependencyTrack.SkipUnlessV5(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccExtensionConfig(`{"enabled": true, "apiUrl": "http://localhost:8081", "apiToken": "no-such-secret"}`),
				ExpectError: regexp.MustCompile("does not exist"),
			},
		},
	})
}

func TestAccExtensionConfigResource_notAnObject(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccExtensionConfig(`["enabled"]`),
				ExpectError: regexp.MustCompile("config must be a JSON object"),
			},
		},
	})
}

func TestAccExtensionConfigResource_requiresV5(t *testing.T) {
	if testDependencyTrack.IsV5() {
		t.Skip("checks the error on Dependency-Track v4")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccExtensionConfig(`{"enabled": false}`),
				ExpectError: regexp.MustCompile("requires Dependency-Track v5"),
			},
		},
	})
}

func testAccExtensionConfig(config string) string {
	return testDependencyTrack.AddProviderConfiguration(fmt.Sprintf(`
resource "dependencytrack_extension_config" "test" {
	extension_point = "vuln-analyzer"
	extension       = "trivy"
	config          = %q
}
`, config))
}

func testAccExtensionConfigSecretOnly(secretName string) string {
	return testDependencyTrack.AddProviderConfiguration(fmt.Sprintf(`
resource "dependencytrack_secret" "token" {
	name             = %q
	value_wo         = "token"
	value_wo_version = 1
}
`, secretName))
}

// testAccExtensionConfigWithSecret fills the secret's name into configFormat through a reference, so Terraform orders the two.
func testAccExtensionConfigWithSecret(secretName, configFormat string) string {
	return testutils.ComposeConfigs(
		testAccExtensionConfigSecretOnly(secretName),
		fmt.Sprintf(`
resource "dependencytrack_extension_config" "test" {
	extension_point = "vuln-analyzer"
	extension       = "trivy"
	config          = format(%q, dependencytrack_secret.token.name)
}
`, configFormat),
	)
}

func testAccCheckTrivyConfig(ctx context.Context, expected string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		config, err := testDependencyTrack.Client.Extension.GetConfig(ctx, "vuln-analyzer", "trivy")
		if err != nil {
			return fmt.Errorf("failed to get Trivy config: %w", err)
		}
		if !utils.JSONEqual(string(config), expected) {
			return fmt.Errorf("Trivy config is %s, expected %s", config, expected)
		}
		return nil
	}
}
