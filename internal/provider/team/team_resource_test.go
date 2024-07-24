package team_test

import (
	"context"
	"errors"
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"testing"
)

func TestAccTeamResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	teamName := acctest.RandomWithPrefix("test-team")
	otherTeamName := acctest.RandomWithPrefix("other-test-team")
	teamResourceName := createTeamResourceName("test")

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
					testAccCheckTeamExistsAndHasExpectedData(ctx, testDependencyTrack, teamResourceName, testTeam),
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
					testAccCheckTeamExistsAndHasExpectedData(ctx, testDependencyTrack, teamResourceName, testUpdatedTeam),
					resource.TestCheckResourceAttr(teamResourceName, "name", otherTeamName),
				),
			},
		},
		CheckDestroy: testAccCheckTeamDoesNotExists(ctx, testDependencyTrack, teamResourceName),
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

func testAccCheckTeamExistsAndHasExpectedData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedTeam dtrack.Team) resource.TestCheckFunc {
	return testAccCheckTeamExistsAndHasExpectedLazyData(ctx, testDependencyTrack, resourceName, func() dtrack.Team { return expectedTeam })
}

func testAccCheckTeamExistsAndHasExpectedLazyData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedTeamCreator func() dtrack.Team) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		expectedTeam := expectedTeamCreator()

		team, err := findTeamByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if team == nil {
			return fmt.Errorf("team for resource %s does not exist in Dependency-Track", resourceName)
		}

		diff := cmp.Diff(team, &expectedTeam, cmpopts.IgnoreFields(dtrack.Team{}, "UUID"))
		if diff != "" {
			return fmt.Errorf("team for resource %s is different than expected: %s", resourceName, diff)
		}

		return nil
	}
}

func testAccCheckTeamDoesNotExists(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		team, err := findTeamByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if team != nil {
			return fmt.Errorf("team for resource %s exists in Dependency-Track, even though it shouldn't: %v", resourceName, team)
		}

		return nil
	}
}

func findTeamByResourceName(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, state *terraform.State, resourceName string) (*dtrack.Team, error) {
	teamID, err := testutils.GetResourceID(state, resourceName)
	if err != nil {
		return nil, err
	}

	team, err := findTeam(ctx, testDependencyTrack, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team for resource %s: %w", resourceName, err)
	}

	return team, nil
}

func findTeam(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, teamID uuid.UUID) (*dtrack.Team, error) {
	team, err := testDependencyTrack.Client.Team.Get(ctx, teamID)
	if err != nil {
		var apiErr *dtrack.APIError
		ok := errors.As(err, &apiErr)
		if !ok || apiErr.StatusCode != 404 {
			return nil, fmt.Errorf("failed to get team from Dependency-Track: %w", err)
		}

		return nil, nil
	}

	// normalize the returned object not to contain an empty array reference
	if len(team.Permissions) == 0 {
		team.Permissions = nil
	}

	return &team, nil
}

func createTeamResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_team.%s", localName)
}
