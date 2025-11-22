#!/bin/bash
set -euo pipefail

CONFIG_DIR=${CONFIG_DIR:-/config}
CONFIG_FILE=${CONFIG_FILE:-${CONFIG_DIR}/config.json}
DEFAULT_CONFIG_TEMPLATE=/app/config/config.example.json
DEFAULT_CONFIG_TARGET="${CONFIG_DIR}/config.example.json"

mkdir -p "${CONFIG_DIR}"
mkdir -p "${CONFIG_DIR}/logs"
mkdir -p "${CONFIG_DIR}/temp"

# Always make the example config available in the mounted volume.
if [ ! -f "${DEFAULT_CONFIG_TARGET}" ]; then
  cp "${DEFAULT_CONFIG_TEMPLATE}" "${DEFAULT_CONFIG_TARGET}"
fi

if [[ "${1:-}" == "--configure" ]]; then
  shift
  cd "${CONFIG_DIR}"
  exec /usr/local/bin/configure-client.sh "$@"
fi

if [ ! -f "${CONFIG_FILE}" ]; then
  echo "❌ config.json not found at ${CONFIG_FILE}"
  echo ""
  echo "A copy of config.example.json has been placed in ${CONFIG_DIR}."
  echo "Update it with your server credentials and rename it to config.json, or rerun with --configure."
  exit 1
fi

cd "${CONFIG_DIR}"

exec capture-client "$@"
