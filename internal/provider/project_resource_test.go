package provider_test

import (
	"context"
	"errors"
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
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

func TestAccProjectResource(t *testing.T) {
	ctx := context.Background()

	resourceName := "dependencytrack_project.test"
	projectName := acctest.RandomWithPrefix("test-project")

	expectedProject := dtrack.Project{
		Name:        projectName,
		Classifier:  "APPLICATION",
		Description: "Description",
		Active:      true,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceConfig(testDependencyTrack, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckProjectExistsAndHasData(ctx, testDependencyTrack, resourceName, expectedProject),
					resource.TestCheckResourceAttr(resourceName, "name", expectedProject.Name),
					resource.TestCheckResourceAttr(resourceName, "classifier", expectedProject.Classifier),
					resource.TestCheckResourceAttr(resourceName, "description", expectedProject.Description),
					resource.TestCheckResourceAttr(resourceName, "active", strconv.FormatBool(expectedProject.Active)),
					resource.TestCheckNoResourceAttr(resourceName, "parent_id"),
				),
			},
		},
		CheckDestroy: testAccCheckProjectDoesNotExists(ctx, testDependencyTrack, resourceName),
	})
}

func testAccProjectResourceConfig(testDependencyTrack *testutils.TestDependencyTrack, projectName string) string {
	resources := fmt.Sprintf(`
resource "dependencytrack_project" "test" {
  name       = %[1]q
  classifier = "APPLICATION"
  description = "Description"
}
`, projectName)

	return testDependencyTrack.AddProviderConfiguration(resources)
}

func testAccCheckProjectExistsAndHasData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedProject dtrack.Project) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		project, err := findProjectByResourceName(ctx, testDependencyTrack, state, resourceName)
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

func testAccCheckProjectDoesNotExists(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		project, err := findProjectByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if project != nil {
			return fmt.Errorf("project for resource %s exists in Dependency-Track, even though it shouldn't: %v", resourceName, project)
		}

		return nil
	}
}

func findProjectByResourceName(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, state *terraform.State, resourceName string) (*dtrack.Project, error) {
	res, ok := state.RootModule().Resources[resourceName]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", resourceName)
	}

	projectID := uuid.MustParse(res.Primary.ID)

	project, err := findProject(ctx, testDependencyTrack, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project for resource %s: %w", resourceName, err)
	}

	return project, nil
}

func findProject(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, projectID uuid.UUID) (*dtrack.Project, error) {
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
