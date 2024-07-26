package projecttestutils

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
)

func TestAccCheckProjectExistsAndHasExpectedData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedProject dtrack.Project) resource.TestCheckFunc {
	return TestAccCheckProjectExistsAndHasExpectedLazyData(ctx, testDependencyTrack, resourceName, func() dtrack.Project { return expectedProject })
}

func TestAccCheckProjectExistsAndHasExpectedLazyData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedProjectCreator func() dtrack.Project) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		expectedProject := expectedProjectCreator()

		project, err := FindProjectByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if project == nil {
			return fmt.Errorf("project for resource %s does not exist in Dependency-Track", resourceName)
		}

		diff := cmp.Diff(project, &expectedProject, cmpopts.IgnoreFields(dtrack.Project{}, "UUID", "Properties", "Tags", "Metrics"))
		if diff != "" {
			return fmt.Errorf("project for resource %s is different than expected: %s", resourceName, diff)
		}

		return nil
	}
}

func TestAccCheckProjectDoesNotExists(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		project, err := FindProjectByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if project != nil {
			return fmt.Errorf("project for resource %s exists in Dependency-Track, even though it shouldn't: %v", resourceName, project)
		}

		return nil
	}
}

func FindProjectByResourceName(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, state *terraform.State, resourceName string) (*dtrack.Project, error) {
	projectID, err := testutils.GetResourceID(state, resourceName)
	if err != nil {
		return nil, err
	}

	project, err := FindProject(ctx, testDependencyTrack, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project for resource %s: %w", resourceName, err)
	}

	return project, nil
}

func FindProject(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, projectID uuid.UUID) (*dtrack.Project, error) {
	project, err := testDependencyTrack.Client.Project.Get(ctx, projectID)
	if err != nil {
		var apiErr *dtrack.APIError
		ok := errors.As(err, &apiErr)
		if !ok || apiErr.StatusCode != 404 {
			return nil, fmt.Errorf("failed to get project from Dependency-Track: %w", err)
		}

		return nil, nil
	}

	return &project, nil
}

func CreateProjectResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_project.%s", localName)
}
