package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var dataSourceTopicMetricsSourceID = os.Getenv("TF_VAR_data_source_topic_metrics_source_id")
var dataSourceTopicMetricsTopicName = os.Getenv("TF_VAR_data_source_topic_metrics_topic_name")

func TestAccDataSourceTopicMetrics(t *testing.T) {
	if dataSourceTopicMetricsSourceID == "" || dataSourceTopicMetricsTopicName == "" {
		t.Skip("Skipping TestAccDataSourceTopicMetrics: TF_VAR_data_source_topic_metrics_source_id or TF_VAR_data_source_topic_metrics_topic_name not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
variable "data_source_topic_metrics_source_id" {
	type = string
}

variable "data_source_topic_metrics_topic_name" {
	type = string
}

data "streamkap_topic_metrics" "test" {
	entities {
		source_id  = var.data_source_topic_metrics_source_id
		topic_name = var.data_source_topic_metrics_topic_name
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.streamkap_topic_metrics.test", "results.#"),
				),
			},
		},
	})
}
