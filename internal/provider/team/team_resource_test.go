// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package team_test

import (
	"fmt"
	"testing"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/teamtestutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	teamName := acctest.RandomWithPrefix("test-team")
	otherTeamName := acctest.RandomWithPrefix("other-test-team")
	teamResourceName := teamtestutils.CreateTeamResourceName("test")

	testTeam := dtrack.Team{
		Name: teamName,
	}

	testUpdatedTeam := testTeam
	testUpdatedTeam.Name = otherTeamName

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfigBasic(testDependencyTrack, teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamExistsAndHasExpectedData(ctx, testDependencyTrack, teamResourceName, testTeam),
					resource.TestCheckResourceAttrSet(teamResourceName, "id"),
					resource.TestCheckResourceAttr(teamResourceName, "name", teamName),
				),
			},
			{
				ResourceName:      teamResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTeamConfigBasic(testDependencyTrack, otherTeamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					teamtestutils.TestAccCheckTeamExistsAndHasExpectedData(ctx, testDependencyTrack, teamResourceName, testUpdatedTeam),
					resource.TestCheckResourceAttr(teamResourceName, "name", otherTeamName),
				),
			},
		},
		CheckDestroy: teamtestutils.TestAccCheckTeamDoesNotExists(ctx, testDependencyTrack, teamResourceName),
	})
}

func testAccTeamConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, teamName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_team" "test" {
	name        = %[1]q
}
`,
			teamName,
		),
	)
}
