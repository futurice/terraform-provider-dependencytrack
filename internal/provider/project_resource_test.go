package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/provider"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

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
	// FIXME This test is skipped for now because it puts DT in an inconsistent state.
	//
	//   The inconsistent state manifests so, that when we GET the parent project by ID we see the
	//   child project normally under the `children` field in the response, but when we GET the child project by ID
	//   the response has just an empty `parent: {}` field, without listing the parent uuid. This is a malformed
	//   response, the `parent` field should either contain both `uuid` and `name` fields or be absent altogether if there
	//   is no parent.
	//
	//   I've seen those broken responses in a completely separate REST client (Bruno). Even
	//   DT frontend breaks in this situation - it correctly lists the child project under the
	//   parent in the list, but crashes when navigating to child project details. So this is a problem in
	//   DT, but the mystery is why it occurs only when running this test. Also, it is possible that
	//   a request run with Bruno _will_ get a correct response, and that fixes all the following requests. So, if I put
	//   a breakpoint in ProjectResource.Read (which is called after the two Creates), and then when that breakpoint
	//   is hit just let the program run on, the test will fail. But if I send a request with Bruno while the
	//   test is stopped on that breakpoint, and get a correct response in Bruno (which does not always happen),
	//   and _then_ let the test continue, it will get a correct response in Read and pass.
	//
	//   Testing this with PUT request from Bruno I could not replicate the problem with minimalistic requests
	//   not containing any extra fields, and I was _usually_ not able to replicate it sending the
	//   same requests that our provider sends, with only the parent ID changed (the dtrack lib sends a lot
	//   of unnecessary stuff in these requests). But once I _was_ able to reproduce it in Bruno this way
	//   (unless I've made some mistake and was just confused - as this happened only once).
	//   In the test the bug seems to reliably happen always, if there are not other requests intervening.
	//
	//   What about our requests causes this bug to appear is unclear. Suspecting some race condition I tested putting a
	//   delay between the creation of the parent and of the child, and between creations and reads, but that does not seem to be it.
	//   Another possibility would be that in the context of the test the GET request is run on the same connection as the PUT request, and
	//   that connection does not see correct state, while a new connection would "usually" see it. But it's not clear why we would get
	//   this connection reuse only in the test.
	//
	//   To make this even more weird, I was not able to reproduce the bug:
	//     - when just running `terraform apply` with the CLI with a similar template
	//     - in the two "low level" tests below (was trying to find a minimal case)
	//     - in the "very low level" test below, which side-steps dtrack client lib completely for project creation
	//       and uses verbatim requests that our provider sends, copied from the log - still nothing...
	t.Skip("Skipped due to breaking Dependency-Track")

	//ctx := testutils.CreateTestContext(t)
	//
	//projectResourceName := createProjectResourceName("test")
	//parentProjectResourceName := createProjectResourceName("parent")

	projectName := acctest.RandomWithPrefix("test-project")

	//createTestProject := func(parentID uuid.UUID) dtrack.Project {
	//	return dtrack.Project{
	//		Name:       projectName,
	//		Classifier: "APPLICATION",
	//		Active:     true,
	//		ParentRef:  &dtrack.ParentRef{UUID: parentID},
	//	}
	//}

	//var parentProjectID uuid.UUID

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigParent(testDependencyTrack, projectName),
				// FIXME uncomment and possibly fix/finnish these checks once the config applies correctly
				//Check: resource.ComposeAggregateTestCheckFunc(
				//	testutils.TestAccCheckGetResourceID(parentProjectResourceName, &parentProjectID),
				//	testAccCheckProjectExistsAndHasExpectedData(ctx, testDependencyTrack, projectResourceName, createTestProject(parentProjectID)),
				//	resource.TestCheckResourceAttr(projectResourceName, "parent_id", parentProjectID.String()),
				//),
			},
		},
	})
}

// FIXME remove this "test" when not needed.
func TestAccLowLevel(t *testing.T) {
	ctx := context.Background()

	parentProjectRequest := dtrack.Project{
		Name:       "low-level-parent",
		Classifier: "APPLICATION",
		Active:     true,
	}

	parentProject, err := testDependencyTrack.Client.Project.Create(ctx, parentProjectRequest)
	if err != nil {
		t.Fatalf("Failed to create parent project: %v", err)
	}

	childProjectRequest := dtrack.Project{
		Name:       "low-level-child",
		Classifier: "APPLICATION",
		Active:     true,
		ParentRef: &dtrack.ParentRef{
			UUID: parentProject.UUID,
		},
	}

	childProject, err := testDependencyTrack.Client.Project.Create(ctx, childProjectRequest)
	if err != nil {
		t.Fatalf("Failed to create child project: %v", err)
	}

	readChildProject, err := testDependencyTrack.Client.Project.Get(ctx, childProject.UUID)
	if err != nil {
		t.Fatalf("Failed to read child project: %v", err)
	}

	if readChildProject.ParentRef.UUID != parentProject.UUID {
		t.Errorf("Expected %s got %s", parentProject.UUID.String(), readChildProject.ParentRef.UUID.String())
	}

	// cascades to child
	err = testDependencyTrack.Client.Project.Delete(ctx, parentProject.UUID)
	if err != nil {
		t.Fatalf("Failed to delete parent project: %v", err)
	}
}

// FIXME remove this "test" when not needed.
func TestAccLowLevelTfMapping(t *testing.T) {
	ctx := context.Background()

	parentProjectTf := provider.ProjectResourceModel{
		ID:          types.StringNull(),
		ParentID:    types.StringNull(),
		Name:        types.StringValue("low-level-tf-parent"),
		Classifier:  types.StringValue("APPLICATION"),
		Description: types.StringNull(),
		Active:      types.BoolValue(true),
	}

	parentProjectRequest, diag := provider.TFProjectToDTProject(ctx, parentProjectTf)
	if diag.HasError() {
		t.Fatalf("Failed to map parent project: %v", diag.Errors())
	}

	fmt.Printf("Parent request: %v\n", parentProjectRequest)

	parentProject, err := testDependencyTrack.Client.Project.Create(ctx, parentProjectRequest)
	if err != nil {
		t.Fatalf("Failed to create parent project: %v", err)
	}

	childProjectTf := provider.ProjectResourceModel{
		ID:          types.StringNull(),
		ParentID:    types.StringValue(parentProject.UUID.String()),
		Name:        types.StringValue("low-level-tf-child"),
		Classifier:  types.StringValue("APPLICATION"),
		Description: types.StringNull(),
		Active:      types.BoolValue(true),
	}

	childProjectRequest, diag := provider.TFProjectToDTProject(ctx, childProjectTf)
	if diag.HasError() {
		t.Fatalf("Failed to map child project: %v", diag.Errors())
	}

	fmt.Printf("Child request: %v\n", childProjectRequest)

	childProject, err := testDependencyTrack.Client.Project.Create(ctx, childProjectRequest)
	if err != nil {
		t.Fatalf("Failed to create child project: %v", err)
	}

	readChildProject, err := testDependencyTrack.Client.Project.Get(ctx, childProject.UUID)
	if err != nil {
		t.Fatalf("Failed to read child project: %v", err)
	}

	if readChildProject.ParentRef.UUID != parentProject.UUID {
		t.Errorf("Expected %s got %s", parentProject.UUID.String(), readChildProject.ParentRef.UUID.String())
	}

	// cascades to child
	err = testDependencyTrack.Client.Project.Delete(ctx, parentProject.UUID)
	if err != nil {
		t.Fatalf("Failed to delete parent project: %v", err)
	}
}

// FIXME remove this "test" when not needed.
func TestAccVeryLowLevel(t *testing.T) {
	ctx := context.Background()

	// these requests are copied verbatim from the log when running the failing TestAccProjectResource_parent test
	//   in the child request we need to inject the proper ID
	parentRequestBody := `{"uuid":"00000000-0000-0000-0000-000000000000","name":"parent-test-project-876802509603185655","classifier":"APPLICATION","active":true,"metrics":{"firstOccurrence":0,"lastOccurrence":0,"inheritedRiskScore":0,"vulnerabilities":0,"vulnerableComponents":0,"components":0,"suppressed":0,"critical":0,"high":0,"medium":0,"low":0,"unassigned":0,"findingsTotal":0,"findingsAudited":0,"findingsUnaudited":0,"policyViolationsTotal":0,"policyViolationsFail":0,"policyViolationsWarn":0,"policyViolationsInfo":0,"policyViolationsAudited":0,"policyViolationsUnaudited":0,"policyViolationsSecurityTotal":0,"policyViolationsSecurityAudited":0,"policyViolationsSecurityUnaudited":0,"policyViolationsLicenseTotal":0,"policyViolationsLicenseAudited":0,"policyViolationsLicenseUnaudited":0,"policyViolationsOperationalTotal":0,"policyViolationsOperationalAudited":0,"policyViolationsOperationalUnaudited":0},"lastBomImport":0}`

	fmt.Printf("Parent request body: %s\n", parentRequestBody)

	parentId, err := createProjectVeryLowLevel(testDependencyTrack, parentRequestBody)
	if err != nil {
		t.Fatalf("Failed to create parent project: %v", err)
	}

	childRequestBody := fmt.Sprintf(`{"uuid":"00000000-0000-0000-0000-000000000000","name":"test-project-876802509603185655","classifier":"APPLICATION","active":true,"metrics":{"firstOccurrence":0,"lastOccurrence":0,"inheritedRiskScore":0,"vulnerabilities":0,"vulnerableComponents":0,"components":0,"suppressed":0,"critical":0,"high":0,"medium":0,"low":0,"unassigned":0,"findingsTotal":0,"findingsAudited":0,"findingsUnaudited":0,"policyViolationsTotal":0,"policyViolationsFail":0,"policyViolationsWarn":0,"policyViolationsInfo":0,"policyViolationsAudited":0,"policyViolationsUnaudited":0,"policyViolationsSecurityTotal":0,"policyViolationsSecurityAudited":0,"policyViolationsSecurityUnaudited":0,"policyViolationsLicenseTotal":0,"policyViolationsLicenseAudited":0,"policyViolationsLicenseUnaudited":0,"policyViolationsOperationalTotal":0,"policyViolationsOperationalAudited":0,"policyViolationsOperationalUnaudited":0},"parent":{"uuid":"%s"},"lastBomImport":0}`, parentId)

	fmt.Printf("Child request body: %s\n", childRequestBody)

	childId, err := createProjectVeryLowLevel(testDependencyTrack, childRequestBody)
	if err != nil {
		t.Fatalf("Failed to create child project: %v", err)
	}

	readChildProject, err := testDependencyTrack.Client.Project.Get(ctx, uuid.MustParse(childId))
	if err != nil {
		t.Fatalf("Failed to read child project: %v", err)
	}

	if readChildProject.ParentRef.UUID.String() != parentId {
		t.Errorf("Expected %s got %s", parentId, readChildProject.ParentRef.UUID.String())
	}

	// cascades to child
	err = testDependencyTrack.Client.Project.Delete(ctx, uuid.MustParse(parentId))
	if err != nil {
		t.Fatalf("Failed to delete parent project: %v", err)
	}
}

func createProjectVeryLowLevel(testDependencyTrack *testutils.TestDependencyTrack, requestBody string) (string, error) {
	httpClient := &http.Client{}

	projectUrl, err := url.Parse(fmt.Sprintf("%s/api/v1/project", testDependencyTrack.Endpoint))
	if err != nil {
		return "", fmt.Errorf("failed to build URL: %w", err)
	}

	response, err := httpClient.Do(&http.Request{
		URL:    projectUrl,
		Method: "PUT",
		Header: map[string][]string{
			"Content-Type": {"application/json"},
			"Accept":       {"application/json"},
			"X-API-Key":    {testDependencyTrack.ApiKey},
		},
		Body: io.NopCloser(strings.NewReader(requestBody)),
	})
	if err != nil || response.StatusCode != 201 {
		return "", fmt.Errorf("failed to create project: %w %v", err, response)
	}

	responseBodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("Response body: %s\n", string(responseBodyBytes))

	type JustUuid struct {
		Uuid string `json:"uuid"`
	}

	var projectIDWrapper JustUuid
	err = json.Unmarshal(responseBodyBytes, &projectIDWrapper)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshall response: %w", err)
	}

	return projectIDWrapper.Uuid, nil
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
