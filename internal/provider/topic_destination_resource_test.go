package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccTopicDestinationConfig(name string, objects string) string {
	return providerConfig + fmt.Sprintf(`
variable "source_hubspot_token" {
	type        = string
	sensitive   = true
	description = "HubSpot private app token"
}
variable "destination_clickhouse_hostname" {
	type        = string
	description = "The hostname of the ClickHouse database"
}
variable "destination_clickhouse_connection_username" {
	type        = string
	description = "The username for the ClickHouse database"
}
variable "destination_clickhouse_connection_password" {
	type        = string
	sensitive   = true
	description = "The password for the ClickHouse database"
}
resource "streamkap_source_hubspot" "test" {
	name      = %[1]q
	token     = var.source_hubspot_token
	resources = ["contacts", "companies"]
}
resource "streamkap_destination_clickhouse" "test" {
	name                = %[1]q
	hostname            = var.destination_clickhouse_hostname
	connection_username = var.destination_clickhouse_connection_username
	connection_password = var.destination_clickhouse_connection_password
	ingestion_mode      = "upsert"
	port                = 8123
	database            = "default"
	ssl                 = false
}
resource "streamkap_topic_destination" "test" {
	for_each       = toset(%[2]s)
	topic_id       = "source_${streamkap_source_hubspot.test.id}.hubspot.${each.value}"
	destination_id = streamkap_destination_clickhouse.test.id
}
`, name, objects)
}

func TestAccTopicDestinationResource(t *testing.T) {
	if sourceHubSpotToken == "" || os.Getenv("TF_VAR_destination_clickhouse_hostname") == "" {
		t.Skip("Skipping TestAccTopicDestinationResource: TF_VAR_source_hubspot_token or TF_VAR_destination_clickhouse_hostname not set")
	}

	name := acctestName(t, "main")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			if err := testAccCheckTopicDestinationDestroy(s); err != nil {
				return err
			}
			if err := testAccCheckSourceDestroy(s); err != nil {
				return err
			}
			return testAccCheckDestinationDestroy(s)
		},
		Steps: []resource.TestStep{
			{
				// Both links share one managed binding and are created in parallel.
				Config: testAccTopicDestinationConfig(name, `["contacts", "companies"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(`streamkap_topic_destination.test["contacts"]`, "binding_id"),
					resource.TestCheckResourceAttrPair(
						`streamkap_topic_destination.test["contacts"]`, "binding_id",
						`streamkap_topic_destination.test["companies"]`, "binding_id",
					),
				),
			},
			{
				ResourceName:      `streamkap_topic_destination.test["contacts"]`,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Removing one link must leave the other attached; the post-apply
				// refresh would drop companies from state if it had been detached.
				Config: testAccTopicDestinationConfig(name, `["companies"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						if _, ok := s.RootModule().Resources[`streamkap_topic_destination.test["contacts"]`]; ok {
							return fmt.Errorf("contacts link is still in state")
						}
						return nil
					},
					resource.TestCheckResourceAttrSet(`streamkap_topic_destination.test["companies"]`, "binding_id"),
				),
			},
		},
	})
}
