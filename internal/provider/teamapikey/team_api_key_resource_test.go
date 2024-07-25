package teamapikey_test

import (
	"fmt"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/teamtestutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"os"
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

	var teamID, otherTeamID, teamAPIKey, otherTeamAPIKey string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamAPIKeyConfigBasic(testDependencyTrack, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckGetTeamSingleAPIKey(ctx, testDependencyTrack, teamResourceName, &teamAPIKey),
					testutils.TestAccCheckGetResourceID(teamResourceName, &teamID),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "team_id", &teamID),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "value", &teamAPIKey),
				),
			},
			{
				ResourceName: apiKeyResourceName,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					return fmt.Sprintf("%s/%s", teamID, teamAPIKey), nil
				},
				ImportState: true,
				// Unable to verify since the resource has no ID and no non-sensitive ID can be synthesised; we are just smoke-testing the import
				ImportStateVerify: false,
			},
			{
				Config: testAccTeamAPIKeyConfigOtherTeam(testDependencyTrack, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamHasNoAPIKeys(ctx, testDependencyTrack, teamResourceName),
					teamtestutils.TestAccCheckGetTeamSingleAPIKey(ctx, testDependencyTrack, otherTeamResourceName, &otherTeamAPIKey),
					testutils.TestAccCheckGetResourceID(otherTeamResourceName, &otherTeamID),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "team_id", &otherTeamID),
					resource.TestCheckResourceAttrPtr(apiKeyResourceName, "value", &otherTeamAPIKey),
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
