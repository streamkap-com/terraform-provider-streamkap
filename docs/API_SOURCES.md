# API sources (provider development)

API-source schemas come from two backend artifacts per vendor, `app/sources/plugins/<vendor>/configuration.latest.json` and `form.schema.json`, which the backend generates from its vendor config models. How tfgen reads them is in `docs/CODE_GENERATOR.md`; the runtime hooks are in `docs/ARCHITECTURE.md`.

Regenerate only the API sources when the backend branch is not `main`, so no unrelated CDC change rides along. `make generate` regenerates every connector and refuses a non-`main` backend unless `ALLOW_NONMAIN=1`:

```bash
(
  set -euo pipefail   # a failed step stops the rest, so no docs or snapshots from a partial regen
  ALLOW_NONMAIN=1 bash scripts/codegen-preflight.sh   # stops on an unset or unreadable path; check the branch it prints
  for v in hubspot salesforce netsuite stripe zendesk google_analytics facebook_ads google_ads; do
    go run ./cmd/tfgen generate --backend-path="$STREAMKAP_BACKEND_PATH" --output=internal/generated --entity-type sources --connector "$v"
  done
  go generate main.go   # docs, after the schemas
  make snapshots        # then read the diff
)
```

A new vendor also needs its generated `Source<Vendor>APISource` in `apiSources` (`internal/resource/source/api_sources.go`), which registers the resource and keeps `streamkap_pipeline` off it, a one-line constructor there for its tests, a `TestSchemaBackwardsCompatibility_APISources` row, `examples/resources/streamkap_source_<vendor>/{basic,complete}.tf` and `import.sh`, and a lifecycle case in `internal/provider/api_source_vendors_local_test.go` if it adds an auth pattern.

Traps:
- A form `HIDE` rule hides a field only for the values it lists. Reading it as always-hidden once dropped HubSpot `token` and Google Ads `client_id`/`client_secret`; check each vendor's credentials survive a regen.
- The backend normalizes echoed values: `backfill_start` to a UTC datetime, Google Ads IDs lose dashes, Facebook Ads IDs lose `act_`, Salesforce `domain` gains `https://`, Zendesk `subdomain` is lowercased. `ConnectorConfigKeepsConfiguredForm` keeps the configured spelling only for the fields `normalizedFields` in `api_sources.go` lists: a field the backend starts normalizing needs a row there, or its apply fails as an inconsistent result. Lifecycle fixtures must normalize the same way.
- An update replaces the whole config. The backend keeps an absent secret and restores the fields a grant owns (Salesforce `org_id`, `environment`, and `domain` in oauth mode). The provider resends every secret the current auth mode shows, so a new location (Salesforce `domain`, NetSuite `account_id`) never pairs with a kept secret, which the backend refuses. It omits an unchanged secret the form hides (grant-filled, and the backend may have rotated it) and an unchanged `oauth_grant_id` (spent).
- A grant is authorized for one host (`oauth.host_field`, the Zendesk `subdomain`); changing it without a new grant fails at plan time.
- On an auth-mode switch the backend drops the old mode's secrets, and it refuses a real secret typed for another mode; the provider checks the latter at plan time for pasted modes. Removed gated fields are planned null, because Read refills a null-echoed secret from state.
- In OAuth mode the grant fills fields the form hides (HubSpot `token`, Salesforce `domain`, Stripe `token`, Google Ads `refresh_token`). They are kept while `oauth_grant_id` is unchanged on a source already in OAuth mode, and planned unknown when a new grant is sent; a switch into OAuth mode needs a new grant.
- `null` clears an optional secret; the backend refuses it for a required one. Config is `extra="forbid"`, so a vendor without OAuth must never be sent `oauth_grant_id`.
- Conditional rules belong in the backend artifacts (SHOW rules, `dependentRequired`), never in hand-written validators here. Matching `custom_<name>` resources to `custom_*` definitions cannot be expressed; the backend reports it as a 400 at apply.
- A refused save carries `fields: [{field, message}]` beside `detail`; the provider puts each message on its attribute.
- A source update sends a new resource's topic to every destination the source already sends to, and stops a removed one, outside `streamkap_topic_destination`.
