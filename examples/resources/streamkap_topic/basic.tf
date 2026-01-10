# Minimal topic configuration
# Manages partition count for a Streamkap topic

resource "streamkap_topic" "example" {
  topic_id        = "source_abc123.public.users"
  partition_count = 10
}
