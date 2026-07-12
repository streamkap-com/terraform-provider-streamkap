package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceDocumentDBConnectionString = os.Getenv("TF_VAR_source_documentdb_connection_string")

func TestAccSourceDocumentDBResource(t *testing.T) {
	if sourceDocumentDBConnectionString == "" {
		t.Skip("Skipping TestAccSourceDocumentDBResource: TF_VAR_source_documentdb_connection_string not set")
	}

	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "source_documentdb_connection_string" {
	type        = string
	sensitive   = true
	description = "The DocumentDB connection string"
}
resource "streamkap_source_documentdb" "test" {
	name                                     = %q
	mongodb_connection_string                = var.source_documentdb_connection_string
	transforms_unwrap_array_encoding         = "array_string"
	transforms_unwrap_document_encoding      = "document"
	database_include_list                    = "streamkap"
	collection_include_list                  = "streamkap.customers"
	signal_data_collection_schema_or_database = "streamkap"
	ssh_enabled                              = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "mongodb_connection_string", sourceDocumentDBConnectionString),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "transforms_unwrap_array_encoding", "array_string"),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "transforms_unwrap_document_encoding", "document"),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "database_include_list", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "collection_include_list", "streamkap.customers"),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "signal_data_collection_schema_or_database", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_source_documentdb.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "source_documentdb_connection_string" {
	type        = string
	sensitive   = true
	description = "The DocumentDB connection string"
}
resource "streamkap_source_documentdb" "test" {
	name                                     = %q
	mongodb_connection_string                = var.source_documentdb_connection_string
	transforms_unwrap_array_encoding         = "array"
	transforms_unwrap_document_encoding      = "string"
	database_include_list                    = "streamkap"
	collection_include_list                  = "streamkap.customers,streamkap.orders"
	signal_data_collection_schema_or_database = "streamkap"
	ssh_enabled                              = false
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "transforms_unwrap_array_encoding", "array"),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "transforms_unwrap_document_encoding", "string"),
					resource.TestCheckResourceAttr("streamkap_source_documentdb.test", "collection_include_list", "streamkap.customers,streamkap.orders"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
