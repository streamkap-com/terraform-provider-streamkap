package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationRedshiftDomain = os.Getenv("TF_VAR_destination_redshift_domain")
var destinationRedshiftUsername = os.Getenv("TF_VAR_destination_redshift_username")
var destinationRedshiftPassword = os.Getenv("TF_VAR_destination_redshift_password")
var destinationRedshiftDatabase = os.Getenv("TF_VAR_destination_redshift_database")

func TestAccDestinationRedshiftResource(t *testing.T) {
	if destinationRedshiftDomain == "" || destinationRedshiftUsername == "" || destinationRedshiftPassword == "" {
		t.Skip("Skipping TestAccDestinationRedshiftResource: TF_VAR_destination_redshift_domain, TF_VAR_destination_redshift_username, or TF_VAR_destination_redshift_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_redshift_domain" {
	type        = string
	description = "Redshift cluster domain"
}
variable "destination_redshift_username" {
	type        = string
	description = "Redshift username"
}
variable "destination_redshift_password" {
	type        = string
	sensitive   = true
	description = "Redshift password"
}
variable "destination_redshift_database" {
	type        = string
	description = "Redshift database name"
	default     = ""
}
resource "streamkap_destination_redshift" "test" {
	name                 = "tf-acc-test-destination-redshift"
	aws_redshift_domain  = var.destination_redshift_domain
	aws_redshift_port    = 5439
	aws_redshift_database = var.destination_redshift_database
	connection_username  = var.destination_redshift_username
	connection_password  = var.destination_redshift_password
	primary_key_fields   = "id"
	schema_evolution     = "basic"
	table_name_prefix    = "streamkap"
	tasks_max            = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "name", "tf-acc-test-destination-redshift"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "aws_redshift_domain", destinationRedshiftDomain),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "aws_redshift_port", "5439"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "connection_username", destinationRedshiftUsername),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "primary_key_fields", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "table_name_prefix", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_redshift.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "connector", "redshift"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_redshift.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_redshift_domain" {
	type        = string
	description = "Redshift cluster domain"
}
variable "destination_redshift_username" {
	type        = string
	description = "Redshift username"
}
variable "destination_redshift_password" {
	type        = string
	sensitive   = true
	description = "Redshift password"
}
variable "destination_redshift_database" {
	type        = string
	description = "Redshift database name"
	default     = ""
}
resource "streamkap_destination_redshift" "test" {
	name                 = "tf-acc-test-destination-redshift-updated"
	aws_redshift_domain  = var.destination_redshift_domain
	aws_redshift_port    = 5439
	aws_redshift_database = var.destination_redshift_database
	connection_username  = var.destination_redshift_username
	connection_password  = var.destination_redshift_password
	primary_key_fields   = "id,created_at"
	schema_evolution     = "none"
	table_name_prefix    = "streamkap_updated"
	tasks_max            = 10
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "name", "tf-acc-test-destination-redshift-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "primary_key_fields", "id,created_at"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "table_name_prefix", "streamkap_updated"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "tasks_max", "10"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
