---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "streamkap Provider"
subcategory: ""
description: |-
  
---

# streamkap Provider



## Example Usage

```terraform
terraform {
  required_providers {
    streamkap = {
      source = "github.com/streamkap-com/streamkap"
    }
  }
}

provider "streamkap" {
  host      = "https://api.streamkap.com"
  client_id = "client_id"
  secret    = "secret"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `client_id` (String)
- `host` (String)
- `secret` (String, Sensitive)
