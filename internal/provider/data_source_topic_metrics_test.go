package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var dataSourceTopicMetricsSourceID = os.Getenv("TF_VAR_data_source_topic_metrics_source_id")
var dataSourceTopicMetricsTopicName = os.Getenv("TF_VAR_data_source_topic_metrics_topic_name")
var dataSourceTopicMetricsTopicDBID = os.Getenv("TF_VAR_data_source_topic_metrics_topic_db_id")

func TestAccDataSourceTopicMetrics(t *testing.T) {
	if dataSourceTopicMetricsSourceID == "" || dataSourceTopicMetricsTopicName == "" || dataSourceTopicMetricsTopicDBID == "" {
		t.Skip("Skipping TestAccDataSourceTopicMetrics: source ID, topic name, and topic database ID variables must be set")
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

variable "data_source_topic_metrics_topic_db_id" {
	type = string
}

data "streamkap_topic_metrics" "test" {
	entities {
		id           = var.data_source_topic_metrics_source_id
		entity_type  = "sources"
		connector    = "postgresql"
		topic_ids    = [var.data_source_topic_metrics_topic_name]
		topic_db_ids = [var.data_source_topic_metrics_topic_db_id]
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.streamkap_topic_metrics.test", "results.#"),
					resource.TestCheckResourceAttr("data.streamkap_topic_metrics.test", "results.0.entity_id", dataSourceTopicMetricsSourceID),
					resource.TestCheckResourceAttrSet("data.streamkap_topic_metrics.test", "results.0.topic_id"),
					resource.TestCheckResourceAttrSet("data.streamkap_topic_metrics.test", "results.0.id"),
					resource.TestCheckResourceAttrSet("data.streamkap_topic_metrics.test", "results.0.snapshot_status_json"),
				),
			},
		},
	})
}
