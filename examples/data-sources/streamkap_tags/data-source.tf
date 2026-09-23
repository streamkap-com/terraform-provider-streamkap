terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
      # Set version to the release matching this documentation.
      # Beta releases require an exact prerelease version; omission selects stable.
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

# List every tag in the tenant.
data "streamkap_tags" "all" {}

# Filter by name. All filters are optional; when several are set the backend ANDs them.
data "streamkap_tags" "by_name" {
  filter_name = "production"
}

# Filter by the entity types a tag applies to.
# Valid values: environment, general, sources, destinations, pipelines,
# transforms, topics, services, users, tenant.
data "streamkap_tags" "for_sources" {
  filter_type = ["sources", "destinations"]
}

variable "tag_ids" {
  type        = list(string)
  description = "Existing Streamkap tag IDs to look up."
}

# Resolve a known set of tag IDs to their full records in one call.
data "streamkap_tags" "by_ids" {
  filter_ids = var.tag_ids
}

output "all_tag_names" {
  value = [for t in data.streamkap_tags.all.tags : t.name]
}

output "matching_tag_ids" {
  value = [for tag in data.streamkap_tags.by_name.tags : tag.id]
}
