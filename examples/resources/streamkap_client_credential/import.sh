# A client credential is imported by its client ID.
# The secret cannot be recovered after creation — an imported credential has an
# empty `secret` in state, and re-reading will not populate it.
terraform import streamkap_client_credential.example 00000000-0000-0000-0000-000000000000
