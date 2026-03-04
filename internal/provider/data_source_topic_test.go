package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var dataSourceTopicID = os.Getenv("TF_VAR_data_source_topic_id")

func TestAccDataSourceTopic(t *testing.T) {
	if dataSourceTopicID == "" {
		t.Skip("Skipping TestAccDataSourceTopic: TF_VAR_data_source_topic_id not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
variable "data_source_topic_id" {
	type = string
}

data "streamkap_topic" "test" {
	id = var.data_source_topic_id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.streamkap_topic.test", "id", dataSourceTopicID),
					resource.TestCheckResourceAttrSet("data.streamkap_topic.test", "name"),
				),
			},
		},
	})
}
