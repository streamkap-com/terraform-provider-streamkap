package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceMongoDBConnectionString = os.Getenv("TF_VAR_source_mongodb_connection_string")
var sourceMongoDBSSHHost = os.Getenv("TF_VAR_source_mongodb_ssh_host")

func TestAccSourceMongoDBResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
			{
				Config: providerConfig + `
variable "source_mongodb_connection_string" {
	type        = string
	sensitive   = true
	description = "The connection string of the MongoDB database"
}
resource "streamkap_source_mongodb" "test" {
	name                                         = "test-source-mongodb"
	mongodb_connection_string                    = var.source_mongodb_connection_string
	database_include_list                        = "Test"
	collection_include_list                      = "Test.test_data,Test.test_data2"
	signal_data_collection_schema_or_database    = "Test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "name", "test-source-mongodb"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "mongodb_connection_string", sourceMongoDBConnectionString),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "database_include_list", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "collection_include_list", "Test.test_data,Test.test_data2"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "signal_data_collection_schema_or_database", "Test"),
					// Check computed and default values
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "connector", "mongodb"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "array_encoding", "array_string"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "nested_document_encoding", "document"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "ssh_enabled", "false"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:      "streamkap_source_mongodb.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read testing
			{
				Config: providerConfig + `
variable "source_mongodb_connection_string" {
	type        = string
	sensitive   = true
	description = "The connection string of the MongoDB database"
}
resource "streamkap_source_mongodb" "test" {
	name                                         = "test-source-mongodb-updated"
	mongodb_connection_string                    = var.source_mongodb_connection_string
	database_include_list                        = "Test"
	collection_include_list                      = "Test.test_data,Test.test_data2,Test.test_data3"
	signal_data_collection_schema_or_database    = "Test"
	array_encoding                               = "array"
	nested_document_encoding                     = "string"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "name", "test-source-mongodb-updated"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "mongodb_connection_string", sourceMongoDBConnectionString),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "database_include_list", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "collection_include_list", "Test.test_data,Test.test_data2,Test.test_data3"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "signal_data_collection_schema_or_database", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "array_encoding", "array"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "nested_document_encoding", "string"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "ssh_enabled", "false"),
				),
			},
			// Step 4: Update to test SSH enabled
			{
				Config: providerConfig + `
variable "source_mongodb_connection_string" {
	type        = string
	sensitive   = true
	description = "The connection string of the MongoDB database"
}
variable "source_mongodb_ssh_host" {
	type        = string
	description = "The SSH host for the MongoDB database"
}
resource "streamkap_source_mongodb" "test" {
	name                                         = "test-source-mongodb-ssh"
	mongodb_connection_string                    = var.source_mongodb_connection_string
	database_include_list                        = "Test"
	collection_include_list                      = "Test.test_data,Test.test_data2"
	signal_data_collection_schema_or_database    = "Test"
	ssh_enabled                                  = true
	ssh_host                                     = var.source_mongodb_ssh_host
	ssh_port                                     = "22"
	ssh_user                                     = "streamkap"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "name", "test-source-mongodb-ssh"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "mongodb_connection_string", sourceMongoDBConnectionString),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "database_include_list", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "collection_include_list", "Test.test_data,Test.test_data2"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "signal_data_collection_schema_or_database", "Test"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "ssh_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "ssh_host", sourceMongoDBSSHHost),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "ssh_port", "22"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "ssh_user", "streamkap"),
					// Check that optional fields revert to their default values
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "array_encoding", "array_string"),
					resource.TestCheckResourceAttr("streamkap_source_mongodb.test", "nested_document_encoding", "document"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
