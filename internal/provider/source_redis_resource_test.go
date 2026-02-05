package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceRedisHost = os.Getenv("TF_VAR_source_redis_host")
var sourceRedisPassword = os.Getenv("TF_VAR_source_redis_password")

func TestAccSourceRedisResource(t *testing.T) {
	if sourceRedisHost == "" || sourceRedisPassword == "" {
		t.Skip("Skipping TestAccSourceRedisResource: TF_VAR_source_redis_host or TF_VAR_source_redis_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_redis_host" {
	type        = string
	description = "The hostname of the Redis server"
}
variable "source_redis_password" {
	type        = string
	sensitive   = true
	description = "The password for the Redis server"
}
resource "streamkap_source_redis" "test" {
	name                       = "tf-acc-test-source-redis"
	connector_class_type       = "Stream"
	redis_host                 = var.source_redis_host
	redis_port                 = 6379
	redis_password             = var.source_redis_password
	ssl_enabled                = true
	redis_stream_name          = "streamkap-test-stream"
	redis_stream_offset        = "Latest"
	redis_stream_delivery      = "At Least Once"
	redis_stream_block_seconds = 1
	redis_stream_consumer_group = "kafka-consumer-group"
	redis_stream_consumer_name = "consumer"
	mode                       = "LIVE"
	topic_use_stream_name      = false
	topic                      = "streamkap-redis-test-topic"
	tasks_max                  = 1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "name", "tf-acc-test-source-redis"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "connector_class_type", "Stream"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_host", sourceRedisHost),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_port", "6379"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_password", sourceRedisPassword),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "ssl_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_name", "streamkap-test-stream"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_offset", "Latest"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_delivery", "At Least Once"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_block_seconds", "1"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_consumer_group", "kafka-consumer-group"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_consumer_name", "consumer"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "mode", "LIVE"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "topic_use_stream_name", "false"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "topic", "streamkap-redis-test-topic"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "tasks_max", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_redis.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_redis_host" {
	type        = string
	description = "The hostname of the Redis server"
}
variable "source_redis_password" {
	type        = string
	sensitive   = true
	description = "The password for the Redis server"
}
resource "streamkap_source_redis" "test" {
	name                       = "tf-acc-test-source-redis-updated"
	connector_class_type       = "Stream"
	redis_host                 = var.source_redis_host
	redis_port                 = 6379
	redis_password             = var.source_redis_password
	ssl_enabled                = true
	redis_stream_name          = "streamkap-test-stream-updated"
	redis_stream_offset        = "Earliest"
	redis_stream_delivery      = "At Most Once"
	redis_stream_block_seconds = 5
	redis_stream_consumer_group = "kafka-consumer-group-updated"
	redis_stream_consumer_name = "consumer-updated"
	mode                       = "LIVEONLY"
	topic_use_stream_name      = true
	topic                      = "streamkap-redis-test-topic-updated"
	tasks_max                  = 2
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "name", "tf-acc-test-source-redis-updated"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_name", "streamkap-test-stream-updated"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_offset", "Earliest"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_delivery", "At Most Once"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "redis_stream_block_seconds", "5"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "mode", "LIVEONLY"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "topic_use_stream_name", "true"),
					resource.TestCheckResourceAttr("streamkap_source_redis.test", "tasks_max", "2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
