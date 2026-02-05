# Fan Out transform with implementation_json
# Route records to multiple output topics based on content

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

resource "streamkap_transform_fan_out" "example" {
  name = "fanout-by-region"

  transforms_language             = "JavaScript"
  transforms_input_topic_pattern  = "my-source\\.public\\.orders"
  transforms_output_topic_pattern = "routed-$0"

  # Manage fan out implementation via Terraform
  # The topic_transform function determines output topic(s) for each record
  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = <<-JS
      function _streamkap_transform(inputObj) {
        // Optionally modify the record before routing
        return inputObj;
      }
    JS
    topic_transform = <<-JS
      function _streamkap_topic_transform(inputObj, inputTopic) {
        // Route to different topics based on region
        var region = inputObj.region || 'unknown';

        // Return an array of topics to fan out to multiple destinations
        switch(region) {
          case 'US':
            return ['orders-us', 'orders-all'];
          case 'EU':
            return ['orders-eu', 'orders-all'];
          case 'APAC':
            return ['orders-apac', 'orders-all'];
          default:
            return ['orders-other'];
        }
      }
    JS
  })
}

output "transform_id" {
  value = streamkap_transform_fan_out.example.id
}
