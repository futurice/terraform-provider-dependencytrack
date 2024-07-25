package teamtestutils

import (
	"context"
	"errors"
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"slices"
)

func TestAccCheckTeamExistsAndHasExpectedData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedTeam dtrack.Team) resource.TestCheckFunc {
	return TestAccCheckTeamExistsAndHasExpectedLazyData(ctx, testDependencyTrack, resourceName, func() dtrack.Team { return expectedTeam })
}

func TestAccCheckTeamExistsAndHasExpectedLazyData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedTeamCreator func() dtrack.Team) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		expectedTeam := expectedTeamCreator()

		team, err := FindTeamByResourceName(ctx, testDependencyTrack, state, resourceName)
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

func TestAccCheckTeamDoesNotExists(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		team, err := FindTeamByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if team != nil {
			return fmt.Errorf("team for resource %s exists in Dependency-Track, even though it shouldn't: %v", resourceName, team)
		}

		return nil
	}
}

func TestAccCheckTeamHasExpectedPermissions(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedPermissions []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		team, err := FindTeamByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if team == nil {
			return fmt.Errorf("team for resource %s does not exist in Dependency-Track", resourceName)
		}

		if len(team.Permissions) != len(expectedPermissions) {
			return fmt.Errorf("team for resource %s has %d permissions instead of the expected %d", resourceName, len(team.Permissions), len(expectedPermissions))
		}

		actualPermissions := make([]string, len(team.Permissions))
		for i, permission := range team.Permissions {
			actualPermissions[i] = permission.Name
		}

		for _, expectedPermission := range expectedPermissions {
			if !slices.Contains(actualPermissions, expectedPermission) {
				return fmt.Errorf("team for resource %s is missing expected permission %s, got [%v]", resourceName, expectedPermission, actualPermissions)
			}
		}

		return nil
	}
}

func FindTeamByResourceName(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, state *terraform.State, resourceName string) (*dtrack.Team, error) {
	teamID, err := testutils.GetResourceID(state, resourceName)
	if err != nil {
		return nil, err
	}

	team, err := FindTeam(ctx, testDependencyTrack, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team for resource %s: %w", resourceName, err)
	}

	return team, nil
}

func FindTeam(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, teamID uuid.UUID) (*dtrack.Team, error) {
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

func CreateTeamResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_team.%s", localName)
}

func CreateTeamPermissionResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_team_permission.%s", localName)
}
