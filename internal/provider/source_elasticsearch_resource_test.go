package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceElasticsearchHost = os.Getenv("TF_VAR_source_elasticsearch_host")
var sourceElasticsearchPassword = os.Getenv("TF_VAR_source_elasticsearch_password")

func TestAccSourceElasticsearchResource(t *testing.T) {
	if sourceElasticsearchHost == "" || sourceElasticsearchPassword == "" {
		t.Skip("Skipping TestAccSourceElasticsearchResource: TF_VAR_source_elasticsearch_host or TF_VAR_source_elasticsearch_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_elasticsearch_host" {
	type        = string
	description = "The hostname of the Elasticsearch cluster"
}
variable "source_elasticsearch_password" {
	type        = string
	sensitive   = true
	description = "The password of the Elasticsearch user"
}
resource "streamkap_source_elasticsearch" "test" {
	name                  = "tf-acc-test-source-elasticsearch"
	es_host               = var.source_elasticsearch_host
	es_scheme             = "https"
	es_port               = 443
	http_auth             = "Basic"
	http_auth_user        = "elastic"
	http_auth_password    = var.source_elasticsearch_password
	endpoint_include_list = "my-index"
	datetime_field_name   = "timestamp"
	tasks_max             = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "name", "tf-acc-test-source-elasticsearch"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "es_host", sourceElasticsearchHost),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "es_scheme", "https"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "es_port", "443"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "http_auth", "Basic"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "http_auth_user", "elastic"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "http_auth_password", sourceElasticsearchPassword),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "endpoint_include_list", "my-index"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "datetime_field_name", "timestamp"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "tasks_max", "5"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_elasticsearch.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_elasticsearch_host" {
	type        = string
	description = "The hostname of the Elasticsearch cluster"
}
variable "source_elasticsearch_password" {
	type        = string
	sensitive   = true
	description = "The password of the Elasticsearch user"
}
resource "streamkap_source_elasticsearch" "test" {
	name                  = "tf-acc-test-source-elasticsearch-updated"
	es_host               = var.source_elasticsearch_host
	es_scheme             = "https"
	es_port               = 9200
	http_auth             = "Basic"
	http_auth_user        = "elastic"
	http_auth_password    = var.source_elasticsearch_password
	endpoint_include_list = "my-index,my-other-index"
	datetime_field_name   = "updated_at"
	tasks_max             = 3
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "name", "tf-acc-test-source-elasticsearch-updated"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "es_port", "9200"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "endpoint_include_list", "my-index,my-other-index"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "datetime_field_name", "updated_at"),
					resource.TestCheckResourceAttr("streamkap_source_elasticsearch.test", "tasks_max", "3"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
