#!/bin/bash
# Load PEM key files for Snowflake tests
#
# Usage: source scripts/load-pem-keys.sh
#
# This script exports TF_VAR_destination_snowflake_private_key and
# TF_VAR_destination_snowflake_private_key_nocrypt from PEM files
# in the project root.
#
# NOTE: Run this AFTER loading .env (godotenv handles other vars)
#       but BEFORE running tests that need Snowflake.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Load the encrypted private key
if [ -f "$PROJECT_ROOT/TF_VAR_destination_snowflake_private_key.pem" ]; then
    export TF_VAR_destination_snowflake_private_key="$(cat "$PROJECT_ROOT/TF_VAR_destination_snowflake_private_key.pem")"
    echo "Loaded: TF_VAR_destination_snowflake_private_key"
else
    echo "Warning: TF_VAR_destination_snowflake_private_key.pem not found"
fi

# Load the unencrypted private key
if [ -f "$PROJECT_ROOT/TF_VAR_destination_snowflake_private_key_nocrypt.pem" ]; then
    export TF_VAR_destination_snowflake_private_key_nocrypt="$(cat "$PROJECT_ROOT/TF_VAR_destination_snowflake_private_key_nocrypt.pem")"
    echo "Loaded: TF_VAR_destination_snowflake_private_key_nocrypt"
else
    echo "Warning: TF_VAR_destination_snowflake_private_key_nocrypt.pem not found"
fi
