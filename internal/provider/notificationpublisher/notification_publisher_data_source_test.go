package notificationpublisher_test

import (
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	notificationpublishertestutils "github.com/futurice/terraform-provider-dependencytrack/internal/testutils/notificationpublisher"
	"strconv"
	"testing"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNotificationPublisherDataSource_basic(t *testing.T) {
	publisherResourceName := notificationpublishertestutils.CreateNotificationPublisherResourceName("test")
	publisherDataSourceName := notificationpublishertestutils.CreateNotificationPublisherDataSourceName("test")

	testPublisher := dtrack.NotificationPublisher{
		Name:             acctest.RandomWithPrefix("test-notification-publisher"),
		PublisherClass:   "org.dependencytrack.notification.publisher.SlackPublisher",
		TemplateMimeType: "application/json",
		Template:         `{}`,
		DefaultPublisher: false,
		Description:      "Some description",
	}

	var publisherID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPublisherDataSourceConfigBasic(testDependencyTrack, testPublisher.Name, testPublisher.PublisherClass, testPublisher.TemplateMimeType, testPublisher.Template, testPublisher.Description),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.TestAccCheckGetResourceID(publisherResourceName, &publisherID),
					resource.TestCheckResourceAttrPtr(publisherDataSourceName, "id", &publisherID),
					resource.TestCheckResourceAttr(publisherDataSourceName, "name", testPublisher.Name),
					resource.TestCheckResourceAttr(publisherDataSourceName, "publisher_class", testPublisher.PublisherClass),
					resource.TestCheckResourceAttr(publisherDataSourceName, "template_mime_type", testPublisher.TemplateMimeType),
					resource.TestCheckResourceAttr(publisherDataSourceName, "template", testPublisher.Template),
					resource.TestCheckResourceAttr(publisherDataSourceName, "default_publisher", strconv.FormatBool(testPublisher.DefaultPublisher)),
					resource.TestCheckResourceAttr(publisherDataSourceName, "description", testPublisher.Description),
				),
			},
		},
	})
}

func testAccNotificationPublisherDataSourceConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, publisherName, publisherClass, templateMimeType, template, description string) string {
	return testDependencyTrack.AddProviderConfiguration(
		testutils.ComposeConfigs(
			fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
	name               = %[1]q
	publisher_class    = %[2]q
	template_mime_type = %[3]q
	template           = %[4]q
	description        = %[5]q
}
`,
				publisherName, publisherClass, templateMimeType, template, description,
			),
			`
data "dependencytrack_notification_publisher" "test" {
	name = dependencytrack_notification_publisher.test.name
}
`,
		),
	)
}
