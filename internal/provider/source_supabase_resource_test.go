package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceSupabaseHostname = os.Getenv("TF_VAR_source_supabase_hostname")
var sourceSupabasePassword = os.Getenv("TF_VAR_source_supabase_password")

func TestAccSourceSupabaseResource(t *testing.T) {
	if sourceSupabaseHostname == "" || sourceSupabasePassword == "" {
		t.Skip("Skipping TestAccSourceSupabaseResource: TF_VAR_source_supabase_hostname or TF_VAR_source_supabase_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_supabase_hostname" {
	type        = string
	description = "The hostname of the Supabase database"
}
variable "source_supabase_password" {
	type        = string
	sensitive   = true
	description = "The password of the Supabase database"
}
resource "streamkap_source_supabase" "test" {
	name                                         = "tf-acc-test-source-supabase"
	database_hostname                            = var.source_supabase_hostname
	database_port                                = 5432
	database_user                                = "streamkap"
	database_password                            = var.source_supabase_password
	database_dbname                              = "sandbox"
	snapshot_read_only                           = "Yes"
	database_sslmode                             = "require"
	schema_include_list                          = "public"
	table_include_list                           = "public.users"
	signal_data_collection_schema_or_database    = "public"
	heartbeat_enabled                            = true
	heartbeat_data_collection_schema_or_database = "public"
	slot_name                                    = "streamkap_pgoutput_slot_n"
	publication_name                             = "streamkap_pub_n"
	binary_handling_mode                         = "bytes"
	include_source_db_name_in_table_name         = false
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "name", "tf-acc-test-source-supabase"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "database_hostname", sourceSupabaseHostname),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "database_port", "5432"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "database_user", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "database_password", sourceSupabasePassword),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "database_dbname", "sandbox"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "snapshot_read_only", "Yes"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "database_sslmode", "require"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "schema_include_list", "public"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "table_include_list", "public.users"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "signal_data_collection_schema_or_database", "public"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "heartbeat_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "heartbeat_data_collection_schema_or_database", "public"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "slot_name", "streamkap_pgoutput_slot"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "publication_name", "streamkap_pub"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "binary_handling_mode", "bytes"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "include_source_db_name_in_table_name", "false"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "ssh_enabled", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_supabase.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_supabase_hostname" {
	type        = string
	description = "The hostname of the Supabase database"
}
variable "source_supabase_password" {
	type        = string
	sensitive   = true
	description = "The password of the Supabase database"
}
resource "streamkap_source_supabase" "test" {
	name                                         = "tf-acc-test-source-supabase-updated"
	database_hostname                            = var.source_supabase_hostname
	database_port                                = 5432
	database_user                                = "streamkap"
	database_password                            = var.source_supabase_password
	database_dbname                              = "sandbox"
	snapshot_read_only                           = "No"
	database_sslmode                             = "require"
	schema_include_list                          = "public"
	table_include_list                           = "public.users,public.orders"
	signal_data_collection_schema_or_database    = "public"
	heartbeat_enabled                            = false
	heartbeat_data_collection_schema_or_database = "public"
	slot_name                                    = "streamkap_pgoutput_slot"
	publication_name                             = "streamkap_pub"
	binary_handling_mode                         = "base64"
	include_source_db_name_in_table_name         = true
	ssh_enabled                                  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "name", "tf-acc-test-source-supabase-updated"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "snapshot_read_only", "No"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "table_include_list", "public.users,public.orders"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "heartbeat_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "binary_handling_mode", "base64"),
					resource.TestCheckResourceAttr("streamkap_source_supabase.test", "include_source_db_name_in_table_name", "true"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
