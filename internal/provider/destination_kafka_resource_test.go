package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationKafkaResource_basic(t *testing.T) {
	bootstrapServers := os.Getenv("TF_VAR_destination_kafka_bootstrap_servers")
	if bootstrapServers == "" {
		t.Skip("TF_VAR_destination_kafka_bootstrap_servers not set, skipping test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDestinationKafkaResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "name", "tf-acc-test-destination-kafka"),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafka.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafka.test", "connector"),
				),
			},
			{
				ResourceName:            "streamkap_destination_kafka.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"sasl_password"},
			},
		},
	})
}

func testAccDestinationKafkaResourceConfig() string {
	return `
variable "destination_kafka_bootstrap_servers" {
  type = string
}

variable "destination_kafka_sasl_username" {
  type    = string
  default = ""
}

variable "destination_kafka_sasl_password" {
  type      = string
  sensitive = true
  default   = ""
}

resource "streamkap_destination_kafka" "test" {
  name              = "tf-acc-test-destination-kafka"
  kafka_sink_bootstrap = var.destination_kafka_bootstrap_servers
  destination_format   = "avro"
  schema_registry_url  = "https://schema-registry-dev.streamkap.net"
}
`
}
