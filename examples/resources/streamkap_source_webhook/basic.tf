# Minimal Webhook source configuration
# This example creates a webhook endpoint to receive data

resource "streamkap_source_webhook" "example" {
  name = "my-webhook-source"

  # Topic to publish webhook data
  topic_include_list = "webhook-events"
}

# Note: After creation, the webhook URL and API key will be available
# in the Streamkap UI or via the API response.
