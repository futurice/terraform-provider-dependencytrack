// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package notificationrule_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	dtrack "github.com/futurice/dependency-track-client-go"
	notificationpublishertestutils "github.com/futurice/terraform-provider-dependencytrack/internal/testutils/notificationpublisher"
	notificationruletestutils "github.com/futurice/terraform-provider-dependencytrack/internal/testutils/notificationrule"
	"github.com/google/uuid"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testPublisher, testOtherPublisher dtrack.NotificationPublisher

func init() {
	// PublisherClass or ExtensionName is filled in by applyTestPublisherToRule, once the server version is known.
	testPublisher = dtrack.NotificationPublisher{
		Name:             acctest.RandomWithPrefix("test-notification-publisher"),
		TemplateMimeType: "application/json",
		Template:         `{}`,
	}

	testOtherPublisher = testPublisher
	testOtherPublisher.Name = testPublisher.Name + "-other"
}

var testDependencyTrack *testutils.TestDependencyTrack

func TestMain(m *testing.M) {
	if os.Getenv(resource.EnvTfAcc) != "" {
		var cleanup func()
		testDependencyTrack, cleanup = testutils.InitTestDependencyTrack()
		defer cleanup()
	}

	m.Run()
}

func TestAccNotificationRuleResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	ruleResourceName := notificationruletestutils.CreateNotificationRuleResourceName("test")
	publisherResourceName := notificationpublishertestutils.CreateNotificationPublisherResourceName("test")

	testRule := dtrack.NotificationRule{
		Name:              acctest.RandomWithPrefix("test-notification-rule"),
		NotificationLevel: "INFORMATIONAL",
		// Publisher is filled in dynamically below
		Scope:                "PORTFOLIO",
		Enabled:              true,
		NotifyChildren:       true,
		NotifyOn:             []string{},
		LogSuccessfulPublish: false,
		PublisherConfig:      defaultPublisherConfig(),
	}

	testUpdatedRule := testRule
	testRule.Name = acctest.RandomWithPrefix("other-test-notification-rule")
	testRule.NotificationLevel = "WARNING"

	var publisherID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRuleConfigBasic(testDependencyTrack, testPublisher.Name, testRule.Name, testRule.Scope, testRule.NotificationLevel),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(publisherResourceName, &publisherID),
					notificationruletestutils.TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx, testDependencyTrack, ruleResourceName, func() dtrack.NotificationRule {
						return applyTestPublisherToRule(testRule, testPublisher, &publisherID)
					}),
					resource.TestCheckResourceAttrSet(ruleResourceName, "id"),
					resource.TestCheckResourceAttr(ruleResourceName, "name", testRule.Name),
					resource.TestCheckResourceAttr(ruleResourceName, "notification_level", testRule.NotificationLevel),
					resource.TestCheckResourceAttrPtr(ruleResourceName, "publisher_id", &publisherID),
					resource.TestCheckResourceAttr(ruleResourceName, "scope", testRule.Scope),
					resource.TestCheckResourceAttr(ruleResourceName, "enabled", strconv.FormatBool(testRule.Enabled)),
					resource.TestCheckResourceAttr(ruleResourceName, "notify_children", strconv.FormatBool(testRule.NotifyChildren)),
					resource.TestCheckResourceAttr(ruleResourceName, "notify_on.#", "0"),
					resource.TestCheckResourceAttr(ruleResourceName, "log_successful_publish", strconv.FormatBool(testRule.LogSuccessfulPublish)),
					checkDefaultPublisherConfig(ruleResourceName),
				),
			},
			{
				ResourceName:      ruleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNotificationRuleConfigBasic(testDependencyTrack, testPublisher.Name, testUpdatedRule.Name, testUpdatedRule.Scope, testUpdatedRule.NotificationLevel),
				Check: resource.ComposeAggregateTestCheckFunc(
					notificationruletestutils.TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx, testDependencyTrack, ruleResourceName, func() dtrack.NotificationRule {
						return applyTestPublisherToRule(testUpdatedRule, testPublisher, &publisherID)
					}),
					resource.TestCheckResourceAttr(ruleResourceName, "name", testUpdatedRule.Name),
					resource.TestCheckResourceAttr(ruleResourceName, "notification_level", testUpdatedRule.NotificationLevel),
				),
			},
		},
		CheckDestroy: notificationruletestutils.TestAccCheckNotificationRuleDoesNotExists(ctx, testDependencyTrack, ruleResourceName),
	})
}

func TestAccNotificationRuleResource_publisherID(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	ruleResourceName := notificationruletestutils.CreateNotificationRuleResourceName("test")
	publisherResourceName := notificationpublishertestutils.CreateNotificationPublisherResourceName("test")
	otherPublisherResourceName := notificationpublishertestutils.CreateNotificationPublisherResourceName("test-other")

	testRule := dtrack.NotificationRule{
		Name:              acctest.RandomWithPrefix("test-notification-rule"),
		NotificationLevel: "INFORMATIONAL",
		// Publisher is filled in dynamically below
		Scope:                "SYSTEM",
		Enabled:              true,
		NotifyChildren:       true,
		NotifyOn:             []string{},
		LogSuccessfulPublish: false,
		PublisherConfig:      defaultPublisherConfig(),
	}

	var publisherID, otherPublisherID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRuleConfigBasic(testDependencyTrack, testPublisher.Name, testRule.Name, testRule.Scope, testRule.NotificationLevel),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(publisherResourceName, &publisherID),
					notificationruletestutils.TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx, testDependencyTrack, ruleResourceName, func() dtrack.NotificationRule {
						return applyTestPublisherToRule(testRule, testPublisher, &publisherID)
					}),
					resource.TestCheckResourceAttrPtr(ruleResourceName, "publisher_id", &publisherID),
				),
			},
			{
				Config: testAccNotificationRuleConfigOtherPublisher(testDependencyTrack, testPublisher.Name, testRule.Name, testRule.Scope, testRule.NotificationLevel),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(otherPublisherResourceName, &otherPublisherID),
					notificationruletestutils.TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx, testDependencyTrack, ruleResourceName, func() dtrack.NotificationRule {
						return applyTestPublisherToRule(testRule, testOtherPublisher, &otherPublisherID)
					}),
					resource.TestCheckResourceAttrPtr(ruleResourceName, "publisher_id", &otherPublisherID),
				),
			},
		},
	})
}

func TestAccNotificationRuleResource_scope(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	ruleResourceName := notificationruletestutils.CreateNotificationRuleResourceName("test")
	publisherResourceName := notificationpublishertestutils.CreateNotificationPublisherResourceName("test")

	testRule := dtrack.NotificationRule{
		Name:              acctest.RandomWithPrefix("test-notification-rule"),
		NotificationLevel: "INFORMATIONAL",
		// Publisher is filled in dynamically below
		Scope:                "SYSTEM",
		Enabled:              true,
		NotifyChildren:       true,
		NotifyOn:             []string{},
		LogSuccessfulPublish: false,
		PublisherConfig:      defaultPublisherConfig(),
	}

	testUpdatedRule := testRule
	testUpdatedRule.Scope = "PORTFOLIO"

	var publisherID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRuleConfigBasic(testDependencyTrack, testPublisher.Name, testRule.Name, testRule.Scope, testRule.NotificationLevel),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(publisherResourceName, &publisherID),
					notificationruletestutils.TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx, testDependencyTrack, ruleResourceName, func() dtrack.NotificationRule {
						return applyTestPublisherToRule(testRule, testPublisher, &publisherID)
					}),
					resource.TestCheckResourceAttr(ruleResourceName, "scope", testRule.Scope),
				),
			},
			{
				Config: testAccNotificationRuleConfigBasic(testDependencyTrack, testPublisher.Name, testUpdatedRule.Name, testUpdatedRule.Scope, testUpdatedRule.NotificationLevel),
				Check: resource.ComposeAggregateTestCheckFunc(
					notificationruletestutils.TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx, testDependencyTrack, ruleResourceName, func() dtrack.NotificationRule {
						return applyTestPublisherToRule(testUpdatedRule, testPublisher, &publisherID)
					}),
					resource.TestCheckResourceAttr(ruleResourceName, "scope", testUpdatedRule.Scope),
				),
			},
		},
	})
}

func TestAccNotificationRuleResource_settings(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	ruleResourceName := notificationruletestutils.CreateNotificationRuleResourceName("test")
	publisherResourceName := notificationpublishertestutils.CreateNotificationPublisherResourceName("test")

	testRule := dtrack.NotificationRule{
		Name:              acctest.RandomWithPrefix("test-notification-rule"),
		NotificationLevel: "INFORMATIONAL",
		// Publisher is filled in dynamically below
		Scope:                "PORTFOLIO",
		Enabled:              false,
		NotifyChildren:       false,
		NotifyOn:             []string{"USER_DELETED"},
		LogSuccessfulPublish: true,
		PublisherConfig:      slackPublisherConfig("b"),
	}

	testUpdatedRule := testRule
	testUpdatedRule.Enabled = true
	testUpdatedRule.NotifyChildren = true
	testUpdatedRule.NotifyOn = []string{"NEW_VULNERABLE_DEPENDENCY", "PROJECT_CREATED"}
	testUpdatedRule.LogSuccessfulPublish = false
	testUpdatedRule.PublisherConfig = slackPublisherConfig("c")

	var publisherID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRuleConfigWithSettings(testDependencyTrack, testPublisher.Name, testRule),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(publisherResourceName, &publisherID),
					notificationruletestutils.TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx, testDependencyTrack, ruleResourceName, func() dtrack.NotificationRule {
						return applyTestPublisherToRule(testRule, testPublisher, &publisherID)
					}),
					resource.TestCheckResourceAttr(ruleResourceName, "enabled", strconv.FormatBool(testRule.Enabled)),
					resource.TestCheckResourceAttr(ruleResourceName, "notify_children", strconv.FormatBool(testRule.NotifyChildren)),
					resource.TestCheckResourceAttr(ruleResourceName, "notify_on.#", "1"),
					resource.TestCheckTypeSetElemAttr(ruleResourceName, "notify_on.*", testRule.NotifyOn[0]),
					resource.TestCheckResourceAttr(ruleResourceName, "log_successful_publish", strconv.FormatBool(testRule.LogSuccessfulPublish)),
					resource.TestCheckResourceAttr(ruleResourceName, "publisher_config", testRule.PublisherConfig),
				),
			},
			{
				Config: testAccNotificationRuleConfigWithSettings(testDependencyTrack, testPublisher.Name, testUpdatedRule),
				Check: resource.ComposeAggregateTestCheckFunc(
					notificationruletestutils.TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx, testDependencyTrack, ruleResourceName, func() dtrack.NotificationRule {
						return applyTestPublisherToRule(testUpdatedRule, testPublisher, &publisherID)
					}),
					resource.TestCheckResourceAttr(ruleResourceName, "enabled", strconv.FormatBool(testUpdatedRule.Enabled)),
					resource.TestCheckResourceAttr(ruleResourceName, "notify_children", strconv.FormatBool(testUpdatedRule.NotifyChildren)),
					resource.TestCheckResourceAttr(ruleResourceName, "notify_on.#", "2"),
					resource.TestCheckTypeSetElemAttr(ruleResourceName, "notify_on.*", testUpdatedRule.NotifyOn[0]),
					resource.TestCheckTypeSetElemAttr(ruleResourceName, "notify_on.*", testUpdatedRule.NotifyOn[1]),
					resource.TestCheckResourceAttr(ruleResourceName, "log_successful_publish", strconv.FormatBool(testUpdatedRule.LogSuccessfulPublish)),
					resource.TestCheckResourceAttr(ruleResourceName, "publisher_config", testUpdatedRule.PublisherConfig),
				),
			},
		},
	})
}

func TestAccNotificationRuleResource_keepsTagsAndFilterExpression(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	ruleResourceName := notificationruletestutils.CreateNotificationRuleResourceName("test")
	ruleName := acctest.RandomWithPrefix("test-notification-rule")
	filterExpression := `subject.project.name == "x"`
	var ruleID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationRuleConfigBasic(testDependencyTrack, testPublisher.Name, ruleName, "PORTFOLIO", "INFORMATIONAL"),
				Check:  testutils.TestAccCheckGetResourceID(ruleResourceName, &ruleID),
			},
			{
				// Set what the UI would, then change the rule through Terraform.
				PreConfig: func() { setTagsAndFilterExpression(ctx, t, ruleID, filterExpression) },
				Config:    testAccNotificationRuleConfigBasic(testDependencyTrack, testPublisher.Name, ruleName, "PORTFOLIO", "WARNING"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ruleResourceName, "notification_level", "WARNING"),
					checkTagsAndFilterExpression(ctx, ruleResourceName, filterExpression),
				),
			},
		},
	})
}

// setTagsAndFilterExpression sets what the UI would: a tag, and on v5 a filter expression.
func setTagsAndFilterExpression(ctx context.Context, t *testing.T, ruleID, filterExpression string) {
	t.Helper()

	rule, err := notificationruletestutils.FindNotificationRule(ctx, testDependencyTrack, uuid.MustParse(ruleID))
	if err != nil {
		t.Fatalf("unable to find rule %s: %v", ruleID, err)
	}
	if rule == nil {
		t.Fatalf("rule %s not found", ruleID)
	}

	rule.Tags = []dtrack.Tag{{Name: "keep-me"}}
	if testDependencyTrack.IsV5() {
		rule.FilterExpression = filterExpression
	}
	if _, err := testDependencyTrack.Client.Notification.UpdateRule(ctx, *rule); err != nil {
		t.Fatalf("unable to set tags: %v", err)
	}
}

func checkTagsAndFilterExpression(ctx context.Context, resourceName, filterExpression string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rule, err := notificationruletestutils.FindNotificationRuleByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if rule == nil {
			return fmt.Errorf("rule for %s not found", resourceName)
		}
		if len(rule.Tags) != 1 || rule.Tags[0].Name != "keep-me" {
			return fmt.Errorf("expected the tag to survive the update, got %v", rule.Tags)
		}
		if testDependencyTrack.IsV5() && rule.FilterExpression != filterExpression {
			return fmt.Errorf("expected the filter expression to survive the update, got %q", rule.FilterExpression)
		}
		return nil
	}
}

func TestAccNotificationRuleResource_failedUpdateTaintsRule(t *testing.T) {
	if !testDependencyTrack.IsV5() {
		t.Skip("only v5 validates publisher_config, which is what makes the follow-up update fail")
	}
	ctx := testutils.CreateTestContext(t)

	rule := dtrack.NotificationRule{
		Name:              acctest.RandomWithPrefix("test-notification-rule"),
		NotificationLevel: "INFORMATIONAL",
		Scope:             "PORTFOLIO",
		Enabled:           true,
		NotifyOn:          []string{"NEW_VULNERABILITY"},
		PublisherConfig:   `{}`,
	}
	validRule := rule
	validRule.PublisherConfig = slackPublisherConfig("tainted")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccNotificationRuleConfigWithSettings(testDependencyTrack, testPublisher.Name, rule),
				ExpectError: regexp.MustCompile(`follow-up\s+update\s+failed`),
			},
			{
				// The tainted rule is replaced, so exactly one rule has the name.
				Config: testAccNotificationRuleConfigWithSettings(testDependencyTrack, testPublisher.Name, validRule),
				Check: func(*terraform.State) error {
					rules, err := dtrack.FetchAll(func(po dtrack.PageOptions) (dtrack.Page[dtrack.NotificationRule], error) {
						return testDependencyTrack.Client.Notification.GetAllRules(ctx, po)
					})
					if err != nil {
						return err
					}
					count := 0
					for _, r := range rules {
						if r.Name == rule.Name {
							count++
						}
					}
					if count != 1 {
						return fmt.Errorf("expected 1 rule named %s, found %d", rule.Name, count)
					}
					return nil
				},
			},
		},
	})
}

func testAccNotificationRuleConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, providerName, ruleName, scope, notificationLevel string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	%[2]s
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName, testDependencyTrack.PublisherKindAttribute("slack"),
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_rule" "test" {
	name               = %[1]q
	publisher_id       = dependencytrack_notification_publisher.test.id
	scope              = %[2]q
	notification_level = %[3]q
}
`,
				ruleName, scope, notificationLevel,
			),
		),
	)
}

func testAccNotificationRuleConfigOtherPublisher(testDependencyTrack *testutils.TestDependencyTrack, providerName, ruleName, scope, notificationLevel string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	%[2]s
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName, testDependencyTrack.PublisherKindAttribute("slack"),
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test-other" {
	name               = "%[1]s-other"
	%[2]s
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName, testDependencyTrack.PublisherKindAttribute("slack"),
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_rule" "test" {
	name               = %[1]q
	publisher_id       = dependencytrack_notification_publisher.test-other.id
	scope              = %[2]q
	notification_level = %[3]q
}
`,
				ruleName, scope, notificationLevel,
			),
		),
	)
}

func testAccNotificationRuleConfigWithSettings(testDependencyTrack *testutils.TestDependencyTrack, providerName string, rule dtrack.NotificationRule) string {
	notifyOnQuoted := make([]string, len(rule.NotifyOn))
	for i, notifyOn := range rule.NotifyOn {
		notifyOnQuoted[i] = fmt.Sprintf("%q", notifyOn)
	}
	notifyOnString := fmt.Sprintf("[%s]", strings.Join(notifyOnQuoted, ", "))

	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	%[2]s
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName, testDependencyTrack.PublisherKindAttribute("slack"),
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_rule" "test" {
	name                   = %[1]q
	publisher_id           = dependencytrack_notification_publisher.test.id
	scope                  = %[2]q
	notification_level     = %[3]q
	enabled                = %[4]t
	notify_children        = %[5]t
	notify_on              = %[6]s
	log_successful_publish = %[7]t
	publisher_config       = %[8]q
}
`,
				rule.Name,
				rule.Scope,
				rule.NotificationLevel,
				rule.Enabled,
				rule.NotifyChildren,
				notifyOnString,
				rule.LogSuccessfulPublish,
				rule.PublisherConfig,
			),
		),
	)
}

func applyTestPublisherToRule(ruleTemplate dtrack.NotificationRule, publisherTemplate dtrack.NotificationPublisher, publisherID *string) dtrack.NotificationRule {
	publisherWithID := testDependencyTrack.WithPublisherKind(publisherTemplate, "slack")
	publisherWithID.UUID = uuid.MustParse(*publisherID)
	ruleWithPublisher := ruleTemplate
	ruleWithPublisher.Publisher = publisherWithID
	return ruleWithPublisher
}

// defaultPublisherConfig is the Slack config a rule gets when none is set, which v5 fills in on create.
func defaultPublisherConfig() string {
	if testDependencyTrack.IsV5() {
		return `{"destinationUrl": "https://slack.example.com"}`
	}
	return ""
}

func checkDefaultPublisherConfig(resourceName string) resource.TestCheckFunc {
	if testDependencyTrack.IsV5() {
		// The update response and the rule list format the same JSON differently.
		return resource.TestCheckResourceAttrWith(resourceName, "publisher_config", func(value string) error {
			if !utils.JSONEqual(value, defaultPublisherConfig()) {
				return fmt.Errorf("expected publisher_config equivalent to %s, got %s", defaultPublisherConfig(), value)
			}
			return nil
		})
	}
	return resource.TestCheckNoResourceAttr(resourceName, "publisher_config")
}

// slackPublisherConfig returns a config v5 accepts for the Slack publisher, which v4 stores as is.
func slackPublisherConfig(path string) string {
	return fmt.Sprintf(`{"destinationUrl": "https://example.com/%s"}`, path)
}
