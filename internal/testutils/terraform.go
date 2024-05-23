package testutils

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
