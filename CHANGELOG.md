## 0.1.0 (Unreleased)

FEATURES:

BUG FIXES:

* **Provider error handling**: API errors that return a non-JSON body (gateway HTML pages, WAF blocks, proxy 5xx) now surface as `unexpected <status> <status text> from <method> <url>: <body snippet>` instead of the cryptic `invalid character '<' looking for beginning of value`. The HTTP status, URL, and a truncated body snippet are included directly in the Terraform error, so failures like 504 gateway timeouts on long-running operations are diagnosable without re-running with `TF_LOG=DEBUG`. Resolves ENG-2460.

* **Resource read**: When a resource (source, destination, pipeline) is deleted out-of-band — for example via the Streamkap UI, ops cleanup, or a prior failed `terraform destroy` — the `Read` handler now removes it from Terraform state instead of returning `... does not exist`. `terraform refresh` / `terraform plan` recover automatically and propose recreating the resource; previously the only workaround was `terraform state rm`. Resolves ENG-2461.
