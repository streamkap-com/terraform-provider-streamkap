package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationKafkaDirectPassword = os.Getenv("TF_VAR_destination_kafkadirect_password")
var destinationKafkaDirectWhitelistIps = os.Getenv("TF_VAR_destination_kafkadirect_whitelist_ips")

func TestAccDestinationKafkadirectResource(t *testing.T) {
	if destinationKafkaDirectPassword == "" || destinationKafkaDirectWhitelistIps == "" {
		t.Skip("Skipping TestAccDestinationKafkadirectResource: TF_VAR_destination_kafkadirect_password or TF_VAR_destination_kafkadirect_whitelist_ips not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_kafkadirect_password" {
	type        = string
	sensitive   = true
	description = "Kafka Direct password"
}
variable "destination_kafkadirect_whitelist_ips" {
	type        = string
	description = "Kafka Direct whitelist IPs"
}
resource "streamkap_destination_kafkadirect" "test" {
	name          = "tf-acc-test-destination-kafkadirect"
	password      = var.destination_kafkadirect_password
	whitelist_ips = var.destination_kafkadirect_whitelist_ips
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_kafkadirect.test", "name", "tf-acc-test-destination-kafkadirect"),
					resource.TestCheckResourceAttr("streamkap_destination_kafkadirect.test", "whitelist_ips", destinationKafkaDirectWhitelistIps),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafkadirect.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_kafkadirect.test", "connector", "kafkadirect"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_kafkadirect.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_kafkadirect_password" {
	type        = string
	sensitive   = true
	description = "Kafka Direct password"
}
variable "destination_kafkadirect_whitelist_ips" {
	type        = string
	description = "Kafka Direct whitelist IPs"
}
resource "streamkap_destination_kafkadirect" "test" {
	name          = "tf-acc-test-destination-kafkadirect-updated"
	password      = var.destination_kafkadirect_password
	whitelist_ips = "10.0.0.0/8,192.168.0.0/16"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_kafkadirect.test", "name", "tf-acc-test-destination-kafkadirect-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_kafkadirect.test", "whitelist_ips", "10.0.0.0/8,192.168.0.0/16"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
