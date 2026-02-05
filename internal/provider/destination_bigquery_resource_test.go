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
	description = "BigQuery JSON credentials file content"
}
variable "destination_bigquery_dataset" {
	type        = string
	description = "BigQuery dataset name (table_name_prefix)"
}
resource "streamkap_destination_bigquery" "test" {
	name                            = "tf-acc-test-destination-bigquery"
	bigquery_json                   = var.destination_bigquery_json
	table_name_prefix               = var.destination_bigquery_dataset
	bigquery_region                 = "us-central1"
	bigquery_time_based_partition   = false
	tasks_max                       = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "name", "tf-acc-test-destination-bigquery"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "bigquery_json", destinationBigqueryJson),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "table_name_prefix", destinationBigqueryDataset),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "bigquery_region", "us-central1"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "bigquery_time_based_partition", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_bigquery.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "connector", "bigquery"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_bigquery.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_bigquery_json" {
	type        = string
	sensitive   = true
	description = "BigQuery JSON credentials file content"
}
variable "destination_bigquery_dataset" {
	type        = string
	description = "BigQuery dataset name (table_name_prefix)"
}
resource "streamkap_destination_bigquery" "test" {
	name                            = "tf-acc-test-destination-bigquery-updated"
	bigquery_json                   = var.destination_bigquery_json
	table_name_prefix               = var.destination_bigquery_dataset
	bigquery_region                 = "us-east1"
	bigquery_time_based_partition   = true
	tasks_max                       = 3
	custom_bigquery_partition_field = "_streamkap_ts"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "name", "tf-acc-test-destination-bigquery-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "bigquery_region", "us-east1"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "bigquery_time_based_partition", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "tasks_max", "3"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "custom_bigquery_partition_field", "_streamkap_ts"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
