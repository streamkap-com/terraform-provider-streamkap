# A Kafka user is imported by its username, which is also its resource ID.
# The password is write-only and cannot be recovered, so set it in config and
# expect the first plan after import to show a password change.
terraform import streamkap_kafka_user.example analytics-consumer
