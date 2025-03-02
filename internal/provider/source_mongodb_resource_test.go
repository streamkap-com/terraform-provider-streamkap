package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceMongoDBConnectionString = os.Getenv("TF_VAR_source_mongodb_connection_string")

func TestAccSourceMongoDBResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_mongodb_connection_string" {
	type        = string
	description = "The connection string of the MongoDB database"
}

resource "streamkap_source_mongodb" "test" {
	name                                      = "test-source-mongodb"
	mongodb_connection_string                 = var.source_mongodb_connection_string
	array_encoding                            = "array_string"
	nested_document_encoding                  = "document"
	database_include_list                     = "Test"
	collection_include_list                   = "Test.test_data4,Test.test_data2"
	signal_data_collection_schema_or_database = "Test"
	ssh_enabled                               = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "name", "test-source-mongodb"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "mongodb_connection_string", sourceMongoDBConnectionString),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "array_encoding", "array_string"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "nested_document_encoding", "document"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "database_include_list", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "collection_include_list", "Test.test_data4,Test.test_data2"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "signal_data_collection_schema_or_database", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_mongodb.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_mongodb_connection_string" {
	type        = string
	description = "The connection string of the MongoDB database"
}

resource "streamkap_source_mongodb" "test" {
	name                                      = "test-source-mongodb-updated"
	mongodb_connection_string                 = var.source_mongodb_connection_string
	array_encoding                            = "array_string"
	nested_document_encoding                  = "document"
	database_include_list                     = "Test"
	collection_include_list                   = "Test.test_data4"
	signal_data_collection_schema_or_database = "Test"
	ssh_enabled                               = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "name", "test-source-mongodb-updated"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "mongodb_connection_string", sourceMongoDBConnectionString),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "array_encoding", "array_string"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "nested_document_encoding", "document"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "database_include_list", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "collection_include_list", "Test.test_data4"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "signal_data_collection_schema_or_database", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "ssh_enabled", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
