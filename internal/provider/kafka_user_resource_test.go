package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKafkaUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKafkaUserDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "streamkap_kafka_user" "test" {
	username = "tf-acc-test-kafkauser"
	password = "TestPassword123!"

	kafka_acls {
		topic_name            = "test-topic"
		operation             = "READ"
		resource_pattern_type = "LITERAL"
		resource              = "TOPIC"
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "username", "tf-acc-test-kafkauser"),
					resource.TestCheckResourceAttrSet("streamkap_kafka_user.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_kafka_user.test", "kafka_proxy_endpoint"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "is_create_schema_registry", "false"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.topic_name", "test-topic"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.operation", "READ"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.resource_pattern_type", "LITERAL"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.resource", "TOPIC"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_kafka_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "streamkap_kafka_user" "test" {
	username      = "tf-acc-test-kafkauser"
	password      = "TestPassword123!"
	whitelist_ips = "192.168.1.0/24"

	kafka_acls {
		topic_name            = "test-topic"
		operation             = "ALL"
		resource_pattern_type = "LITERAL"
		resource              = "TOPIC"
	}

	kafka_acls {
		topic_name            = "test-group"
		operation             = "READ"
		resource_pattern_type = "LITERAL"
		resource              = "GROUP"
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "username", "tf-acc-test-kafkauser"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "whitelist_ips", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.operation", "ALL"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.1.topic_name", "test-group"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.1.resource", "GROUP"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
