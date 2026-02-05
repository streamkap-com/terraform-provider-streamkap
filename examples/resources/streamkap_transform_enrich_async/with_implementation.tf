# Enrich Async transform with implementation_json
# Asynchronous enrichment allows calling external APIs during transformation

terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

resource "streamkap_transform_enrich_async" "example" {
  name = "async-enrich-orders"

  transforms_language             = "JavaScript"
  transforms_input_topic_pattern  = "my-source\\.public\\.orders"
  transforms_output_topic_pattern = "enriched-$0"

  # Manage async transform implementation with timeout and capacity settings
  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = <<-JS
      async function _streamkap_transform(inputObj) {
        // Example: Enrich with data from an external API
        const response = await fetch(
          'https://api.example.com/customers/' + inputObj.customer_id
        );
        const customer = await response.json();

        inputObj.customer_name = customer.name;
        inputObj.customer_email = customer.email;

        return inputObj;
      }
    JS
    # Async-specific settings
    async_timeout_ms = 5000  # 5 second timeout for external calls
    async_capacity   = 20    # Process up to 20 records concurrently
  })
}

output "transform_id" {
  value = streamkap_transform_enrich_async.example.id
}
