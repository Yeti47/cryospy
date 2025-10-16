#!/usr/bin/env bash
set -euo pipefail

if [[ ${DEBUG_ENTRYPOINT:-0} == 1 ]]; then
  set -x
fi

# Allow overriding HOME for custom data directory, otherwise rely on image default
HOME_DIR="${HOME:-/home/cryospy}"
DATA_ROOT="${HOME_DIR}/cryospy"
LOG_DIR="${DATA_ROOT}/logs"

mkdir -p "${LOG_DIR}"

# Provide helpful symlink when HOME is redirected via CRYOSPY_DATA_ROOT
if [[ -n "${CRYOSPY_DATA_DIR:-}" && "${CRYOSPY_DATA_DIR}" != "${DATA_ROOT}" ]]; then
  mkdir -p "${CRYOSPY_DATA_DIR}"
  DATA_ROOT="${CRYOSPY_DATA_DIR}"
  LOG_DIR="${DATA_ROOT}/logs"
  mkdir -p "${LOG_DIR}"

  TARGET_LINK="${HOME_DIR}/cryospy"
  if [[ -e "${TARGET_LINK}" && ! -L "${TARGET_LINK}" ]]; then
    rm -rf "${TARGET_LINK}"
  fi
  ln -sfn "${DATA_ROOT}" "${TARGET_LINK}"
fi

  # Determine config path after resolving data root
  CONFIG_PATH="${CRYOSPY_CONFIG_PATH:-${DATA_ROOT}/config.json}"

# Ensure config directory exists and generate default config when missing
CONFIG_DIR="$(dirname "${CONFIG_PATH}")"
mkdir -p "${CONFIG_DIR}"

if [[ ! -f "${CONFIG_PATH}" ]]; then
  cat > "${CONFIG_PATH}" <<EOF
{
  "web_addr": "0.0.0.0",
  "web_port": 8080,
  "capture_port": 8081,
  "database_path": "${DATA_ROOT}/cryospy.db",
  "log_path": "${DATA_ROOT}/logs",
  "log_level": "info",
  "streaming_settings": {
    "cache": {
      "enabled": true,
      "max_size_bytes": 104857600
    },
    "look_ahead": 10,
    "width": 854,
    "height": 480,
    "video_bitrate": "1000k",
    "video_codec": "libx264",
    "frame_rate": 25
  }
}
EOF
  chmod 600 "${CONFIG_PATH}"
fi

cd /opt/cryospy

if [[ $# -gt 0 ]]; then
  exec "$@"
fi

trap 'echo "[entrypoint] Received termination signal, shutting down"; kill 0' SIGINT SIGTERM

/usr/local/bin/capture-server &
CAPTURE_PID=$!

echo "[entrypoint] capture-server started with PID ${CAPTURE_PID}"

/usr/local/bin/dashboard &
DASHBOARD_PID=$!

echo "[entrypoint] dashboard started with PID ${DASHBOARD_PID}"

wait -n "${CAPTURE_PID}" "${DASHBOARD_PID}"
EXIT_CODE=$?

echo "[entrypoint] One of the services exited with status ${EXIT_CODE}, stopping the rest"
kill 0 || true
wait || true

exit ${EXIT_CODE}
