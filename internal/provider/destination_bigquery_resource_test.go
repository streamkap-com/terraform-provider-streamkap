package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationBigqueryResource(t *testing.T) {
	var destinationBigqueryJson = os.Getenv("TF_VAR_destination_bigquery_json")
	var destinationBigqueryDataset = os.Getenv("TF_VAR_destination_bigquery_dataset")
	if destinationBigqueryJson == "" || destinationBigqueryDataset == "" {
		t.Skip("Skipping TestAccDestinationBigqueryResource: TF_VAR_destination_bigquery_json or TF_VAR_destination_bigquery_dataset not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_bigquery_json" {
	type        = string
	sensitive   = true
	description = "BigQuery service-account JSON key file contents"
}
variable "destination_bigquery_dataset" {
	type        = string
	description = "BigQuery destination dataset name"
}
resource "streamkap_destination_bigquery" "test" {
	name                   = "tf-acc-test-destination-bigquery"
	keyfile                = var.destination_bigquery_json
	default_dataset        = var.destination_bigquery_dataset
	time_partitioning_type = "DAY"
	auto_create_tables     = true
	tasks_max              = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "name", "tf-acc-test-destination-bigquery"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "default_dataset", destinationBigqueryDataset),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "time_partitioning_type", "DAY"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "auto_create_tables", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_bigquery.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "connector", "bigquery"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_destination_bigquery.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status", "keyfile"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_bigquery_json" {
	type        = string
	sensitive   = true
	description = "BigQuery service-account JSON key file contents"
}
variable "destination_bigquery_dataset" {
	type        = string
	description = "BigQuery destination dataset name"
}
resource "streamkap_destination_bigquery" "test" {
	name                     = "tf-acc-test-destination-bigquery-updated"
	keyfile                  = var.destination_bigquery_json
	default_dataset          = var.destination_bigquery_dataset
	time_partitioning_type   = "HOUR"
	tasks_max                = 3
	custom_partition_field   = "_streamkap_ts"
	custom_clustering_fields = "id"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "name", "tf-acc-test-destination-bigquery-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "time_partitioning_type", "HOUR"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "tasks_max", "3"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "custom_partition_field", "_streamkap_ts"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "custom_clustering_fields", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
