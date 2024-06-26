package testutils

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"time"
)

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
// to be used in subsequent tests.
func TestAccCheckGetResourceID(resourceName string, uuidOutput *uuid.UUID) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		id, err := GetResourceID(state, resourceName)
		if err != nil {
			return err
		}

		*uuidOutput = id

		return nil
	}
}

// TestAccCheckDelay does not check anything, but can be used to introduce a delay into the test for
// debugging the created resources before Terraform goes on to delete them.
func TestAccCheckDelay(d time.Duration) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		time.Sleep(d)
		return nil
	}
}
