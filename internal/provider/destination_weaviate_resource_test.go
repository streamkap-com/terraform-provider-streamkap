package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationWeaviateConnectionURL = os.Getenv("TF_VAR_destination_weaviate_connection_url")
var destinationWeaviateGrpcURL = os.Getenv("TF_VAR_destination_weaviate_grpc_url")

func TestAccDestinationWeaviateResource(t *testing.T) {
	if destinationWeaviateConnectionURL == "" || destinationWeaviateGrpcURL == "" {
		t.Skip("Skipping TestAccDestinationWeaviateResource: TF_VAR_destination_weaviate_connection_url or TF_VAR_destination_weaviate_grpc_url not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_weaviate_connection_url" {
	type        = string
	description = "Weaviate connection URL"
}
variable "destination_weaviate_grpc_url" {
	type        = string
	description = "Weaviate gRPC URL"
}
variable "destination_weaviate_api_key" {
	type        = string
	sensitive   = true
	description = "Weaviate API key"
	default     = ""
}
resource "streamkap_destination_weaviate" "test" {
	name                     = "tf-acc-test-destination-weaviate"
	weaviate_connection_url  = var.destination_weaviate_connection_url
	weaviate_grpc_url        = var.destination_weaviate_grpc_url
	weaviate_auth_scheme     = "API_KEY"
	weaviate_api_key         = var.destination_weaviate_api_key
	document_id_strategy     = "None"
	weaviate_vectorizer      = "none"
	delete_enabled           = true
	batch_size               = 100
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "name", "tf-acc-test-destination-weaviate"),
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "weaviate_connection_url", destinationWeaviateConnectionURL),
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "weaviate_auth_scheme", "API_KEY"),
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "document_id_strategy", "None"),
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "delete_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "batch_size", "100"),
					resource.TestCheckResourceAttrSet("streamkap_destination_weaviate.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "connector", "weaviate"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_destination_weaviate.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_weaviate_connection_url" {
	type        = string
	description = "Weaviate connection URL"
}
variable "destination_weaviate_grpc_url" {
	type        = string
	description = "Weaviate gRPC URL"
}
variable "destination_weaviate_api_key" {
	type        = string
	sensitive   = true
	description = "Weaviate API key"
	default     = ""
}
resource "streamkap_destination_weaviate" "test" {
	name                     = "tf-acc-test-destination-weaviate-updated"
	weaviate_connection_url  = var.destination_weaviate_connection_url
	weaviate_grpc_url        = var.destination_weaviate_grpc_url
	weaviate_auth_scheme     = "API_KEY"
	weaviate_api_key         = var.destination_weaviate_api_key
	document_id_strategy     = "None"
	weaviate_vectorizer      = "none"
	delete_enabled           = false
	batch_size               = 200
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "name", "tf-acc-test-destination-weaviate-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "delete_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_weaviate.test", "batch_size", "200"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
