package provider

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// kafkaUsername builds a sweepable, unique username inside the backend's limits.
//
// acctestName is unusable here: Kafka usernames are capped at 24 characters and
// restricted to alphanumerics and hyphens (neither leading nor trailing), while
// acctestName embeds the full test name and a nanosecond stamp. The "tf-acc-test"
// prefix is what sweep_test.go's isTestResource matches, so a leaked user is
// still sweepable.
func kafkaUsername(t *testing.T) string {
	t.Helper()
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	return "tf-acc-test-" + stamp[len(stamp)-10:]
}

// TestAccKafkaUserResource covers the full lifecycle: create with one ACL,
// import, then update the password, the IP whitelist and the ACL set.
//
// The backend rejects Kafka users on tenants without a dedicated project
// namespace, or on a cloud provider other than AWS/Hetzner, with a 400 from
// validate_kafka_user_creation. That is an environment precondition, not a
// provider bug — the failure message names the reason.
func TestAccKafkaUserResource(t *testing.T) {
	username := kafkaUsername(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKafkaUserDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_kafka_user" "test" {
	username = %[1]q
	password = "TestPassword123!"

	kafka_acls {
		topic_name            = "test-topic"
		operation             = "READ"
		resource_pattern_type = "LITERAL"
		resource              = "TOPIC"
	}
}
`, username),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "username", username),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "id", username),
					resource.TestCheckResourceAttrSet("streamkap_kafka_user.test", "kafka_proxy_endpoint"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "is_create_schema_registry", "false"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "whitelist_ips", ""),
					// topic_name is the field the backend echoes under a different
					// key than it accepts; asserting it after apply is what catches
					// a regression in KafkaACL's JSON handling.
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.topic_name", "test-topic"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.operation", "READ"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.resource_pattern_type", "LITERAL"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.resource", "TOPIC"),
				),
			},
			// ImportState testing. password is write-only — the API never returns it.
			{
				ResourceName:            "streamkap_kafka_user.test",
				ImportState:             true,
				ImportStateId:           username,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update and Read testing: rotate the password, add a whitelist, and
			// change the ACL set (one operation edited, one GROUP rule added).
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_kafka_user" "test" {
	username      = %[1]q
	password      = "RotatedPassword456!"
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
		resource_pattern_type = "PREFIXED"
		resource              = "GROUP"
	}
}
`, username),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_kafka_user.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "username", username),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "whitelist_ips", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.#", "2"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.topic_name", "test-topic"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.0.operation", "ALL"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.1.topic_name", "test-group"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.1.resource", "GROUP"),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.1.resource_pattern_type", "PREFIXED"),
				),
			},
			// Clearing the whitelist must actually clear it. The update endpoint
			// takes whatever the request body carries, so an omitted-vs-empty
			// mix-up here shows up as a permanent diff.
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_kafka_user" "test" {
	username = %[1]q
	password = "RotatedPassword456!"

	kafka_acls {
		topic_name            = "test-topic"
		operation             = "ALL"
		resource_pattern_type = "LITERAL"
		resource              = "TOPIC"
	}
}
`, username),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "whitelist_ips", ""),
					resource.TestCheckResourceAttr("streamkap_kafka_user.test", "kafka_acls.#", "1"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccKafkaUserResource_schemaRegistry exercises the schema-registry proxy
// path, which provisions a second endpoint alongside the Kafka one and is the
// only way schema_proxy_endpoint gets populated.
func TestAccKafkaUserResource_schemaRegistry(t *testing.T) {
	username := kafkaUsername(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKafkaUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_kafka_user" "registry" {
	username                  = %[1]q
	password                  = "TestPassword123!"
	is_create_schema_registry = true

	kafka_acls {
		topic_name            = "tf-acc-"
		operation             = "READ"
		resource_pattern_type = "PREFIXED"
		resource              = "TOPIC"
	}
}
`, username),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_kafka_user.registry", "is_create_schema_registry", "true"),
					resource.TestCheckResourceAttrSet("streamkap_kafka_user.registry", "schema_proxy_endpoint"),
				),
			},
		},
	})
}

// TestAccKafkaUserResource_usernameForcesReplacement pins that a username change
// replaces the user rather than attempting an in-place rename: the API keys
// Kafka users by username and has no rename endpoint.
func TestAccKafkaUserResource_usernameForcesReplacement(t *testing.T) {
	first := kafkaUsername(t)
	second := kafkaUsername(t) + "b"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKafkaUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_kafka_user" "replace" {
	username = %[1]q
	password = "TestPassword123!"
}
`, first),
				Check: resource.TestCheckResourceAttr("streamkap_kafka_user.replace", "id", first),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_kafka_user" "replace" {
	username = %[1]q
	password = "TestPassword123!"
}
`, second),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("streamkap_kafka_user.replace", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("streamkap_kafka_user.replace", "id", second),
			},
		},
	})
}
