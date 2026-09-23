#!/usr/bin/env bash
# Validate each example file on its own against a locally built provider.
# basic.tf, complete.tf and with_implementation.tf in one directory are
# alternatives that declare the same resources, so they cannot share a module.
set -euo pipefail

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

go build -o "$work/bin/terraform-provider-streamkap" .
cat > "$work/terraformrc" <<EOF
provider_installation {
  dev_overrides {
    "streamkap-com/streamkap" = "$work/bin"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$work/terraformrc"

count=0
failed=0
for example in examples/provider/*.tf examples/resources/*/*.tf examples/data-sources/*/*.tf; do
  count=$((count + 1))
  dir="$work/modules/$count"
  mkdir -p "$dir"
  cp "$example" "$dir/main.tf"
  # Registry snippets may omit the provider requirement that readers add.
  if ! grep -q required_providers "$example"; then
    cat > "$dir/providers.tf" <<'EOF'
terraform {
  required_providers {
    streamkap = { source = "streamkap-com/streamkap" }
  }
}
EOF
  fi
  if ! output="$(terraform -chdir="$dir" validate -no-color 2>&1)"; then
    failed=$((failed + 1))
    echo "validate-examples: $example failed:" >&2
    echo "$output" >&2
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "validate-examples: $failed of $count example files failed" >&2
  exit 1
fi
echo "validate-examples: $count example files valid"
