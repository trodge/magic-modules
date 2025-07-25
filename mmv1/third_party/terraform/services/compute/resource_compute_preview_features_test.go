package compute_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-google/google/acctest"
)

func TestAccComputePreviewFeature_update(t *testing.T) {
	t.Parallel()

	// The specific feature name to test.
	featureName := "alpha-api-access"
	// The resource name in Terraform state.
	resourceName := "google_compute_preview_feature.acceptance"

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories(t),
		Steps: []resource.TestStep{
			// Step 1: Disable the "alpha-api-access" feature and verify its attributes.
			{
				Config: testAccComputePreviewFeature_disable(featureName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", featureName),
					resource.TestCheckResourceAttr(resourceName, "activation_status", "DISABLED"),
				),
			},
			// Step 2: Verify that the resource can be successfully imported.
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rollout_operation"},
			},
		},
	})
}

func testAccComputePreviewFeature_disable(name string) string {
	return fmt.Sprintf(`
resource "google_compute_preview_feature" "acceptance" {
  name              = "%s"
  activation_status = "DISABLED"
  
  rollout_operation {
    rollout_input {
      predefined_rollout_plan = "ROLLOUT_PLAN_FAST_ROLLOUT"
    }
  }
}
`, name)
}
