// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package notificationpublisher_test

import (
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	notificationpublishertestutils "github.com/futurice/terraform-provider-dependencytrack/internal/testutils/notificationpublisher"
	"testing"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNotificationPublisherResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	publisherResourceName := notificationpublishertestutils.CreateNotificationPublisherResourceName("test")

	testPublisher := dtrack.NotificationPublisher{
		Name:             acctest.RandomWithPrefix("test-notification-publisher"),
		PublisherClass:   "org.dependencytrack.notification.publisher.SlackPublisher",
		TemplateMimeType: "application/json",
		Template:         `{}`,
		DefaultPublisher: false,
	}

	testUpdatedPublisher := dtrack.NotificationPublisher{
		Name:             acctest.RandomWithPrefix("other-test-notification-publisher"),
		PublisherClass:   "org.dependencytrack.notification.publisher.WebhookPublisher",
		TemplateMimeType: "application/json+something",
		Template:         `{"a": "b"}`,
		DefaultPublisher: false,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPublisherConfigBasic(testDependencyTrack, testPublisher.Name, testPublisher.PublisherClass, testPublisher.TemplateMimeType, testPublisher.Template),
				Check: resource.ComposeAggregateTestCheckFunc(
					notificationpublishertestutils.TestAccCheckNotificationPublisherExistsAndHasExpectedData(ctx, testDependencyTrack, publisherResourceName, testPublisher),
					resource.TestCheckResourceAttrSet(publisherResourceName, "id"),
					resource.TestCheckResourceAttr(publisherResourceName, "name", testPublisher.Name),
					resource.TestCheckResourceAttr(publisherResourceName, "publisher_class", testPublisher.PublisherClass),
					resource.TestCheckResourceAttr(publisherResourceName, "template_mime_type", testPublisher.TemplateMimeType),
					resource.TestCheckResourceAttr(publisherResourceName, "template", testPublisher.Template),
				),
			},
			{
				ResourceName:      publisherResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNotificationPublisherConfigBasic(testDependencyTrack, testUpdatedPublisher.Name, testUpdatedPublisher.PublisherClass, testUpdatedPublisher.TemplateMimeType, testUpdatedPublisher.Template),
				Check: resource.ComposeAggregateTestCheckFunc(
					notificationpublishertestutils.TestAccCheckNotificationPublisherExistsAndHasExpectedData(ctx, testDependencyTrack, publisherResourceName, testUpdatedPublisher),
					resource.TestCheckResourceAttr(publisherResourceName, "name", testUpdatedPublisher.Name),
					resource.TestCheckResourceAttr(publisherResourceName, "publisher_class", testUpdatedPublisher.PublisherClass),
					resource.TestCheckResourceAttr(publisherResourceName, "template_mime_type", testUpdatedPublisher.TemplateMimeType),
					resource.TestCheckResourceAttr(publisherResourceName, "template", testUpdatedPublisher.Template),
				),
			},
		},
		CheckDestroy: notificationpublishertestutils.TestAccCheckNotificationPublisherDoesNotExists(ctx, testDependencyTrack, publisherResourceName),
	})
}

func TestAccNotificationPublisherResource_description(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	publisherResourceName := notificationpublishertestutils.CreateNotificationPublisherResourceName("test")

	testPublisher := dtrack.NotificationPublisher{
		Name:             acctest.RandomWithPrefix("test-notification-publisher"),
		PublisherClass:   "org.dependencytrack.notification.publisher.SlackPublisher",
		TemplateMimeType: "application/json",
		Template:         `{}`,
		Description:      "Some description",
	}

	testUpdatedPublisher := dtrack.NotificationPublisher{
		Name:             acctest.RandomWithPrefix("test-notification-publisher"),
		PublisherClass:   "org.dependencytrack.notification.publisher.SlackPublisher",
		TemplateMimeType: "application/json",
		Template:         `{}`,
		Description:      "Some other description",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPublisherConfigDescription(testDependencyTrack, testPublisher.Name, testPublisher.Description),
				Check: resource.ComposeAggregateTestCheckFunc(
					notificationpublishertestutils.TestAccCheckNotificationPublisherExistsAndHasExpectedData(ctx, testDependencyTrack, publisherResourceName, testPublisher),
					resource.TestCheckResourceAttr(publisherResourceName, "description", testPublisher.Description),
				),
			},
			{
				Config: testAccNotificationPublisherConfigDescription(testDependencyTrack, testUpdatedPublisher.Name, testUpdatedPublisher.Description),
				Check: resource.ComposeAggregateTestCheckFunc(
					notificationpublishertestutils.TestAccCheckNotificationPublisherExistsAndHasExpectedData(ctx, testDependencyTrack, publisherResourceName, testUpdatedPublisher),
					resource.TestCheckResourceAttr(publisherResourceName, "description", testUpdatedPublisher.Description),
				),
			},
		},
	})
}

func testAccNotificationPublisherConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, publisherName, publisherClass, templateMimeType, template string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	publisher_class    = %[2]q
	template_mime_type = %[3]q
	template           = %[4]q
}
`,
			publisherName, publisherClass, templateMimeType, template,
		),
	)
}

func testAccNotificationPublisherConfigDescription(testDependencyTrack *testutils.TestDependencyTrack, publisherName, description string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	publisher_class    = "org.dependencytrack.notification.publisher.SlackPublisher"
	template_mime_type = "application/json"
	template           = "{}"
	description        = %[2]q
}
`,
			publisherName, description,
		),
	)
}
