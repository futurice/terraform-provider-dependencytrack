// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package notificationpublishertestutils

import (
	"context"
	"fmt"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCheckNotificationPublisherExistsAndHasExpectedData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedPublisher dtrack.NotificationPublisher) resource.TestCheckFunc {
	return TestAccCheckNotificationPublisherExistsAndHasExpectedLazyData(ctx, testDependencyTrack, resourceName, func() dtrack.NotificationPublisher { return expectedPublisher })
}

func TestAccCheckNotificationPublisherExistsAndHasExpectedLazyData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedPublisherCreator func() dtrack.NotificationPublisher) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		expectedPublisher := expectedPublisherCreator()

		publisher, err := FindNotificationPublisherByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if publisher == nil {
			return fmt.Errorf("notification publisher for resource %s does not exist in Dependency-Track", resourceName)
		}

		diff := cmp.Diff(publisher, &expectedPublisher, cmpopts.IgnoreFields(dtrack.NotificationPublisher{}, "UUID"))
		if diff != "" {
			return fmt.Errorf("notification publisher for resource %s is different than expected: %s", resourceName, diff)
		}

		return nil
	}
}

func TestAccCheckNotificationPublisherDoesNotExists(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		publisher, err := FindNotificationPublisherByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if publisher != nil {
			return fmt.Errorf("notification publisher for resource %s exists in Dependency-Track, even though it shouldn't: %v", resourceName, publisher)
		}

		return nil
	}
}

func FindNotificationPublisherByResourceName(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, state *terraform.State, resourceName string) (*dtrack.NotificationPublisher, error) {
	publisherID, err := testutils.GetResourceID(state, resourceName)
	if err != nil {
		return nil, err
	}

	publisher, err := FindNotificationPublisher(ctx, testDependencyTrack, publisherID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification publisher for resource %s: %w", resourceName, err)
	}

	return publisher, nil
}

func FindNotificationPublisher(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, publisherID uuid.UUID) (*dtrack.NotificationPublisher, error) {
	publishers, err := testDependencyTrack.Client.Notification.GetAllPublishers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification publisher from Dependency-Track: %w", err)
	}

	for _, publisher := range publishers {
		if publisher.UUID == publisherID {
			return &publisher, nil
		}
	}

	return nil, nil
}

func CreateNotificationPublisherResourceName(localName string) string {
	return "dependencytrack_notification_publisher." + localName
}

func CreateNotificationPublisherDataSourceName(localName string) string {
	return "data.dependencytrack_notification_publisher." + localName
}
