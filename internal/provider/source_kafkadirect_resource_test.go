package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSourceKafkaDirectResource(t *testing.T) {
	name := acctestName(t, "main")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_source_kafkadirect" "test" {
	name               = %q
  	topic_prefix       = "sample-topic_"
  	format             = "json"
  	schemas_enable     = true
  	topic_include_list = "sample-topic_topic1, sample-topic_topic2, sample-topic_topic3"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "topic_prefix", "sample-topic_"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "format", "json"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "schemas_enable", "true"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "topic_include_list", "sample-topic_topic1, sample-topic_topic2, sample-topic_topic3"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:            "streamkap_source_kafkadirect.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
