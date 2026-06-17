package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceShopifyAccessToken = os.Getenv("TF_VAR_source_shopify_access_token")
var sourceShopifyHMACSecret = os.Getenv("TF_VAR_source_shopify_hmac_secret")

func TestAccSourceShopifyWebhookResource(t *testing.T) {
	if sourceShopifyAccessToken == "" || sourceShopifyHMACSecret == "" {
		t.Skip("Skipping TestAccSourceShopifyWebhookResource: TF_VAR_source_shopify_access_token or TF_VAR_source_shopify_hmac_secret not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_shopify_access_token" {
	type        = string
	sensitive   = true
	description = "Shopify Admin API access token"
}
variable "source_shopify_hmac_secret" {
	type        = string
	sensitive   = true
	description = "Shopify webhook HMAC signing secret"
}
resource "streamkap_source_shopify_webhook" "test" {
	name                                            = "tf-acc-test-source-shopify-webhook"
	camel_source_snapshot_shopify_store_url         = "https://tf-acc-test.myshopify.com"
	camel_source_snapshot_shopify_access_token      = var.source_shopify_access_token
	camel_source_payload_router_shopify_hmac_secret = var.source_shopify_hmac_secret
	topic_include_list                              = "orders,products,customers"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_shopify_webhook.test", "name", "tf-acc-test-source-shopify-webhook"),
					resource.TestCheckResourceAttr("streamkap_source_shopify_webhook.test", "camel_source_snapshot_shopify_store_url", "https://tf-acc-test.myshopify.com"),
					resource.TestCheckResourceAttr("streamkap_source_shopify_webhook.test", "topic_include_list", "orders,products,customers"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_source_shopify_webhook.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_shopify_access_token" {
	type        = string
	sensitive   = true
	description = "Shopify Admin API access token"
}
variable "source_shopify_hmac_secret" {
	type        = string
	sensitive   = true
	description = "Shopify webhook HMAC signing secret"
}
resource "streamkap_source_shopify_webhook" "test" {
	name                                            = "tf-acc-test-source-shopify-webhook-updated"
	camel_source_snapshot_shopify_store_url         = "https://tf-acc-test.myshopify.com"
	camel_source_snapshot_shopify_access_token      = var.source_shopify_access_token
	camel_source_payload_router_shopify_hmac_secret = var.source_shopify_hmac_secret
	topic_include_list                              = "orders,products"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_shopify_webhook.test", "name", "tf-acc-test-source-shopify-webhook-updated"),
					resource.TestCheckResourceAttr("streamkap_source_shopify_webhook.test", "topic_include_list", "orders,products"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
