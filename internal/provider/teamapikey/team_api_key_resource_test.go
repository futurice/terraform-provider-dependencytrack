// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package teamapikey_test

import (
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/teamtestutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"os"
	"strconv"
	"testing"
)

var testDependencyTrack *testutils.TestDependencyTrack

func TestMain(m *testing.M) {
	if os.Getenv(resource.EnvTfAcc) != "" {
		var cleanup func()
		testDependencyTrack, cleanup = testutils.InitTestDependencyTrack()
		defer cleanup()
	}

	m.Run()
}

func TestAccTeamAPIKeyResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	teamName := acctest.RandomWithPrefix("test-team")

	teamResourceName := teamtestutils.CreateTeamResourceName("test")
	otherTeamResourceName := teamtestutils.CreateTeamResourceName("test-other")
	apiKeyResourceName := teamtestutils.CreateTeamAPIKeyResourceName("test")

	var teamAPIKey, otherTeamAPIKey dtrack.APIKey
	var teamID, otherTeamID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamAPIKeyConfigBasic(testDependencyTrack, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckGetTeamSingleAPIKey(ctx, testDependencyTrack, teamResourceName, &teamAPIKey),
					testutils.TestAccCheckGetResourceID(teamResourceName, &teamID),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "id", &teamAPIKey.PublicId),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "team_id", &teamID),
					resource.TestCheckResourceAttr(apiKeyResourceName, "legacy", strconv.FormatBool(teamAPIKey.Legacy)),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "comment", &teamAPIKey.Comment),
				),
			},
			{
				ResourceName: apiKeyResourceName,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					return fmt.Sprintf("%s/%s", teamID, teamAPIKey.PublicId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
			{
				Config: testAccTeamAPIKeyConfigOtherTeam(testDependencyTrack, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamHasNoAPIKeys(ctx, testDependencyTrack, teamResourceName),
					teamtestutils.TestAccCheckGetTeamSingleAPIKey(ctx, testDependencyTrack, otherTeamResourceName, &otherTeamAPIKey),
					testutils.TestAccCheckGetResourceID(otherTeamResourceName, &otherTeamID),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "id", &otherTeamAPIKey.PublicId),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "team_id", &otherTeamID),
					resource.TestCheckResourceAttr(apiKeyResourceName, "legacy", strconv.FormatBool(otherTeamAPIKey.Legacy)),
				),
			},
			{
				Config: testAccTeamAPIKeyConfigNoAPIKey(testDependencyTrack, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamHasNoAPIKeys(ctx, testDependencyTrack, teamResourceName),
					teamtestutils.TestAccCheckTeamHasNoAPIKeys(ctx, testDependencyTrack, otherTeamResourceName),
				),
			},
		},
		// CheckDestroy is not practical here since the team is destroyed as well, and we can no longer query its API Keys
	})
}

func testAccTeamAPIKeyConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, teamName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_team" "test" {
	name        = %[1]q
}
`,
				teamName,
			),
			`
resource "dependencytrack_team_api_key" "test" {
	team_id = dependencytrack_team.test.id
    comment	 	= "This is a test API key"
}
`,
		),
	)
}

func testAccTeamAPIKeyConfigOtherTeam(testDependencyTrack *testutils.TestDependencyTrack, teamName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_team" "test" {
	name        = %[1]q
}
`,
				teamName,
			),
			fmt.Sprintf(`
resource "dependencytrack_team" "test-other" {
	name        = "%[1]s-other"
}
`,
				teamName,
			),
			`
resource "dependencytrack_team_api_key" "test" {
	team_id = dependencytrack_team.test-other.id
}
`,
		),
	)
}

func testAccTeamAPIKeyConfigNoAPIKey(testDependencyTrack *testutils.TestDependencyTrack, teamName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_team" "test" {
	name        = %[1]q
}
`,
				teamName,
			),
			fmt.Sprintf(`
resource "dependencytrack_team" "test-other" {
	name        = "%[1]s-other"
}
`,
				teamName,
			),
		),
	)
}
