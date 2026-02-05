package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationRedisHost = os.Getenv("TF_VAR_destination_redis_host")
var destinationRedisUsername = os.Getenv("TF_VAR_destination_redis_username")
var destinationRedisPassword = os.Getenv("TF_VAR_destination_redis_password")

func TestAccDestinationRedisResource(t *testing.T) {
	if destinationRedisHost == "" || destinationRedisUsername == "" || destinationRedisPassword == "" {
		t.Skip("Skipping TestAccDestinationRedisResource: TF_VAR_destination_redis_host, TF_VAR_destination_redis_username, or TF_VAR_destination_redis_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_redis_host" {
	type        = string
	description = "Redis hostname"
}
variable "destination_redis_username" {
	type        = string
	description = "Redis username"
}
variable "destination_redis_password" {
	type        = string
	sensitive   = true
	description = "Redis password"
}
resource "streamkap_destination_redis" "test" {
	name                = "tf-acc-test-destination-redis"
	redis_host          = var.destination_redis_host
	redis_port          = 6379
	redis_username      = var.destination_redis_username
	redis_password      = var.destination_redis_password
	ssl_enabled         = true
	redis_key           = "streamkap-test-key"
	redis_key_data_type = "Stream"
	tasks_max           = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "name", "tf-acc-test-destination-redis"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "redis_host", destinationRedisHost),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "redis_port", "6379"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "redis_username", destinationRedisUsername),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "ssl_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "redis_key", "streamkap-test-key"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "redis_key_data_type", "Stream"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_redis.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "connector", "redis"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_redis.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_redis_host" {
	type        = string
	description = "Redis hostname"
}
variable "destination_redis_username" {
	type        = string
	description = "Redis username"
}
variable "destination_redis_password" {
	type        = string
	sensitive   = true
	description = "Redis password"
}
resource "streamkap_destination_redis" "test" {
	name                = "tf-acc-test-destination-redis-updated"
	redis_host          = var.destination_redis_host
	redis_port          = 6379
	redis_username      = var.destination_redis_username
	redis_password      = var.destination_redis_password
	ssl_enabled         = false
	redis_key           = "streamkap-test-key-updated"
	redis_key_data_type = "List"
	tasks_max           = 10
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "name", "tf-acc-test-destination-redis-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "ssl_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "redis_key", "streamkap-test-key-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "redis_key_data_type", "List"),
					resource.TestCheckResourceAttr("streamkap_destination_redis.test", "tasks_max", "10"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
