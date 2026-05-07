#!/usr/bin/env bash
# UAC: places calls in a loop, sends RTP (PCMU), mirrors SIP and media stats to HEP3 (UDP).
#
# Start a UAS first, e.g.: ./scripts/hep-uas-listen.sh
#
# HEP media (see docs/qos-reporting.md):
#   HEP_HOMER_LAKE=1 (default): Homer-Lake — HEP proto_type 5 with JSON RTCP SR body (Homer shape).
#   HEP_HOMER_LAKE=0: legacy modes via HEP_RAW_RTCP — 1 = binary RTCP on type 5; 0 = short JSON 0x22/0x24 (requires linked extension).
# The bundled uac_rtp_raw scenario holds RTP for ~65s so periodic reports can repeat.
#
# Environment:
#   HEP_ADDR            required: HEP3 collector host:port
#   HEP_CAPTURE_ID      capture node ID (default 3001; use a different ID than the UAS)
#   HEP_PASSWORD        optional HEP auth key
#   HEP_HOMER_LAKE      1/true = Homer-Lake JSON on HEP type 5 (default); 0 = use HEP_RAW_RTCP instead
#   HEP_RAW_RTCP        when HEP_HOMER_LAKE=0: 1 = binary type 5; 0 = short JSON 0x22/0x24 (requires extension; default 1)
#   UAS_ADDR            remote SIP host:port (default 127.0.0.1:9050)
#   CALL_RATE           calls per second (default 2)
#   MAX_CONCURRENT      max parallel calls (default 32)
#   GOSSIPPER           path to gossipper binary (otherwise: go run ./cmd/gossip)
#   USE_DIST            if 1 and dist/gossipper exists, use that binary
#
# OTLP logs: only when LOG_OTEL_ENDPOINT is set, e.g.:
#   export LOG_OTEL_ENDPOINT=http://127.0.0.1:4318
# Optional: LOG_OTEL_PROTO (default http), LOG_OTEL_INSECURE=1
#
# Example (Homer-Lake RTCP JSON on HEP type 5 — default):
#   export HEP_ADDR=collector.example.com:9081
#   ./scripts/hep-uas-listen.sh   # terminal 1
#   ./scripts/hep-uac-send.sh     # terminal 2
#
# Legacy binary RTCP on type 5 instead:
#   export HEP_HOMER_LAKE=0 HEP_RAW_RTCP=1
#   ./scripts/hep-uac-send.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

if [[ -z "${HEP_ADDR:-}" ]]; then
  echo "HEP_ADDR must be set (e.g. export HEP_ADDR=collector.example.com:9060)" >&2
  exit 1
fi

HEP_CAPTURE_ID="${HEP_CAPTURE_ID:-3001}"
UAS_ADDR="${UAS_ADDR:-127.0.0.1:9050}"
CALL_RATE="${CALL_RATE:-2}"
MAX_CONCURRENT="${MAX_CONCURRENT:-32}"

HEP_HOMER_LAKE="${HEP_HOMER_LAKE:-1}"
case "${HEP_HOMER_LAKE}" in
  1|true|TRUE|yes|YES) homer_lake_flag=true ;;
  *) homer_lake_flag=false ;;
esac

HEP_RAW_RTCP="${HEP_RAW_RTCP:-1}"
case "${HEP_RAW_RTCP}" in
  1|true|TRUE|yes|YES) hep_raw_rtcp_flag=true ;;
  *) hep_raw_rtcp_flag=false ;;
esac

if [[ -n "${GOSSIPPER:-}" ]]; then
  :
elif [[ "${USE_DIST:-0}" == "1" && -x "${ROOT}/dist/gossipper" ]]; then
  GOSSIPPER="${ROOT}/dist/gossipper"
else
  GOSSIPPER="go run ./cmd/gossip"
fi

args=(
  -sf "${ROOT}/testdata/scenarios/uac_rtp_raw.xml"
  -rsa "${UAS_ADDR}"
  -m 0
  -r "${CALL_RATE}"
  -l "${MAX_CONCURRENT}"
  -hep_addr "${HEP_ADDR}"
  -hep_capture_id "${HEP_CAPTURE_ID}"
  -send_media_report=true
)

if [[ "${homer_lake_flag}" == true ]]; then
  args+=( -hep_homer_lake_rtcp=true )
else
  if [[ "${hep_raw_rtcp_flag}" == true ]]; then
    args+=( -hep_raw_rtcp=true )
  else
    args+=( -hep_raw_rtcp=false )
  fi
fi

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

if [[ "${homer_lake_flag}" == true ]]; then
  echo "UAC -> SIP ${UAS_ADDR}  |  HEP3 UDP -> ${HEP_ADDR}  (capture_id=${HEP_CAPTURE_ID}, RTP + Homer-Lake HEP type 5 JSON RTCP)" >&2
elif [[ "${hep_raw_rtcp_flag}" == true ]]; then
  echo "UAC -> SIP ${UAS_ADDR}  |  HEP3 UDP -> ${HEP_ADDR}  (capture_id=${HEP_CAPTURE_ID}, RTP + HEP type 5 binary RTCP SR)" >&2
else
  echo "UAC -> SIP ${UAS_ADDR}  |  HEP3 UDP -> ${HEP_ADDR}  (capture_id=${HEP_CAPTURE_ID}, RTP + short JSON QoS 0x22/0x24 — requires extension)" >&2
fi
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
