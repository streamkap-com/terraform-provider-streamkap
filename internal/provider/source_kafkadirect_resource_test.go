package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSourceKafkaDirectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "streamkap_source_kafkadirect" "test" {
	name                             = "test-source-kafkadirect"
	topic_prefix                     = "test-prefix"
	topic_include_list_user_defined  = "topic1,topic2"
	format                           = "json"
	schemas_enable                   = false
}
`,

				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "name", "test-source-kafkadirect"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "topic_prefix", "test-prefix"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "topic_include_list_user_defined", "topic1,topic2"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "format", "json"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "schemas_enable", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_kafkadirect.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "streamkap_source_kafkadirect" "test" {
	name                             = "test-source-kafkadirect-updated"
	topic_prefix                     = "updated-prefix"
	topic_include_list_user_defined  = "topic1,topic2,topic3"
	format                           = "string"
	schemas_enable                   = true
}
`,

				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "name", "test-source-kafkadirect-updated"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "topic_prefix", "updated-prefix"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "topic_include_list_user_defined", "topic1,topic2,topic3"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "format", "string"),
					resource.TestCheckResourceAttr("streamkap_source_kafkadirect.test", "schemas_enable", "true"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
