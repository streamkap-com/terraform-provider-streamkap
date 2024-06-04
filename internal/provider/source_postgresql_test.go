package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourcePostgreSQLHostname = os.Getenv("SOURCE_POSTGRESQL_HOSTNAME")
var sourcePostgreSQLPassword = os.Getenv("SOURCE_POSTGRESQL_PASSWORD")

func TestAccSourcePostgreSQLResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_source_postgresql" "test" {
	name                                         = "test-source-postgresql"
	database_hostname                            = "%s"
	database_port                                = 5432
	database_user                                = "postgresql"
	database_password                            = "%s"
	database_dbname                              = "postgres"
	database_sslmode                             = "require"
	schema_include_list                          = "public"
	table_include_list                           = "public.users"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = null
	include_source_db_name_in_table_name         = false
	slot_name                                    = "streamkap_pgoutput_slot"
	publication_name                             = "streamkap_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`, sourcePostgreSQLHostname, sourcePostgreSQLPassword),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-source-postgresql"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_hostname", sourcePostgreSQLHostname),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_port", "5432"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_user", "postgresql"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_password", sourcePostgreSQLPassword),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_dbname", "postgres"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "schema_include_list", "public"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "table_include_list", "public.users"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "signal_data_collection_schema_or_database", "streamkap"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "slot_name", "streamkap_pgoutput_slot"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "publication_name", "streamkap_pub"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_postgresql.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_source_postgresql" "test" {
	name                                         = "test-source-postgresql-updated"
	database_hostname                            = "%s"
	database_port                                = 5432
	database_user                                = "postgresql"
	database_password                            = "%s"
	database_dbname                              = "postgres"
	database_sslmode                             = "require"
	schema_include_list                          = "public"
	table_include_list                           = "public.users"
	signal_data_collection_schema_or_database    = "streamkap"
	heartbeat_enabled                            = false
	include_source_db_name_in_table_name         = false
	slot_name                                    = "streamkap_pgoutput_slot"
	publication_name                             = "streamkap_pub"
	binary_handling_mode                         = "bytes"
	ssh_enabled                                  = false
}
`, sourcePostgreSQLHostname, sourcePostgreSQLPassword),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "name", "test-source-postgresql-updated"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_hostname", sourcePostgreSQLHostname),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_port", "5432"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_user", "postgresql"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_password", sourcePostgreSQLPassword),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "database_dbname", "postgres"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "schema_include_list", "public"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "table_include_list", "public.users"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "signal_data_collection_schema_or_database", "streamkap"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "slot_name", "streamkap_pgoutput_slot"),
          resource.TestCheckResourceAttr("streamkap_source_postgresql.test", "publication_name", "streamkap_pub"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
