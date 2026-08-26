#!/usr/bin/env bash
set -e
PORT=18090
BASE="http://localhost:${PORT}"
echo "=== Killing any existing bbc-pipeline-tui ==="
pkill -f "bbc-pipeline-tui" 2>/dev/null || true
sleep 0.5
echo "=== Starting server ==="
cd "$(dirname "$0")"
./bbc-pipeline-tui --webapp "localhost:${PORT}" > /tmp/bblog.txt 2>&1 &
SERVER_PID=$!
sleep 1
cleanup() { echo "=== Shutting down ==="; kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true; }
trap cleanup EXIT

echo "=== 1. Health ==="
curl -sf "${BASE}/api/health" | python3 -m json.tool

echo "=== 2. List pipelines ==="
curl -sf "${BASE}/api/projects/0/pipelines?pagelen=1" | python3 -c "
import sys,json
d=json.load(sys.stdin)
v=d['values']
print(f'{len(v)} pipelines, first: #{v[0][\"build_number\"]} {v[0][\"state\"][\"name\"]}')"

echo "=== 3. Pipeline detail ==="
UUID=$(curl -sf "${BASE}/api/projects/0/pipelines?pagelen=1" | python3 -c "import sys,json;print(json.load(sys.stdin)['values'][0]['uuid'].strip('{}'))")
curl -sf "${BASE}/api/projects/0/pipelines/${UUID}" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(f'#{d[\"pipeline\"][\"build_number\"]} steps={len(d[\"steps\"])}')"

echo "=== 4. Variables ==="
curl -sf "${BASE}/api/projects/0/variables" | python3 -c "import sys,json;print(len(json.load(sys.stdin)['variables']),'variables')"

echo "=== 5. Running pipelines ==="
curl -sf "${BASE}/api/running-pipelines" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(f\"{len(d['running'])} running pipeline(s), {len(d['errors'])} repo error(s)\")"

echo "=== 6. Frontend ==="
curl -sfI "${BASE}/" | head -1

echo "=== ALL TESTS PASSED ==="