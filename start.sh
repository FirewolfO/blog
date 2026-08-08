#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
runtime_dir="$project_dir/.runtime"
mkdir -p "$runtime_dir/bin"
backend_pid=""
frontend_pid=""

listener_pids() { lsof -nP -t -iTCP:"$1" -sTCP:LISTEN 2>/dev/null | sort -u; }
release_port() {
  local port="$1" name="$2" attempt
  local -a pids=()
  mapfile -t pids < <(listener_pids "$port")
  if (( ${#pids[@]} == 0 )); then return; fi
  echo "$name port $port is occupied by PID(s) ${pids[*]}; stopping them."
  kill -TERM "${pids[@]}" 2>/dev/null || true
  for attempt in {1..20}; do sleep 0.25; mapfile -t pids < <(listener_pids "$port"); if (( ${#pids[@]} == 0 )); then return; fi; done
  kill -KILL "${pids[@]}" 2>/dev/null || true
}

if ! command -v lsof >/dev/null 2>&1; then echo "The Blog start script requires lsof." >&2; exit 1; fi
"$project_dir/storage/start.sh"
garage_pid="$(cat "$runtime_dir/garage.pid")"
release_port 8086 "Blog API"
release_port 5179 "Blog UI"

(cd "$project_dir/backend" && go build -o "$runtime_dir/bin/blog-server" ./cmd/server)
cleanup() {
  [[ -n "$backend_pid" ]] && kill "$backend_pid" 2>/dev/null || true
  [[ -n "$frontend_pid" ]] && kill "$frontend_pid" 2>/dev/null || true
  "$project_dir/storage/stop.sh" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

(cd "$project_dir/backend" && BLOG_PEOPLE_AUTHORIZE_URL="${BLOG_PEOPLE_AUTHORIZE_URL:-http://10.251.237.216:5177/oauth/authorize}" exec "$runtime_dir/bin/blog-server") > "$runtime_dir/blog-backend.log" 2>&1 &
backend_pid=$!
echo "$backend_pid" > "$runtime_dir/blog-backend.pid"
(cd "$project_dir/frontend" && exec npm run dev -- --host 0.0.0.0 --port 5179 --strictPort) > "$runtime_dir/blog-frontend.log" 2>&1 &
frontend_pid=$!
echo "$frontend_pid" > "$runtime_dir/blog-frontend.pid"

ready=false
for _ in {1..40}; do
  if curl --fail --silent http://127.0.0.1:8086/health >/dev/null && curl --fail --silent http://127.0.0.1:5179/ >/dev/null; then
    echo "Blog API is ready at http://127.0.0.1:8086"
    echo "Blog UI is ready at http://127.0.0.1:5179"
    ready=true
    break
  fi
  sleep 0.25
done
if [[ "$ready" != true ]]; then echo "Blog services did not become ready. Check $runtime_dir/*.log" >&2; exit 1; fi
wait -n "$backend_pid" "$frontend_pid"
