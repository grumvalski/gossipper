#!/usr/bin/env bash
# UAS with SDP (PCMU): accepts incoming calls until Ctrl+C; mirrors SIP to HEP3 (UDP).
# Pairs with testdata/scenarios/uac_rtp_raw.xml (long media pause before BYE). Mandatory
# recv BYE without XML timeout uses CLI -recv_bye_timeout_ms (default 90s) as a floor.
#
# Environment:
#   HEP_ADDR          required: HEP3 collector host:port
#   HEP_CAPTURE_ID    capture node ID (default 2001)
#   HEP_PASSWORD      optional HEP auth key
#   BIND_IP           local bind IP (default 0.0.0.0)
#   BIND_PORT         SIP UDP port (default 9050; 5060 is often busy)
#   GOSSIPPER         path to gossipper binary (otherwise: go run ./cmd/gossip from source)
#   USE_DIST            if 1 and dist/gossipper exists, use it (may be an outdated build)
#
# OTLP logs: only when you set LOG_OTEL_ENDPOINT (same as gossipper -log_otel_endpoint), e.g.:
#   export LOG_OTEL_ENDPOINT=http://127.0.0.1:4318
# Optional: LOG_OTEL_PROTO (default http), LOG_OTEL_INSECURE=1 -> -log_otel_insecure=true
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

if [[ -z "${HEP_ADDR:-}" ]]; then
  echo "HEP_ADDR must be set (e.g. export HEP_ADDR=collector.example.com:9060)" >&2
  exit 1
fi

HEP_CAPTURE_ID="${HEP_CAPTURE_ID:-2001}"
BIND_IP="${BIND_IP:-0.0.0.0}"
BIND_PORT="${BIND_PORT:-9050}"

if [[ -n "${GOSSIPPER:-}" ]]; then
  :
elif [[ "${USE_DIST:-0}" == "1" && -x "${ROOT}/dist/gossipper" ]]; then
  GOSSIPPER="${ROOT}/dist/gossipper"
else
  GOSSIPPER="go run ./cmd/gossip"
fi

args=(
  -sf "${ROOT}/testdata/scenarios/uas_pcap.xml"
  -i "${BIND_IP}"
  -p "${BIND_PORT}"
  -m 0
  -r 1
  -l 2048
  -hep_addr "${HEP_ADDR}"
  -hep_capture_id "${HEP_CAPTURE_ID}"
)

if [[ -n "${HEP_PASSWORD:-}" ]]; then
  args+=( -hep_password "${HEP_PASSWORD}" )
fi

if [[ -n "${LOG_OTEL_ENDPOINT:-}" ]]; then
  LOG_OTEL_PROTO="${LOG_OTEL_PROTO:-http}"
  args+=( -log_otel_endpoint "${LOG_OTEL_ENDPOINT}" -log_otel_proto "${LOG_OTEL_PROTO}" )
  case "${LOG_OTEL_INSECURE:-0}" in
    1|true|TRUE|yes|YES) args+=( -log_otel_insecure=true ) ;;
  esac
fi

echo "UAS: SIP UDP ${BIND_IP}:${BIND_PORT}  |  HEP3 UDP -> ${HEP_ADDR}  (capture_id=${HEP_CAPTURE_ID})" >&2
if [[ -n "${LOG_OTEL_ENDPOINT:-}" ]]; then
  echo "OTLP logs -> ${LOG_OTEL_ENDPOINT} (${LOG_OTEL_PROTO:-http})" >&2
fi
echo "Stop with Ctrl+C" >&2

if [[ "${GOSSIPPER}" == *"go run"* ]]; then
  # shellcheck disable=SC2086
  exec ${GOSSIPPER} "${args[@]}"
else
  exec "${GOSSIPPER}" "${args[@]}"
fi
