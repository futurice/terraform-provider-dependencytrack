package aclmapping_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/projecttestutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/teamtestutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

func TestAccACLMappingResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	teamName := acctest.RandomWithPrefix("test-team")
	projectName := acctest.RandomWithPrefix("test-project")

	teamResourceName := teamtestutils.CreateTeamResourceName("test")
	otherTeamResourceName := teamtestutils.CreateTeamResourceName("test-other")

	projectResourceName := projecttestutils.CreateProjectResourceName("test")
	otherProjectResourceName := projecttestutils.CreateProjectResourceName("test-other")

	aclMappingResourceName := teamtestutils.CreateACLMappingResourceName("test")

	var teamID, projectID, otherTeamID, otherProjectID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccACLMappingConfigBasic(testDependencyTrack, teamName, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(teamResourceName, &teamID),
					testutils.TestAccCheckGetResourceID(projectResourceName, &projectID),
					teamtestutils.TestAccCheckTeamHasExpectedACLMappings(ctx, testDependencyTrack, teamResourceName, []*string{&projectID}),
					resource.TestCheckResourceAttrPtr(aclMappingResourceName, "team_id", &teamID),
					resource.TestCheckResourceAttrPtr(aclMappingResourceName, "project_id", &projectID),
				),
			},
			{
				ResourceName:      aclMappingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTeamACLMappingConfigOtherTeamAndProject(testDependencyTrack, teamName, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(otherTeamResourceName, &otherTeamID),
					testutils.TestAccCheckGetResourceID(otherProjectResourceName, &otherProjectID),
					teamtestutils.TestAccCheckTeamHasExpectedACLMappings(ctx, testDependencyTrack, teamResourceName, []*string{}),
					teamtestutils.TestAccCheckTeamHasExpectedACLMappings(ctx, testDependencyTrack, otherTeamResourceName, []*string{&otherProjectID}),
					resource.TestCheckResourceAttrPtr(aclMappingResourceName, "team_id", &otherTeamID),
					resource.TestCheckResourceAttrPtr(aclMappingResourceName, "project_id", &otherProjectID),
				),
			},
			{
				Config: testAccACLMappingConfigNoMapping(testDependencyTrack, teamName, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamHasExpectedACLMappings(ctx, testDependencyTrack, teamResourceName, []*string{}),
					teamtestutils.TestAccCheckTeamHasExpectedACLMappings(ctx, testDependencyTrack, otherTeamResourceName, []*string{}),
				),
			},
		},
		// CheckDestroy is not practical here since the team is destroyed as well, and we can no longer query its ACL mappings
	})
}

func testAccACLMappingConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, teamName, projectName string) string {
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
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
			`
resource "dependencytrack_acl_mapping" "test" {
	team_id = dependencytrack_team.test.id
    project_id = dependencytrack_project.test.id
}
`,
		),
	)
}

func testAccTeamACLMappingConfigOtherTeamAndProject(testDependencyTrack *testutils.TestDependencyTrack, teamName, projectName string) string {
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
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test-other" {
	name        = "%[1]s-other"
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
			`
resource "dependencytrack_acl_mapping" "test" {
	team_id = dependencytrack_team.test-other.id
    project_id = dependencytrack_project.test-other.id
}
`,
		),
	)
}

func testAccACLMappingConfigNoMapping(testDependencyTrack *testutils.TestDependencyTrack, teamName, projectName string) string {
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
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test-other" {
	name        = "%[1]s-other"
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
		),
	)
}
