// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package testutils

import (
	"fmt"
	"github.com/futurice/terraform-provider-dependencytrack/internal/provider"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"dependencytrack": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

// TestAccCheckDelay does not check anything, but can be used to introduce a delay into the test for
// debugging the created resources before Terraform goes on to delete them.
func TestAccCheckDelay(d time.Duration) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		time.Sleep(d)
		return nil
	}
}

func GetResourceID(state *terraform.State, resourceName string) (uuid.UUID, error) {
	res, ok := state.RootModule().Resources[resourceName]
	if !ok {
		return uuid.UUID{}, fmt.Errorf("resource not found: %s", resourceName)
	}

	idString := res.Primary.ID

	id, err := uuid.Parse(idString)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("the resource ID must be a valid UUID, got %s: %w", idString, err)
	}

	return id, nil
}

// TestAccCheckGetResourceID does not check anything, but can be used to retrieve the ID of a created resource
// to be used in subsequent tests. The output is a string as this is what will be useful to further assertions.
func TestAccCheckGetResourceID(resourceName string, uuidOutput *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		id, err := GetResourceID(state, resourceName)
		if err != nil {
			return err
		}

		*uuidOutput = id.String()

		return nil
	}
}
