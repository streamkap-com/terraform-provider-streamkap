# Zendesk signs in only through Connect, which Terraform cannot run. Get a
# grant with the Streamkap CLI (the "Get an OAuth grant" guide has the steps):
#   streamkap sources start-source-oauth-connect zendesk --environment acme
# open the returned authorize_url and approve as a Zendesk admin, then run
#   streamkap sources poll-source-oauth-grant zendesk --state <state>
# and apply with the returned oauth_grant_id within 10 minutes.
variable "zendesk_oauth_grant_id" {
  type      = string
  sensitive = true
}

resource "streamkap_source_zendesk" "example" {
  name           = "zendesk-support"
  subdomain      = "acme"
  oauth_grant_id = var.zendesk_oauth_grant_id
  resources      = ["tickets", "users", "organizations"]
}
