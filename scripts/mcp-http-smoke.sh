#!/usr/bin/env bash
# mcp-http-smoke.sh — start wolt-mcp over Streamable HTTP and call wolt_top.
#
# Usage:
#   ./scripts/mcp-http-smoke.sh
#   ./scripts/mcp-http-smoke.sh 3 44.8176 20.4633
#
# Environment:
#   WOLT_MCP_BIN   path to wolt-mcp (default ./bin/wolt-mcp)
#   WOLT_MCP_URL   if set, do not start a server; call this base URL
#                  (example: http://127.0.0.1:8080/mcp)

set -euo pipefail

readonly ROOT="$(cd "$(dirname "$0")/.." && pwd)"
readonly BIN="${WOLT_MCP_BIN:-${ROOT}/bin/wolt-mcp}"
readonly N="${1:-3}"
readonly LAT="${2:-${WOLT_SMOKE_LAT:-44.8176}}"
readonly LON="${3:-${WOLT_SMOKE_LON:-20.4633}}"

accept='application/json, text/event-stream'
content='application/json'

session_id_from_headers() {
  awk 'BEGIN { IGNORECASE = 1 }
    tolower($0) ~ /^mcp-session-id:/ {
      sub(/\r$/, "")
      sub(/^[^:]+:[ \t]*/, "")
      print
      exit
    }' "$1"
}

json_from_body() {
  # Streamable HTTP may reply with SSE (`event: message` / `data: {...}`)
  # or a raw JSON object. Print the last JSON payload.
  if grep -q '^data:' "$1"; then
    awk '/^data:/{ sub(/^data:[ \t]*/, ""); json = $0 } END { print json }' "$1"
  else
    cat "$1"
  fi
}

http_status() {
  awk 'NR==1 { print $2 }' "$1"
}

post() {
  local out_headers="$1"
  local out_body="$2"
  shift 2
  curl -sS -D "${out_headers}" -o "${out_body}" \
    -H "Content-Type: ${content}" \
    -H "Accept: ${accept}" \
    "$@"
}

WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/wolt-mcp-http.XXXXXX")"
cleanup() {
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  rm -rf "${WORKDIR}"
}
trap cleanup EXIT

if [[ -n "${WOLT_MCP_URL:-}" ]]; then
  URL="${WOLT_MCP_URL}"
else
  if [[ ! -x "${BIN}" ]]; then
    echo "building ${BIN}" >&2
    mkdir -p "$(dirname "${BIN}")"
    (cd "${ROOT}" && go build -o "${BIN}" ./cmd/wolt-mcp)
  fi
  PORT="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"
  URL="http://127.0.0.1:${PORT}/mcp"
  echo "starting ${BIN} on ${URL}" >&2
  "${BIN}" --listen "127.0.0.1:${PORT}" >"${WORKDIR}/server.log" 2>&1 &
  SERVER_PID=$!
  ready=0
  for _ in $(seq 1 50); do
    if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
      break
    fi
    if curl -sS -o /dev/null --connect-timeout 0.2 "http://127.0.0.1:${PORT}/" 2>/dev/null; then
      ready=1
      break
    fi
    sleep 0.1
  done
  if [[ "${ready}" != 1 ]] || ! kill -0 "${SERVER_PID}" 2>/dev/null; then
    echo "wolt-mcp exited during startup:" >&2
    cat "${WORKDIR}/server.log" >&2
    exit 1
  fi
fi

echo "initialize ${URL}" >&2
post "${WORKDIR}/init.hdr" "${WORKDIR}/init.body" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"mcp-http-smoke","version":"1"}}}' \
  "${URL}"
if [[ "$(http_status "${WORKDIR}/init.hdr")" != "200" ]]; then
  echo "initialize failed: HTTP $(http_status "${WORKDIR}/init.hdr")" >&2
  cat "${WORKDIR}/init.hdr" >&2
  cat "${WORKDIR}/init.body" >&2
  echo >&2
  exit 1
fi

SID="$(session_id_from_headers "${WORKDIR}/init.hdr")"
if [[ -z "${SID}" ]]; then
  echo "missing Mcp-Session-Id" >&2
  echo "----- headers -----" >&2
  cat "${WORKDIR}/init.hdr" >&2
  echo "----- body -----" >&2
  cat "${WORKDIR}/init.body" >&2
  exit 1
fi
echo "session ${SID}" >&2

echo "notifications/initialized" >&2
# Notifications are allowed to return 202 with an empty body; do not --fail.
curl -sS -D "${WORKDIR}/ack.hdr" -o "${WORKDIR}/ack.body" \
  -H "Content-Type: ${content}" \
  -H "Accept: ${accept}" \
  -H "Mcp-Session-Id: ${SID}" \
  -H "Mcp-Protocol-Version: 2025-11-25" \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  "${URL}" >/dev/null
ACK_CODE="$(http_status "${WORKDIR}/ack.hdr")"
if [[ "${ACK_CODE}" != "200" && "${ACK_CODE}" != "202" ]]; then
  echo "initialized failed: HTTP ${ACK_CODE}" >&2
  cat "${WORKDIR}/ack.hdr" >&2
  cat "${WORKDIR}/ack.body" >&2
  echo >&2
  exit 1
fi

echo "tools/call wolt_top n=${N} lat=${LAT} lon=${LON}" >&2
post "${WORKDIR}/call.hdr" "${WORKDIR}/call.body" \
  -H "Mcp-Session-Id: ${SID}" \
  -H "Mcp-Protocol-Version: 2025-11-25" \
  -d "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"wolt_top\",\"arguments\":{\"n\":${N},\"lat\":${LAT},\"lon\":${LON}}}}" \
  "${URL}"
if [[ "$(http_status "${WORKDIR}/call.hdr")" != "200" ]]; then
  echo "tools/call failed: HTTP $(http_status "${WORKDIR}/call.hdr")" >&2
  cat "${WORKDIR}/call.hdr" >&2
  cat "${WORKDIR}/call.body" >&2
  echo >&2
  exit 1
fi

json_from_body "${WORKDIR}/call.body"
echo
