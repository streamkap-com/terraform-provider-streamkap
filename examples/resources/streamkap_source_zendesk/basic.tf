# Zendesk signs in only through Connect with Zendesk, which runs in a browser.
# Finish Connect in the Streamkap UI, or run
#   streamkap sources start-source-oauth-connect zendesk
# open the returned authorize_url, then run
#   streamkap sources poll-source-oauth-grant zendesk --state <state>
# and pass the returned grant before it expires.
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
