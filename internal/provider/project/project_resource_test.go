package project_test

import (
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/projecttestutils"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

	projectName := acctest.RandomWithPrefix("test-project")
	otherProjectName := acctest.RandomWithPrefix("other-test-project")
	projectResourceName := projecttestutils.CreateProjectResourceName("test")

	testProject := dtrack.Project{
		Name:       projectName,
		Classifier: "APPLICATION",
		Active:     true,
	}

	testUpdatedProject := testProject
	testUpdatedProject.Name = otherProjectName

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigBasic(testDependencyTrack, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testProject),
					resource.TestCheckResourceAttrSet(projectResourceName, "id"),
					resource.TestCheckResourceAttr(projectResourceName, "name", projectName),
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
			{
				Config: testAccProjectConfigBasic(testDependencyTrack, otherProjectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testUpdatedProject),
					resource.TestCheckResourceAttr(projectResourceName, "name", otherProjectName),
				),
			},
		},
		CheckDestroy: projecttestutils.TestAccCheckProjectDoesNotExists(ctx, testDependencyTrack, projectResourceName),
	})
}

func TestAccProjectResource_description(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	projectName := acctest.RandomWithPrefix("test-project")
	projectResourceName := projecttestutils.CreateProjectResourceName("test")

	testProject := dtrack.Project{
		Name:        projectName,
		Classifier:  "APPLICATION",
		Description: "Description",
		Active:      true,
	}

	testUpdatedProject := testProject
	testUpdatedProject.Description = "Other description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigDescription(testDependencyTrack, projectName, testProject.Description),
				Check: resource.ComposeAggregateTestCheckFunc(
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testProject),
					resource.TestCheckResourceAttr(projectResourceName, "description", testProject.Description),
				),
			},
			{
				Config: testAccProjectConfigDescription(testDependencyTrack, projectName, testUpdatedProject.Description),
				Check: resource.ComposeAggregateTestCheckFunc(
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testUpdatedProject),
					resource.TestCheckResourceAttr(projectResourceName, "description", testUpdatedProject.Description),
				),
			},
		},
	})
}

func TestAccProjectResource_inactive(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	projectName := acctest.RandomWithPrefix("test-project")
	projectResourceName := projecttestutils.CreateProjectResourceName("test")

	testProject := dtrack.Project{
		Name:       projectName,
		Classifier: "APPLICATION",
		Active:     false,
	}

	testUpdatedProject := testProject
	testUpdatedProject.Active = true

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigActivity(testDependencyTrack, projectName, testProject.Active),
				Check: resource.ComposeAggregateTestCheckFunc(
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testProject),
					resource.TestCheckResourceAttr(projectResourceName, "active", strconv.FormatBool(testProject.Active)),
				),
			},
			{
				Config: testAccProjectConfigActivity(testDependencyTrack, projectName, testUpdatedProject.Active),
				Check: resource.ComposeAggregateTestCheckFunc(
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testUpdatedProject),
					resource.TestCheckResourceAttr(projectResourceName, "active", strconv.FormatBool(testUpdatedProject.Active)),
				),
			},
		},
	})
}

func TestAccProjectResource_classifier(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	projectName := acctest.RandomWithPrefix("test-project")
	projectResourceName := projecttestutils.CreateProjectResourceName("test")

	testProject := dtrack.Project{
		Name:       projectName,
		Classifier: "CONTAINER",
		Active:     true,
	}

	testUpdatedProject := testProject
	testUpdatedProject.Classifier = "DEVICE"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigClassifier(testDependencyTrack, testProject.Name, testProject.Classifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testProject),
					resource.TestCheckResourceAttr(projectResourceName, "classifier", testProject.Classifier),
				),
			},
			{
				Config: testAccProjectConfigClassifier(testDependencyTrack, testProject.Name, testUpdatedProject.Classifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, testUpdatedProject),
					resource.TestCheckResourceAttr(projectResourceName, "classifier", testUpdatedProject.Classifier),
				),
			},
		},
	})
}

func TestAccProjectResource_parent(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	projectResourceName := projecttestutils.CreateProjectResourceName("test")
	parentProjectResourceName := projecttestutils.CreateProjectResourceName("parent")
	otherParentProjectResourceName := projecttestutils.CreateProjectResourceName("other_parent")

	projectName := acctest.RandomWithPrefix("test-project")

	createTestProject := func(parentID *string) dtrack.Project {
		return dtrack.Project{
			Name:       projectName,
			Classifier: "APPLICATION",
			Active:     true,
			ParentRef:  &dtrack.ParentRef{UUID: uuid.MustParse(*parentID)},
		}
	}

	var parentProjectID, otherParentProjectID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigParent(testDependencyTrack, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(parentProjectResourceName, &parentProjectID),
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedLazyData(ctx, testDependencyTrack, projectResourceName, func() dtrack.Project { return createTestProject(&parentProjectID) }),
					resource.TestCheckResourceAttrPtr(projectResourceName, "parent_id", &parentProjectID),
				),
			},
			{
				Config: testAccProjectConfigOtherParent(testDependencyTrack, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(otherParentProjectResourceName, &otherParentProjectID),
					projecttestutils.TestAccCheckProjectExistsAndHasExpectedLazyData(ctx, testDependencyTrack, projectResourceName, func() dtrack.Project { return createTestProject(&otherParentProjectID) }),
					resource.TestCheckResourceAttrPtr(projectResourceName, "parent_id", &otherParentProjectID),
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

func testAccProjectConfigActivity(testDependencyTrack *testutils.TestDependencyTrack, projectName string, active bool) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
	active		= %[2]t
}
`,
			projectName, active,
		),
	)
}

func testAccProjectConfigClassifier(testDependencyTrack *testutils.TestDependencyTrack, projectName, classifier string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = %[2]q
}
`,
			projectName, classifier,
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

func testAccProjectConfigOtherParent(testDependencyTrack *testutils.TestDependencyTrack, projectName string) string {
	parentProjectName := fmt.Sprintf("parent-%s", projectName)
	otherParentProjectName := fmt.Sprintf("other-parent-%s", projectName)

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
resource "dependencytrack_project" "other_parent" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
				otherParentProjectName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        	= %[1]q
	classifier  	= "APPLICATION"
	parent_id		= dependencytrack_project.other_parent.id
}
`,
				projectName,
			),
		),
	)
}
