package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationKafkaResource_basic(t *testing.T) {
	bootstrapServers := os.Getenv("TF_VAR_destination_kafka_bootstrap_servers")
	if bootstrapServers == "" {
		t.Skip("TF_VAR_destination_kafka_bootstrap_servers not set, skipping test")
	}

	name := acctestName(t, "main")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDestinationKafkaResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_kafka.test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafka.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_destination_kafka.test", "connector"),
				),
			},
			{
				ResourceName:            "streamkap_destination_kafka.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"sasl_password", "connector_status"},
			},
		},
	})
}

func testAccDestinationKafkaResourceConfig(name string) string {
	return fmt.Sprintf(`
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
  name              = %q
  kafka_sink_bootstrap = var.destination_kafka_bootstrap_servers
  destination_format   = "avro"
  schema_registry_url  = "https://schema-registry.example.com"
  transforms_mask_field_fields_include_list       = "public.users.email"
  transforms_mask_field_fields_exclude_list       = "public.users.id"
  transforms_mask_field_mask_function             = "REDACT"
  transforms_mask_field_mask_char                 = "#"
  transforms_mask_field_replace_null_with_default = false
}
`, name)
}
