package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationHttpsinkUrl = os.Getenv("TF_VAR_destination_httpsink_url")

func TestAccDestinationHttpsinkResource(t *testing.T) {
	if destinationHttpsinkUrl == "" {
		t.Skip("Skipping TestAccDestinationHttpsinkResource: TF_VAR_destination_httpsink_url not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_httpsink_url" {
	type        = string
	description = "HTTP Sink destination URL"
}
resource "streamkap_destination_httpsink" "test" {
	name                     = "tf-acc-test-destination-httpsink"
	http_url                 = var.destination_httpsink_url
	http_authorization_type  = "none"
	http_headers_content_type = "application/json"
	batching_enabled         = false
	max_retries              = 1
	retry_backoff_ms         = 3000
	http_timeout             = 30
	decimal_format           = "NUMERIC"
	errors_tolerance         = "none"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "name", "tf-acc-test-destination-httpsink"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "http_url", destinationHttpsinkUrl),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "http_authorization_type", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "http_headers_content_type", "application/json"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "batching_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "max_retries", "1"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "retry_backoff_ms", "3000"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "http_timeout", "30"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "decimal_format", "NUMERIC"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "errors_tolerance", "none"),
					resource.TestCheckResourceAttrSet("streamkap_destination_httpsink.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "connector", "httpsink"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_httpsink.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_httpsink_url" {
	type        = string
	description = "HTTP Sink destination URL"
}
resource "streamkap_destination_httpsink" "test" {
	name                     = "tf-acc-test-destination-httpsink-updated"
	http_url                 = var.destination_httpsink_url
	http_authorization_type  = "none"
	http_headers_content_type = "application/json"
	batching_enabled         = true
	batch_max_size           = 1000
	batch_buffering_enabled  = true
	batch_max_time_ms        = 5000
	batch_prefix             = "["
	batch_suffix             = "]"
	batch_separator          = ","
	max_retries              = 3
	retry_backoff_ms         = 5000
	http_timeout             = 60
	decimal_format           = "BASE64"
	errors_tolerance         = "all"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "name", "tf-acc-test-destination-httpsink-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "batching_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "batch_max_size", "1000"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "batch_buffering_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "batch_max_time_ms", "5000"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "batch_prefix", "["),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "batch_suffix", "]"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "batch_separator", ","),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "max_retries", "3"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "retry_backoff_ms", "5000"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "http_timeout", "60"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "decimal_format", "BASE64"),
					resource.TestCheckResourceAttr("streamkap_destination_httpsink.test", "errors_tolerance", "all"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
