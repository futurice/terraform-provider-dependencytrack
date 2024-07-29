package notificationruleproject_test

import (
	"fmt"
	notificationruletestutils "github.com/futurice/terraform-provider-dependencytrack/internal/testutils/notificationrule"
	"os"
	"testing"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils/projecttestutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

func TestAccNotificationRuleProjectResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	publisherName := acctest.RandomWithPrefix("test-notification-publisher")
	ruleName := acctest.RandomWithPrefix("test-notification-rule")
	projectName := acctest.RandomWithPrefix("test-project")

	ruleResourceName := notificationruletestutils.CreateNotificationRuleResourceName("test")
	otherRuleResourceName := notificationruletestutils.CreateNotificationRuleResourceName("test-other")

	projectResourceName := projecttestutils.CreateProjectResourceName("test")
	otherProjectResourceName := projecttestutils.CreateProjectResourceName("test-other")

	notificationRuleProjectResourceName := notificationruletestutils.CreateNotificationRuleProjectResourceName("test")

	var ruleID, projectID, otherRuleID, otherProjectID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRuleProjectConfigBasic(testDependencyTrack, publisherName, ruleName, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(ruleResourceName, &ruleID),
					testutils.TestAccCheckGetResourceID(projectResourceName, &projectID),
					notificationruletestutils.TestAccCheckNotificationRuleHasExpectedProjects(ctx, testDependencyTrack, ruleResourceName, []*string{&projectID}),
					resource.TestCheckResourceAttrPtr(notificationRuleProjectResourceName, "rule_id", &ruleID),
					resource.TestCheckResourceAttrPtr(notificationRuleProjectResourceName, "project_id", &projectID),
				),
			},
			{
				ResourceName:      notificationRuleProjectResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNotificationRuleProjectConfigOtherRuleAndProject(testDependencyTrack, publisherName, ruleName, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(otherRuleResourceName, &otherRuleID),
					testutils.TestAccCheckGetResourceID(otherProjectResourceName, &otherProjectID),
					notificationruletestutils.TestAccCheckNotificationRuleHasExpectedProjects(ctx, testDependencyTrack, ruleResourceName, []*string{}),
					notificationruletestutils.TestAccCheckNotificationRuleHasExpectedProjects(ctx, testDependencyTrack, otherRuleResourceName, []*string{&otherProjectID}),
					resource.TestCheckResourceAttrPtr(notificationRuleProjectResourceName, "rule_id", &otherRuleID),
					resource.TestCheckResourceAttrPtr(notificationRuleProjectResourceName, "project_id", &otherProjectID),
				),
			},
			{
				Config: testAccNotificationRuleProjectConfigNoProject(testDependencyTrack, publisherName, ruleName, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					notificationruletestutils.TestAccCheckNotificationRuleHasExpectedProjects(ctx, testDependencyTrack, ruleResourceName, []*string{}),
					notificationruletestutils.TestAccCheckNotificationRuleHasExpectedProjects(ctx, testDependencyTrack, otherRuleResourceName, []*string{}),
				),
			},
		},
		// CheckDestroy is not practical here since the notification rule is destroyed as well, and we can no longer query its projects
	})
}

func testAccNotificationRuleProjectConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, providerName, ruleName, projectName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	publisher_class    = "org.dependencytrack.notification.publisher.SlackPublisher"
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName,
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_rule" "test" {
	name               = %[1]q
	publisher_id       = dependencytrack_notification_publisher.test.id
	scope              = "PORTFOLIO"
	notification_level = "INFORMATIONAL"
}
`,
				ruleName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
			`
resource "dependencytrack_notification_rule_project" "test" {
	rule_id = dependencytrack_notification_rule.test.id
    project_id = dependencytrack_project.test.id
}
`,
		),
	)
}

func testAccNotificationRuleProjectConfigOtherRuleAndProject(testDependencyTrack *testutils.TestDependencyTrack, providerName, ruleName, projectName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	publisher_class    = "org.dependencytrack.notification.publisher.SlackPublisher"
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName,
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_rule" "test" {
	name               = %[1]q
	publisher_id       = dependencytrack_notification_publisher.test.id
	scope              = "PORTFOLIO"
	notification_level = "INFORMATIONAL"
}
`,
				ruleName,
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_rule" "test-other" {
	name               = "%[1]s-other"
	publisher_id       = dependencytrack_notification_publisher.test.id
	scope              = "PORTFOLIO"
	notification_level = "INFORMATIONAL"
}
`,
				ruleName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test-other" {
	name        = "%[1]s-other"
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
			`
resource "dependencytrack_notification_rule_project" "test" {
	rule_id = dependencytrack_notification_rule.test-other.id
    project_id = dependencytrack_project.test-other.id
}
`,
		),
	)
}

func testAccNotificationRuleProjectConfigNoProject(testDependencyTrack *testutils.TestDependencyTrack, providerName, ruleName, projectName string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	publisher_class    = "org.dependencytrack.notification.publisher.SlackPublisher"
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName,
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_rule" "test" {
	name               = %[1]q
	publisher_id       = dependencytrack_notification_publisher.test.id
	scope              = "PORTFOLIO"
	notification_level = "INFORMATIONAL"
}
`,
				ruleName,
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_rule" "test-other" {
	name               = "%[1]s-other"
	publisher_id       = dependencytrack_notification_publisher.test.id
	scope              = "PORTFOLIO"
	notification_level = "INFORMATIONAL"
}
`,
				ruleName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test" {
	name        = %[1]q
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
			fmt.Sprintf(`
resource "dependencytrack_project" "test-other" {
	name        = "%[1]s-other"
	classifier  = "APPLICATION"
}
`,
				projectName,
			),
		),
	)
}
