package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var dataSourceTransformID = os.Getenv("TF_VAR_data_source_transform_id")

func TestAccDataSourceTransform(t *testing.T) {
	if dataSourceTransformID == "" {
		t.Skip("Skipping TestAccDataSourceTransform: TF_VAR_data_source_transform_id not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
variable "data_source_transform_id" {
	type = string
}

data "streamkap_transform" "test" {
	id = var.data_source_transform_id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.streamkap_transform.test", "id", dataSourceTransformID),
					resource.TestCheckResourceAttrSet("data.streamkap_transform.test", "name"),
				),
			},
		},
	})
}
