---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "streamkap_transform Data Source - terraform-provider-streamkap"
subcategory: ""
description: |-
  Tranform data source
---

# streamkap_transform (Data Source)

Tranform data source



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) Transform identifier

### Read-Only

- `name` (String) Transform name
- `start_time` (String) Start time
- `topic_ids` (List of String) List of topic identifiers
- `topic_map` (Attributes List) List of topic object, with id and name for each topic (see [below for nested schema](#nestedatt--topic_map))

<a id="nestedatt--topic_map"></a>
### Nested Schema for `topic_map`

Read-Only:

- `id` (String)
- `name` (String)