package teamtestutils

import (
	"context"
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

func TestAccCheckTeamHasExpectedACLMappings(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedACLMappingProjectIDs []*string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		team, err := FindTeamByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if team == nil {
			return fmt.Errorf("team for resource %s does not exist in Dependency-Track", resourceName)
		}

		aclMappings, err := testDependencyTrack.Client.ACLMapping.Get(ctx, team.UUID)
		if err != nil {
			return err
		}

		if len(aclMappings) != len(expectedACLMappingProjectIDs) {
			return fmt.Errorf("team for resource %s has %d permissions instead of the expected %d", resourceName, len(aclMappings), len(expectedACLMappingProjectIDs))
		}

		actualACLMappingProjectIDs := make([]string, len(aclMappings))
		for i, aclMapping := range aclMappings {
			actualACLMappingProjectIDs[i] = aclMapping.UUID.String()
		}

		for _, expectedACLMappingProjectID := range expectedACLMappingProjectIDs {
			if !slices.Contains(actualACLMappingProjectIDs, *expectedACLMappingProjectID) {
				return fmt.Errorf("team for resource %s is missing expected permission %s, got [%v]", resourceName, *expectedACLMappingProjectID, actualACLMappingProjectIDs)
			}
		}

		return nil
	}
}

func TestAccCheckTeamHasNoAPIKeys(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		team, err := FindTeamByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if team == nil {
			return fmt.Errorf("team for resource %s does not exist in Dependency-Track", resourceName)
		}

		if len(team.APIKeys) != 0 {
			return fmt.Errorf("team for resource %s has %d API keys instead of the expected 0", resourceName, len(team.APIKeys))
		}

		return nil
	}
}

func TestAccCheckGetTeamSingleAPIKey(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, apiKeyTarget *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		team, err := FindTeamByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if team == nil {
			return fmt.Errorf("team for resource %s does not exist in Dependency-Track", resourceName)
		}

		if len(team.APIKeys) != 1 {
			return fmt.Errorf("team for resource %s has %d API keys instead of the expected 1", resourceName, len(team.APIKeys))
		}

		*apiKeyTarget = team.APIKeys[0].Key

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
	// Currently the endpoint for getting one team does not return most of the data
	//   see https://github.com/DependencyTrack/dependency-track/issues/4000
	teams, err := testDependencyTrack.Client.Team.GetAll(ctx, dtrack.PageOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get teams from Dependency-Track: %w", err)
	}

	for _, team := range teams.Items {
		if team.UUID == teamID {
			// normalize the returned object not to contain empty array references
			if len(team.Permissions) == 0 {
				team.Permissions = nil
			}
			if len(team.APIKeys) == 0 {
				team.APIKeys = nil
			}
			if len(team.MappedOIDCGroups) == 0 {
				team.MappedOIDCGroups = nil
			}

			return &team, nil
		}
	}

	return nil, nil
}

func CreateTeamResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_team.%s", localName)
}

func CreateTeamDataSourceName(localName string) string {
	return fmt.Sprintf("data.dependencytrack_team.%s", localName)
}

func CreateTeamPermissionResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_team_permission.%s", localName)
}

func CreateTeamAPIKeyResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_team_api_key.%s", localName)
}

func CreateACLMappingResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_acl_mapping.%s", localName)
}
