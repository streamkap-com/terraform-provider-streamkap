package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceTag(t *testing.T) {
	// Use a per-run unique name. With a static name, a previous failed
	// teardown that left tags attached to sources would leak the tag and
	// every subsequent run's destroy step would fail with
	// `Tag is used by N sources entities and can't be deleted`. The
	// sweeper covers cleanup, but a unique name is the cheaper fix and
	// makes parallel runs safe too.
	tagName := fmt.Sprintf("tf-acc-test-datasource-tag-%s", uniqSuffix(t))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_tag" "test" {
	name        = %q
	description = "Test tag for data source"
	type        = ["sources"]
}

data "streamkap_tag" "test" {
	id = streamkap_tag.test.id
}
`, tagName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.streamkap_tag.test", "id", "streamkap_tag.test", "id"),
					resource.TestCheckResourceAttr("data.streamkap_tag.test", "name", tagName),
					resource.TestCheckResourceAttr("data.streamkap_tag.test", "description", "Test tag for data source"),
				),
			},
		},
	})
}
