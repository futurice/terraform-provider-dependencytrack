// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package notificationruletestutils

import (
	"context"
	"fmt"
	"slices"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCheckNotificationRuleExistsAndHasExpectedLazyData(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedRuleCreator func() dtrack.NotificationRule) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		expectedRule := expectedRuleCreator()

		rule, err := FindNotificationRuleByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if rule == nil {
			return fmt.Errorf("notification rule for resource %s does not exist in Dependency-Track", resourceName)
		}

		// Publisher.Template not returned from this endpoint for some reason
		diff := cmp.Diff(rule, &expectedRule, cmpopts.IgnoreFields(dtrack.NotificationRule{}, "UUID", "Projects", "Publisher.Template"))
		if diff != "" {
			return fmt.Errorf("notification rule for resource %s is different than expected: %s", resourceName, diff)
		}

		return nil
	}
}

func TestAccCheckNotificationRuleDoesNotExists(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rule, err := FindNotificationRuleByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if rule != nil {
			return fmt.Errorf("notification rule for resource %s exists in Dependency-Track, even though it shouldn't: %v", resourceName, rule)
		}

		return nil
	}
}

func FindNotificationRuleByResourceName(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, state *terraform.State, resourceName string) (*dtrack.NotificationRule, error) {
	ruleID, err := testutils.GetResourceID(state, resourceName)
	if err != nil {
		return nil, err
	}

	rule, err := FindNotificationRule(ctx, testDependencyTrack, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification rule for resource %s: %w", resourceName, err)
	}

	return rule, nil
}

func FindNotificationRule(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, ruleID uuid.UUID) (*dtrack.NotificationRule, error) {
	rules, err := testDependencyTrack.Client.Notification.GetAllRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification rule from Dependency-Track: %w", err)
	}

	for _, rule := range rules {
		if rule.UUID == ruleID {
			return &rule, nil
		}
	}

	return nil, nil
}

func TestAccCheckNotificationRuleHasExpectedProjects(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, resourceName string, expectedProjectIDs []*string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rule, err := FindNotificationRuleByResourceName(ctx, testDependencyTrack, state, resourceName)
		if err != nil {
			return err
		}
		if rule == nil {
			return fmt.Errorf("notification rule for resource %s does not exist in Dependency-Track", resourceName)
		}

		if len(rule.Projects) != len(expectedProjectIDs) {
			return fmt.Errorf("notification rule for resource %s has %d projects instead of the expected %d", resourceName, len(rule.Projects), len(expectedProjectIDs))
		}

		actualProjectIDs := make([]string, len(rule.Projects))
		for i, project := range rule.Projects {
			actualProjectIDs[i] = project.UUID.String()
		}

		for _, expectedProjectID := range expectedProjectIDs {
			if !slices.Contains(actualProjectIDs, *expectedProjectID) {
				return fmt.Errorf("notification rule for resource %s is missing expected project %s, got [%v]", resourceName, *expectedProjectID, actualProjectIDs)
			}
		}

		return nil
	}
}

func CreateNotificationRuleResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_notification_rule.%s", localName)
}

func CreateNotificationRuleProjectResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_notification_rule_project.%s", localName)
}
