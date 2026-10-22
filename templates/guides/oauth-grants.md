---
page_title: "Get an OAuth grant for an API source"
subcategory: ""
description: |-
  Run Connect with the Streamkap CLI to get the single-use oauth_grant_id an API source resource needs to sign in through OAuth.
---

# Get an OAuth grant for an API source

Some API sources sign in through the vendor's consent page (Connect) instead of a pasted key:

- `streamkap_source_zendesk`: always.
- `streamkap_source_hubspot` and `streamkap_source_stripe`: with `auth_mode = "oauth"`.
- `streamkap_source_salesforce`: with `auth_mode = "oauth"`.
- `streamkap_source_google_ads`: with `auth_mode = "oauth"`, using your own OAuth client.

Consent runs in a browser, which Terraform cannot drive. You run Connect once with the [Streamkap CLI](https://www.npmjs.com/package/@streamkap/tools), and it returns an `oauth_grant_id`. Terraform sends that grant when it creates the source, and the backend swaps it for the credentials Connect stored.

A grant is single-use and expires 10 minutes after Connect finishes. Get it right before `terraform apply`.

## 1. Sign in to the CLI

Use the same client credentials as the provider: a grant only works for the tenant and the user that started Connect. The CLI reads the secret from `STREAMKAP_CLIENT_SECRET`, the provider from `STREAMKAP_SECRET`.

```shell
export STREAMKAP_CLIENT_ID="..."
export STREAMKAP_CLIENT_SECRET="..."
export STREAMKAP_SECRET="$STREAMKAP_CLIENT_SECRET"
streamkap doctor
```

## 2. Start Connect

Run the command for your vendor. Each prints JSON with an `authorize_url` and a `state`.

```shell
# Zendesk: the subdomain, as in acme.zendesk.com
streamkap sources start-source-oauth-connect zendesk --environment acme --json

# HubSpot and Stripe
streamkap sources start-source-oauth-connect hubspot --json
streamkap sources start-source-oauth-connect stripe --json

# Salesforce: production (the default) or sandbox
streamkap sources start-source-oauth-connect salesforce --environment production --json

# Google Ads: your own OAuth client, read from a file so the secret stays out of shell history
# client.json: {"client_id": "...", "client_secret": "..."}
streamkap sources start-source-oauth-connect google_ads --body client.json --json
```

## 3. Approve in the browser

Open the `authorize_url` and approve as a user who can read everything you sync. For Zendesk, that is a Zendesk admin; an agent's grant misses the tickets that agent cannot see.

## 4. Collect the grant

Poll with the `state` from step 2. It returns `{"status": "pending"}` until the browser step finishes, then `{"oauth_grant_id": "..."}`, or `{"error": "..."}` if consent failed. A state answers once, so keep the grant it returns.

```shell
streamkap sources poll-source-oauth-grant zendesk --state '<state>' --json
```

## 5. Apply within 10 minutes

Pass the grant as a sensitive variable, so it stays out of your configuration files:

```terraform
variable "zendesk_oauth_grant_id" {
  type      = string
  sensitive = true
}

resource "streamkap_source_zendesk" "support" {
  name           = "zendesk-support"
  subdomain      = "acme"
  oauth_grant_id = var.zendesk_oauth_grant_id
  resources      = ["tickets", "users", "organizations"]
}
```

```shell
export TF_VAR_zendesk_oauth_grant_id='<oauth_grant_id>'
terraform apply
```

The grant is kept in state and not sent again while it is unchanged. Later plans leave the source alone, so you do not need a fresh grant for every apply.

## Reconnect a source

If the vendor revokes access, you change a Zendesk subdomain, or you switch `auth_mode` to OAuth, run steps 2 to 4 again. Then set the new grant and apply. A grant that differs from the one in state reconnects the source.

## Import instead

You can also create the source in the Streamkap UI, where Connect runs in the browser, and then import it:

```shell
terraform import streamkap_source_zendesk.support <source-id>
```

Leave `oauth_grant_id` unset on an imported source. The backend keeps the credentials from the UI's Connect.

The same import recovers a create whose response was lost: the source exists, so the retried create fails, and you import it instead. That configuration still holds the grant the lost create spent. On the first apply after the import, the provider records that grant without sending it, warns that it treated it as already used, and leaves the source's connection as it is. To reconnect later, get a new grant and set it.

## MCP

The Streamkap MCP server has the same two steps, `streamkap_start_source_oauth_connect` and `streamkap_poll_source_oauth_grant`, if an agent prepares the grant for you.
