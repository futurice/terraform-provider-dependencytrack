package provider_test

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
	"os"
	"strconv"
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

func TestAccProjectResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	projectResourceName := createProjectResourceName("test")

	testProject := dtrack.Project{
		Name:       acctest.RandomWithPrefix("test-project"),
		Classifier: "APPLICATION",
		Active:     true,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigBasic(testDependencyTrack, testProject.Name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testProject),
					resource.TestCheckResourceAttrSet(projectResourceName, "id"),
					resource.TestCheckResourceAttr(projectResourceName, "name", testProject.Name),
					resource.TestCheckResourceAttr(projectResourceName, "classifier", testProject.Classifier),
					resource.TestCheckNoResourceAttr(projectResourceName, "description"),
					resource.TestCheckResourceAttr(projectResourceName, "active", strconv.FormatBool(testProject.Active)),
					resource.TestCheckNoResourceAttr(projectResourceName, "parent_id"),
				),
			},
			{
				ResourceName:      projectResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testAccCheckProjectDoesNotExists(ctx, testDependencyTrack, projectResourceName),
	})
}

func TestAccProjectResource_description(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	projectResourceName := createProjectResourceName("test")

	testProject := dtrack.Project{
		Name:        acctest.RandomWithPrefix("test-project"),
		Classifier:  "APPLICATION",
		Description: "Description",
		Active:      true,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigDescription(testDependencyTrack, testProject.Name, testProject.Description),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testProject),
					resource.TestCheckResourceAttr(projectResourceName, "description", testProject.Description),
				),
			},
		},
	})
}

func TestAccProjectResource_inactive(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	projectResourceName := createProjectResourceName("test")

	testProject := dtrack.Project{
		Name:       acctest.RandomWithPrefix("test-project"),
		Classifier: "APPLICATION",
		Active:     false,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigInactive(testDependencyTrack, testProject.Name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testProject),
					resource.TestCheckResourceAttr(projectResourceName, "active", strconv.FormatBool(false)),
				),
			},
		},
	})
}

func TestAccProjectResource_parent(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	childProjectResourceName := createProjectResourceName("test")
	parentProjectResourceName := createProjectResourceName("parent")

	projectName := acctest.RandomWithPrefix("test-project")

	createTestProject := func(parentID *string) dtrack.Project {
		return dtrack.Project{
			Name:       projectName,
			Classifier: "APPLICATION",
			Active:     true,
			ParentRef:  &dtrack.ParentRef{UUID: uuid.MustParse(*parentID)},
		}
	}

	var parentProjectID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigParent(testDependencyTrack, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(parentProjectResourceName, &parentProjectID),
					testAccCheckProjectExistsAndHasExpectedLazyData(ctx, testDependencyTrack, childProjectResourceName, func() dtrack.Project { return createTestProject(&parentProjectID) }),
					resource.TestCheckResourceAttrPtr(childProjectResourceName, "parent_id", &parentProjectID),
				),
			},
		},
	})
}

func testAccProjectConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, projectName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
			projectName,
		),
	)
}

func testAccProjectConfigDescription(testDependencyTrack *testutils.TestDependencyTrack, projectName, projectDescription string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
	description = %[2]q
}
`,
			projectName,
			projectDescription,
		),
	)
}

func testAccProjectConfigInactive(testDependencyTrack *testutils.TestDependencyTrack, projectName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
	active		= false
}
`,
			projectName,
		),
	)
}

func testAccProjectConfigParent(testDependencyTrack *testutils.TestDependencyTrack, projectName string) string {
	parentProjectName := fmt.Sprintf("parent-%s", projectName)

	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_project" "parent" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
				parentProjectName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        	= %[1]q
	classifier  	= "APPLICATION"
	parent_id		= dependencytrack_project.parent.id
}
`,
				projectName,
			),
		),
	)
}

func testAccCheckProjectExistsAndHasExpectedData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedProject dtrack.Project) resource.TestCheckFunc {
	return testAccCheckProjectExistsAndHasExpectedLazyData(ctx, testDependencyTrack, resourceName, func() dtrack.Project { return expectedProject })
}

func testAccCheckProjectExistsAndHasExpectedLazyData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedProjectCreator func() dtrack.Project) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		expectedProject := expectedProjectCreator()

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
	projectID, err := testutils.GetResourceID(state, resourceName)
	if err != nil {
		return nil, err
	}

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

func createProjectResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_project.%s", localName)
}
