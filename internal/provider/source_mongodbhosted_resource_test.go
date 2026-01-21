package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceMongoDBHostedConnectionString = os.Getenv("TF_VAR_source_mongodbhosted_connection_string")

func TestAccSourceMongoDBHostedResource(t *testing.T) {
	if sourceMongoDBHostedConnectionString == "" {
		t.Skip("Skipping TestAccSourceMongoDBHostedResource: TF_VAR_source_mongodbhosted_connection_string not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_mongodbhosted_connection_string" {
	type        = string
	sensitive   = true
	description = "The MongoDB connection string"
}
resource "streamkap_source_mongodbhosted" "test" {
	name                                     = "tf-acc-test-source-mongodbhosted"
	mongodb_connection_string                = var.source_mongodbhosted_connection_string
	transforms_unwrap_array_encoding         = "array_string"
	transforms_unwrap_document_encoding      = "document"
	database_include_list                    = "streamkap"
	collection_include_list                  = "streamkap.customer"
	signal_data_collection_schema_or_database = "streamkap"
	ssh_enabled                              = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "name", "tf-acc-test-source-mongodbhosted"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "mongodb_connection_string", sourceMongoDBHostedConnectionString),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "transforms_unwrap_array_encoding", "array_string"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "transforms_unwrap_document_encoding", "document"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "database_include_list", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "collection_include_list", "streamkap.customer"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_mongodbhosted.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_mongodbhosted_connection_string" {
	type        = string
	sensitive   = true
	description = "The MongoDB connection string"
}
resource "streamkap_source_mongodbhosted" "test" {
	name                                     = "tf-acc-test-source-mongodbhosted-updated"
	mongodb_connection_string                = var.source_mongodbhosted_connection_string
	transforms_unwrap_array_encoding         = "array"
	transforms_unwrap_document_encoding      = "string"
	database_include_list                    = "streamkap"
	collection_include_list                  = "streamkap.customer,streamkap.orders"
	signal_data_collection_schema_or_database = "streamkap"
	ssh_enabled                              = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "name", "tf-acc-test-source-mongodbhosted-updated"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "transforms_unwrap_array_encoding", "array"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "transforms_unwrap_document_encoding", "string"),
					resource.TestCheckResourceAttr("streamkap_source_mongodbhosted.test", "collection_include_list", "streamkap.customer,streamkap.orders"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
