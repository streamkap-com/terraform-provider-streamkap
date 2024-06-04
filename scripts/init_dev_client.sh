#!/bin/bash
export STREAMKAP_CLIENT_ID=
export STREAMKAP_SECRET=

# Determine the directory where the script is located
cwd="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$cwd/init_resources_infor.sh"