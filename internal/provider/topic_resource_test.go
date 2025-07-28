package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)


func TestAccTopicResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
			{
				Config: providerConfig + `
resource "streamkap_topic" "test" {
	topic_id                                   = "source_67adbcc172417ef6338e01a1.default.tst-junit-2"
  	partition_count                            = 25
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_topic.test", "topic_id", "source_67adbcc172417ef6338e01a1.default.tst-junit-2"),
					resource.TestCheckResourceAttr("streamkap_topic.test", "partition_count", 25)
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:      "topic_id.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read testing
			{
				Config: providerConfig + `
resource "streamkap_topic" "test" {
	topic_id                                   = "source_67adbcc172417ef6338e01a1.default.tst-junit-2"
  	partition_count                            = 26
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_topic.test", "topic_id", "source_67adbcc172417ef6338e01a1.default.tst-junit-2"),
					resource.TestCheckResourceAttr("streamkap_topic.test", "partition_count", 26)
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
