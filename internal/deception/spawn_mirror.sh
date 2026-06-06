#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_SCRIPT="${SCRIPT_DIR}/deploy_maze.sh"

chmod +x "$DEPLOY_SCRIPT"
exec sudo "$DEPLOY_SCRIPT"
