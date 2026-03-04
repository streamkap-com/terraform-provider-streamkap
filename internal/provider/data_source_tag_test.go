package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceTag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "streamkap_tag" "test" {
	name        = "tf-acc-test-datasource-tag"
	description = "Test tag for data source"
	type        = ["sources"]
}

data "streamkap_tag" "test" {
	id = streamkap_tag.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.streamkap_tag.test", "id", "streamkap_tag.test", "id"),
					resource.TestCheckResourceAttr("data.streamkap_tag.test", "name", "tf-acc-test-datasource-tag"),
					resource.TestCheckResourceAttr("data.streamkap_tag.test", "description", "Test tag for data source"),
				),
			},
		},
	})
}
