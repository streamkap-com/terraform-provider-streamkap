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

// TestAccDestinationBigqueryResource_PartitionAndClustering exercises the custom
// partition key and (single and multi-field) clustering keys. Field names mirror the
// sample sales schema (products: created_at / category_id, price; orders: order_date /
// customer_id, status, order_number) so the config reads like a real pipeline target.
// Clustering accepts a comma-separated list of up to 4 fields.
func TestAccDestinationBigqueryResource_PartitionAndClustering(t *testing.T) {
	var destinationBigqueryJson = os.Getenv("TF_VAR_destination_bigquery_json")
	var destinationBigqueryDataset = os.Getenv("TF_VAR_destination_bigquery_dataset")
	if destinationBigqueryJson == "" || destinationBigqueryDataset == "" {
		t.Skip("Skipping TestAccDestinationBigqueryResource_PartitionAndClustering: TF_VAR_destination_bigquery_json or TF_VAR_destination_bigquery_dataset not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Single partition field + single clustering field (sales.products shape)
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
	name                     = "tf-acc-test-bigquery-partclust"
	keyfile                  = var.destination_bigquery_json
	default_dataset          = var.destination_bigquery_dataset
	time_partitioning_type   = "DAY"
	custom_partition_field   = "created_at"
	custom_clustering_fields = "category_id"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "time_partitioning_type", "DAY"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "custom_partition_field", "created_at"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "custom_clustering_fields", "category_id"),
					resource.TestCheckResourceAttrSet("streamkap_destination_bigquery.test", "id"),
				),
			},
			// Update: different partition field + multi-field clustering, max 4 (sales.orders shape)
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
	name                     = "tf-acc-test-bigquery-partclust"
	keyfile                  = var.destination_bigquery_json
	default_dataset          = var.destination_bigquery_dataset
	time_partitioning_type   = "HOUR"
	custom_partition_field   = "order_date"
	custom_clustering_fields = "customer_id,status,order_number"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "time_partitioning_type", "HOUR"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "custom_partition_field", "order_date"),
					resource.TestCheckResourceAttr("streamkap_destination_bigquery.test", "custom_clustering_fields", "customer_id,status,order_number"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
