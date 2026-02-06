# Map/Filter transform with implementation_json
# This example shows how to manage transform implementation code via Terraform

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

resource "streamkap_transform_map_filter" "example" {
  name = "transform-with-code"

  transforms_language                   = "JavaScript"
  transforms_input_topic_pattern        = "my-source\\.public\\.orders"
  transforms_output_topic_pattern       = "transformed-$0"
  transforms_input_serialization_format = "Avro"

  # Manage transform implementation code via Terraform
  # This allows version control of your transformation logic
  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = <<-JS
      function _streamkap_transform(inputObj) {
        // Add a timestamp field
        inputObj.processed_at = new Date().toISOString();

        // Filter out records with null values
        if (inputObj.customer_id === null) {
          return null;
        }

        return inputObj;
      }
    JS
  })
}

output "transform_id" {
  value = streamkap_transform_map_filter.example.id
}
