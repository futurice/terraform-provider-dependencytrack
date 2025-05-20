// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package configproperty_test

import (
	"context"
	"fmt"
	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"os"
	"testing"

	"github.com/futurice/terraform-provider-dependencytrack/internal/testutils"
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

func TestAccConfigPropertyResource_basic(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	groupName := "email"
	name := "smtp.from.address"
	value := "test@example.com"
	updatedValue := "test2@example.com"
	originalValue := "original@example.com"

	configPropertyResourceName := createConfigPropertyResourceName("test")

	// fix the "original" value before the test
	err := setConfigProperty(ctx, testDependencyTrack, groupName, name, originalValue)
	if err != nil {
		t.Fatalf("Failed to set original value before the test: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigPropertyConfigBasic(testDependencyTrack, groupName, name, value),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, groupName, name, value),
					resource.TestCheckResourceAttrSet(configPropertyResourceName, "id"),
					resource.TestCheckResourceAttr(configPropertyResourceName, "group_name", groupName),
					resource.TestCheckResourceAttr(configPropertyResourceName, "name", name),
					resource.TestCheckResourceAttr(configPropertyResourceName, "value", value),
					resource.TestCheckNoResourceAttr(configPropertyResourceName, "destroy_value"),
					resource.TestCheckResourceAttr(configPropertyResourceName, "original_value", originalValue),
				),
			},
			{
				Config: testAccConfigPropertyConfigBasic(testDependencyTrack, groupName, name, updatedValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, groupName, name, updatedValue),
					resource.TestCheckResourceAttrSet(configPropertyResourceName, "id"),
					resource.TestCheckResourceAttr(configPropertyResourceName, "group_name", groupName),
					resource.TestCheckResourceAttr(configPropertyResourceName, "name", name),
					resource.TestCheckResourceAttr(configPropertyResourceName, "value", updatedValue),
					resource.TestCheckNoResourceAttr(configPropertyResourceName, "destroy_value"),
					resource.TestCheckResourceAttr(configPropertyResourceName, "original_value", originalValue),
				),
			},
		},
		CheckDestroy: testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, groupName, name, originalValue),
	})
}

func TestAccConfigPropertyResource_destroyValue(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	groupName := "email"
	name := "smtp.from.address"
	value := "test@example.com"
	destroyValue := "destroy@example.com"
	updatedDestroyValue := "destroy2@example.com"
	originalValue := "original@example.com"

	configPropertyResourceName := createConfigPropertyResourceName("test")

	// fix the "original" value before the test
	err := setConfigProperty(ctx, testDependencyTrack, groupName, name, originalValue)
	if err != nil {
		t.Fatalf("Failed to set original value before the test: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigPropertyConfigDestroyValue(testDependencyTrack, groupName, name, value, destroyValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, groupName, name, value),
					resource.TestCheckResourceAttrSet(configPropertyResourceName, "id"),
					resource.TestCheckResourceAttr(configPropertyResourceName, "group_name", groupName),
					resource.TestCheckResourceAttr(configPropertyResourceName, "name", name),
					resource.TestCheckResourceAttr(configPropertyResourceName, "value", value),
					resource.TestCheckResourceAttr(configPropertyResourceName, "destroy_value", destroyValue),
					resource.TestCheckResourceAttr(configPropertyResourceName, "original_value", originalValue),
				),
			},
			{
				Config: testAccConfigPropertyConfigDestroyValue(testDependencyTrack, groupName, name, value, updatedDestroyValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, groupName, name, value),
					resource.TestCheckResourceAttrSet(configPropertyResourceName, "id"),
					resource.TestCheckResourceAttr(configPropertyResourceName, "group_name", groupName),
					resource.TestCheckResourceAttr(configPropertyResourceName, "name", name),
					resource.TestCheckResourceAttr(configPropertyResourceName, "value", value),
					resource.TestCheckResourceAttr(configPropertyResourceName, "destroy_value", updatedDestroyValue),
					resource.TestCheckResourceAttr(configPropertyResourceName, "original_value", originalValue),
				),
			},
		},
		CheckDestroy: testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, groupName, name, updatedDestroyValue),
	})
}

func TestAccConfigPropertyResource_changeProperty(t *testing.T) {
	ctx := testutils.CreateTestContext(t)

	groupName := "email"
	name := "smtp.from.address"
	value := "test@example.com"
	originalValue := "original@example.com"

	otherGroupName := "integrations"
	otherName := "defectdojo.apiKey"
	otherValue := "someKey"
	otherOriginalValue := "originalKey"

	configPropertyResourceName := createConfigPropertyResourceName("test")

	// fix the "original" values before the test
	err := setConfigProperty(ctx, testDependencyTrack, groupName, name, originalValue)
	if err != nil {
		t.Fatalf("Failed to set original value before the test: %v", err)
	}

	err = setConfigProperty(ctx, testDependencyTrack, otherGroupName, otherName, otherOriginalValue)
	if err != nil {
		t.Fatalf("Failed to set original value before the test: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigPropertyConfigBasic(testDependencyTrack, groupName, name, value),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, groupName, name, value),
					testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, otherGroupName, otherName, otherOriginalValue),
				),
			},
			{
				Config: testAccConfigPropertyConfigBasic(testDependencyTrack, otherGroupName, otherName, otherValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, groupName, name, originalValue),
					testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, otherGroupName, otherName, otherValue),
					resource.TestCheckResourceAttrSet(configPropertyResourceName, "id"),
					resource.TestCheckResourceAttr(configPropertyResourceName, "group_name", otherGroupName),
					resource.TestCheckResourceAttr(configPropertyResourceName, "name", otherName),
					resource.TestCheckResourceAttr(configPropertyResourceName, "value", otherValue),
					resource.TestCheckNoResourceAttr(configPropertyResourceName, "destroy_value"),
					resource.TestCheckResourceAttr(configPropertyResourceName, "original_value", otherOriginalValue),
				),
			},
		},
		CheckDestroy: testAccCheckConfigPropertyHasExpectedValue(ctx, testDependencyTrack, otherGroupName, otherName, otherOriginalValue),
	})
}

func testAccConfigPropertyConfigBasic(testDependencyTrack *testutils.TestDependencyTrack, groupName, name, value string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_config_property" "test" {
	group_name         = %[1]q
	name               = %[2]q
	value              = %[3]q
}
`,
			groupName, name, value,
		),
	)
}

func testAccConfigPropertyConfigDestroyValue(testDependencyTrack *testutils.TestDependencyTrack, groupName, name, value, destroyValue string) string {
	return testDependencyTrack.AddProviderConfiguration(
		fmt.Sprintf(`
resource "dependencytrack_config_property" "test" {
	group_name         = %[1]q
	name               = %[2]q
	value              = %[3]q
	destroy_value      = %[4]q
}
`,
			groupName, name, value, destroyValue,
		),
	)
}

func createConfigPropertyResourceName(localName string) string {
	return fmt.Sprintf("dependencytrack_config_property.%s", localName)
}

func testAccCheckConfigPropertyHasExpectedValue(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, groupName, name, expectedValue string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		configProperty, err := findConfigProperty(ctx, testDependencyTrack, groupName, name)
		if err != nil {
			return err
		}
		if configProperty == nil {
			return fmt.Errorf("failed to find config property [%s]/[%s] from Dependency-Track", groupName, name)
		}
		if configProperty.Value == "" {
			return fmt.Errorf("config property [%s]/[%s] has no value", groupName, name)
		}

		configPropertyValue := configProperty.Value
		if configPropertyValue != expectedValue {
			return fmt.Errorf("config property [%s]/[%s] has value [%s] instead of the expected [%s]", groupName, name, configPropertyValue, expectedValue)
		}

		return nil
	}
}

func findConfigProperty(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, groupName, name string) (*dtrack.ConfigProperty, error) {
	configProperties, err := testDependencyTrack.Client.Config.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get config properties from Dependency-Track: %w", err)
	}

	for _, configProperty := range configProperties {
		if configProperty.GroupName == groupName && configProperty.Name == name {
			return &configProperty, nil
		}
	}

	return nil, nil
}

func setConfigProperty(ctx context.Context, testDependencyTrack *testutils.TestDependencyTrack, groupName, name, value string) error {
	setConfigPropertyRequest := dtrack.ConfigProperty{
		GroupName: groupName,
		Name:      name,
		Value:     value,
	}

	_, err := testDependencyTrack.Client.Config.Update(ctx, setConfigPropertyRequest)
	if err != nil {
		return fmt.Errorf("failed to set config property in Dependency-Track: %w", err)
	}

	return nil
}
