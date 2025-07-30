// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package notificationrule_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	dtrack "github.com/futurice/dependency-track-client-go"
	notificationpublishertestutils "github.com/futurice/terraform-provider-dependencytrack/internal/testutils/notificationpublisher"
	notificationruletestutils "github.com/futurice/terraform-provider-dependencytrack/internal/testutils/notificationrule"
	"github.com/google/uuid"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testPublisher, testOtherPublisher dtrack.NotificationPublisher

func init() {
	testPublisher = dtrack.NotificationPublisher{
		Name:             acctest.RandomWithPrefix("test-notification-publisher"),
		PublisherClass:   "org.dependencytrack.notification.publisher.SlackPublisher",
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
		PublisherConfig:      "",
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
					resource.TestCheckNoResourceAttr(ruleResourceName, "publisher_config"),
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
		PublisherConfig:      "",
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
		PublisherConfig:      "",
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
		PublisherConfig:      `{"a": "b"}`,
	}

	testUpdatedRule := testRule
	testUpdatedRule.Enabled = true
	testUpdatedRule.NotifyChildren = true
	testUpdatedRule.NotifyOn = []string{"NEW_VULNERABLE_DEPENDENCY", "PROJECT_CREATED"}
	testUpdatedRule.LogSuccessfulPublish = false
	testUpdatedRule.PublisherConfig = `{"a": "c"}`

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

func testAccNotificationRuleConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, providerName, ruleName, scope, notificationLevel string) string {
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
	publisher_class    = "org.dependencytrack.notification.publisher.SlackPublisher"
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName,
			),
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test-other" {
	name               = "%[1]s-other"
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
	publisher_class    = "org.dependencytrack.notification.publisher.SlackPublisher"
	template_mime_type = "application/json"
	template           = "{}"
}
`,
				providerName,
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
	publisherWithID := publisherTemplate
	publisherWithID.UUID = uuid.MustParse(*publisherID)
	ruleWithPublisher := ruleTemplate
	ruleWithPublisher.Publisher = publisherWithID
	return ruleWithPublisher
}
