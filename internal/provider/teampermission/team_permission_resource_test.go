package teampermission_test

import (
	"fmt"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/teamtestutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestAccTeamResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	teamName := acctest.RandomWithPrefix("test-team")
	permissionName := "ACCESS_MANAGEMENT"
	otherPermissionName := "BOM_UPLOAD"

	teamResourceName := teamtestutils.CreateTeamResourceName("test")
	otherTeamResourceName := teamtestutils.CreateTeamResourceName("test-other")
	permissionResourceName := teamtestutils.CreateTeamPermissionResourceName("test")

	var teamID, otherTeamID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamPermissionConfigBasic(testDependencyTrack, teamName, permissionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamHasExpectedPermissions(ctx, testDependencyTrack, teamResourceName, []string{permissionName}),
					testutils.TestAccCheckGetResourceID(teamResourceName, &teamID),
					resource.TestCheckResourceAttrSet(permissionResourceName, "id"),
					resource.TestCheckResourceAttrPtr(permissionResourceName, "team_id", &teamID),
					resource.TestCheckResourceAttr(permissionResourceName, "name", permissionName),
				),
			},
			{
				ResourceName:      permissionResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTeamPermissionConfigBasic(testDependencyTrack, teamName, otherPermissionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamHasExpectedPermissions(ctx, testDependencyTrack, teamResourceName, []string{otherPermissionName}),
					resource.TestCheckResourceAttrSet(permissionResourceName, "id"),
					resource.TestCheckResourceAttrPtr(permissionResourceName, "team_id", &teamID),
					resource.TestCheckResourceAttr(permissionResourceName, "name", otherPermissionName),
				),
			},
			{
				Config: testAccTeamPermissionConfigOtherTeam(testDependencyTrack, teamName, permissionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamHasExpectedPermissions(ctx, testDependencyTrack, teamResourceName, []string{}),
					teamtestutils.TestAccCheckTeamHasExpectedPermissions(ctx, testDependencyTrack, otherTeamResourceName, []string{permissionName}),
					testutils.TestAccCheckGetResourceID(otherTeamResourceName, &otherTeamID),
					resource.TestCheckResourceAttrSet(permissionResourceName, "id"),
					resource.TestCheckResourceAttrPtr(permissionResourceName, "team_id", &otherTeamID),
					resource.TestCheckResourceAttr(permissionResourceName, "name", permissionName),
				),
			},
			{
				Config: testAccTeamPermissionConfigNoPermission(testDependencyTrack, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamHasExpectedPermissions(ctx, testDependencyTrack, teamResourceName, []string{}),
					teamtestutils.TestAccCheckTeamHasExpectedPermissions(ctx, testDependencyTrack, otherTeamResourceName, []string{}),
				),
			},
		},
		// CheckDestroy is not practical here since the team is destroyed as well, and we can no longer query its permissions
	})
}

func testAccTeamPermissionConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, teamName string, permissionName string) string {
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
resource "dependencytrack_team_permission" "test" {
	team_id = dependencytrack_team.test.id
    name = %[1]q
}
`,
				permissionName,
			),
		),
	)
}

func testAccTeamPermissionConfigOtherTeam(testDependencyTrack *testutils.TestDependencyTrack, teamName string, permissionName string) string {
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
			fmt.Sprintf(`
resource "dependencytrack_team_permission" "test" {
	team_id = dependencytrack_team.test-other.id
    name = %[1]q
}
`,
				permissionName,
			),
		),
	)
}

func testAccTeamPermissionConfigNoPermission(testDependencyTrack *testutils.TestDependencyTrack, teamName string) string {
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
