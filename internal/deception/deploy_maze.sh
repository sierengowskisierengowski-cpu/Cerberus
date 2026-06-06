#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_MAZE_DIR="${SCRIPT_DIR}/shadow_layer"
MAZE_DIR="${CERBERUS_MAZE_DIR:-$DEFAULT_MAZE_DIR}"
TMPFS_SIZE="${CERBERUS_MAZE_SIZE:-64M}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "[!] Cerberus deception deployment requires sudo/root privileges."
  exit 1
fi

mkdir -p "$MAZE_DIR"

echo "[+] Deploying Cerberus deception maze at: $MAZE_DIR"

if mountpoint -q "$MAZE_DIR"; then
  echo "[i] Existing tmpfs mount detected at $MAZE_DIR; reusing it."
else
  mount -t tmpfs -o "size=${TMPFS_SIZE}" tmpfs "$MAZE_DIR"
fi

mkdir -p \
  "$MAZE_DIR/Documents" \
  "$MAZE_DIR/Downloads" \
  "$MAZE_DIR/System_Configs"

cat > "$MAZE_DIR/Documents/secrets.txt" <<'EOF'
This is a Cerberus deception artifact for research and testing.
Do not store real credentials in this location.
EOF

ln -sf /proc/self/fd/1 "$MAZE_DIR/System_Configs/output_loop"

echo "[+] Deception maze deployed successfully."
echo "[i] To override the location, set CERBERUS_MAZE_DIR."
echo "[i] To override tmpfs size, set CERBERUS_MAZE_SIZE."
