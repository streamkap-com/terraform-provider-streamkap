package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceStripeAPIKey = os.Getenv("TF_VAR_source_stripe_api_key")
var sourceStripeSigningSecret = os.Getenv("TF_VAR_source_stripe_signing_secret")

func TestAccSourceStripeWebhookResource(t *testing.T) {
	if sourceStripeAPIKey == "" || sourceStripeSigningSecret == "" {
		t.Skip("Skipping TestAccSourceStripeWebhookResource: TF_VAR_source_stripe_api_key or TF_VAR_source_stripe_signing_secret not set")
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
variable "source_stripe_api_key" {
	type        = string
	sensitive   = true
	description = "Stripe secret API key"
}
variable "source_stripe_signing_secret" {
	type        = string
	sensitive   = true
	description = "Stripe webhook signing secret"
}
resource "streamkap_source_stripe_webhook" "test" {
	name                                              = %q
	camel_source_snapshot_stripe_api_key              = var.source_stripe_api_key
	camel_source_payload_router_stripe_signing_secret = var.source_stripe_signing_secret
	topic_include_list                                = "customer,payment_intent,charge,invoice"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_stripe_webhook.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_source_stripe_webhook.test", "topic_include_list", "customer,payment_intent,charge,invoice"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_source_stripe_webhook.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "source_stripe_api_key" {
	type        = string
	sensitive   = true
	description = "Stripe secret API key"
}
variable "source_stripe_signing_secret" {
	type        = string
	sensitive   = true
	description = "Stripe webhook signing secret"
}
resource "streamkap_source_stripe_webhook" "test" {
	name                                              = %q
	camel_source_snapshot_stripe_api_key              = var.source_stripe_api_key
	camel_source_payload_router_stripe_signing_secret = var.source_stripe_signing_secret
	topic_include_list                                = "customer,payment_intent,charge"
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_stripe_webhook.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_source_stripe_webhook.test", "topic_include_list", "customer,payment_intent,charge"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
