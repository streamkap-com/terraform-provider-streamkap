# Minimal Kafka Direct source configuration
# This example reads data directly from external Kafka topics

resource "streamkap_source_kafkadirect" "example" {
  name = "my-kafka-source"

  # Topic configuration
  topic_prefix       = "myapp_"
  topic_include_list = "myapp_orders,myapp_customers"
}
