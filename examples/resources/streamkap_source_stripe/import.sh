# After import, an oauth_grant_id still in the configuration is taken as the one
# already spent: it is recorded, not sent, and the source keeps its connection.
# Set a new grant only to reconnect.
terraform import streamkap_source_stripe.example 00000000000000000000000000
