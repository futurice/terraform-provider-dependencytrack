package team_test

import (
	"fmt"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/teamtestutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"strings"
	"testing"
)

func TestAccTeamDataSource_basic(t *testing.T) {
	teamName := acctest.RandomWithPrefix("test-team")

	teamResourceName := teamtestutils.CreateTeamResourceName("test")
	teamDataSourceName := teamtestutils.CreateTeamDataSourceName("test")

	var teamID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDataSourceConfigBasic(testDependencyTrack, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(teamResourceName, &teamID),
					resource.TestCheckResourceAttrPtr(teamDataSourceName, "id", &teamID),
					resource.TestCheckResourceAttr(teamDataSourceName, "name", teamName),
				),
			},
		},
	})
}

func TestAccTeamDataSource_permissions(t *testing.T) {
	teamName := acctest.RandomWithPrefix("test-team")
	permissionNames := []string{"ACCESS_MANAGEMENT", "BOM_UPLOAD"}
	teamDataSourceName := teamtestutils.CreateTeamDataSourceName("test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDataSourceConfigPermissions(testDependencyTrack, teamName, permissionNames),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(teamDataSourceName, "permissions.#", "2"),
					resource.TestCheckTypeSetElemAttr(teamDataSourceName, "permissions.*", "ACCESS_MANAGEMENT"),
					resource.TestCheckTypeSetElemAttr(teamDataSourceName, "permissions.*", "BOM_UPLOAD"),
				),
			},
		},
	})
}

func TestAccTeamDataSource_mappedOIDCGroups(t *testing.T) {
	// TODO either add the required resources or add test mappings with direct API calls to complete this test
	//   additionally this simply does not work now, see https://github.com/DependencyTrack/dependency-track/issues/4000
	t.Skip("Currently reading mapped OIDC groups does not work")
}

func testAccTeamDataSourceConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, teamName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_team" "test" {
	name        = %[1]q
}

data "dependencytrack_team" "test" {
	id          = dependencytrack_team.test.id
}
`,
			teamName,
		),
	)
}

func testAccTeamDataSourceConfigPermissions(testDependencyTrack *testutils.TestDependencyTrack, teamName string, permissionNames []string) string {
	permissionResourceNames := make([]string, len(permissionNames))
	permissionResources := make([]string, len(permissionNames))

	for i, permissionName := range permissionNames {
		permissionResourceNames[i] = fmt.Sprintf("dependencytrack_team_permission.test-%[1]s", permissionName)

		permissionResources[i] = fmt.Sprintf(`
resource "dependencytrack_team_permission" "test-%[1]s" {
	team_id = dependencytrack_team.test.id
    name = %[1]q
}
`,
			permissionName,
		)
	}

	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_team" "test" {
	name        = %[1]q
}
`,
				teamName,
			),
			testutils.ComposeConfigs(permissionResources...),
			fmt.Sprintf(`
data "dependencytrack_team" "test" {
	id          = dependencytrack_team.test.id
	depends_on  = [%[1]s]
}
`,
				strings.Join(permissionResourceNames, ", "),
			),
		),
	)
}
