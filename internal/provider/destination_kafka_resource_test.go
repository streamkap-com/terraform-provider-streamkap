package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationKafkaResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "streamkap_destination_kafka" "test" {
	name                 = "test-destination-kafka"
	kafka_sink_bootstrap = "localhost:9092"
	destination_format   = "json"
	json_schema_enable   = false
	topic_prefix         = "test-prefix"
	topic_suffix         = "-suffix"
}
`,

				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "name", "test-destination-kafka"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "kafka_sink_bootstrap", "localhost:9092"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "destination_format", "json"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "json_schema_enable", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "topic_prefix", "test-prefix"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "topic_suffix", "-suffix"),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafka.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "connector", "kafka"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_kafka.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "streamkap_destination_kafka" "test" {
	name                             = "test-destination-kafka-updated"
	kafka_sink_bootstrap             = "kafka-broker:9092,kafka-broker2:9092"
	destination_format               = "avro"
	schema_registry_url_user_defined = "http://schema-registry:8081"
	topic_prefix                     = "updated-prefix"
	topic_suffix                     = "-new-suffix"
}
`,

				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "name", "test-destination-kafka-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "kafka_sink_bootstrap", "kafka-broker:9092,kafka-broker2:9092"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "destination_format", "avro"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "schema_registry_url_user_defined", "http://schema-registry:8081"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "topic_prefix", "updated-prefix"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "topic_suffix", "-new-suffix"),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafka.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "connector", "kafka"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
