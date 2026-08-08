#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime_dir="$project_dir/.runtime"
binary="$runtime_dir/bin/garage"
config_file="$runtime_dir/garage.toml"
pid_file="$runtime_dir/garage.pid"
log_file="$runtime_dir/garage.log"
access_key="${BLOG_STORAGE_ACCESS_KEY:-GK0123456789abcdef0123456789abcdef}"
secret_key="${BLOG_STORAGE_SECRET_KEY:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}"
bucket="${BLOG_STORAGE_BUCKET:-blog-media}"

"$project_dir/storage/install.sh"
mkdir -p "$runtime_dir/garage/meta" "$runtime_dir/garage/data"

if [[ ! -f "$config_file" ]]; then
  umask 077
  rpc_secret="$(openssl rand -hex 32)"
  admin_token="$(openssl rand -base64 32)"
  metrics_token="$(openssl rand -base64 32)"
  cat > "$config_file" <<EOF
metadata_dir = "$runtime_dir/garage/meta"
data_dir = "$runtime_dir/garage/data"
db_engine = "sqlite"
replication_factor = 1
rpc_bind_addr = "127.0.0.1:3901"
rpc_public_addr = "127.0.0.1:3901"
rpc_secret = "$rpc_secret"

[s3_api]
s3_region = "garage"
api_bind_addr = "127.0.0.1:3900"
root_domain = ".s3.garage.localhost"

[s3_web]
bind_addr = "127.0.0.1:3902"
root_domain = ".web.garage.localhost"
index = "index.html"

[admin]
api_bind_addr = "127.0.0.1:3903"
admin_token = "$admin_token"
metrics_token = "$metrics_token"
EOF
fi

if [[ -f "$pid_file" ]] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
  echo "Garage is already running with PID $(cat "$pid_file")"
  exit 0
fi
if command -v lsof >/dev/null 2>&1 && lsof -nP -t -iTCP:3900 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "Port 3900 is already occupied; refusing to replace an unknown service." >&2
  exit 1
fi

GARAGE_CONFIG_FILE="$config_file" \
GARAGE_DEFAULT_ACCESS_KEY="$access_key" \
GARAGE_DEFAULT_SECRET_KEY="$secret_key" \
GARAGE_DEFAULT_BUCKET="$bucket" \
nohup "$binary" server --single-node --default-bucket > "$log_file" 2>&1 &
garage_pid=$!
echo "$garage_pid" > "$pid_file"

for _ in {1..40}; do
  if GARAGE_CONFIG_FILE="$config_file" "$binary" status >/dev/null 2>&1; then
    echo "Garage is ready at http://127.0.0.1:3900 (PID $garage_pid)"
    exit 0
  fi
  if ! kill -0 "$garage_pid" 2>/dev/null; then
    echo "Garage exited during startup:" >&2
    tail -40 "$log_file" >&2
    exit 1
  fi
  sleep 0.25
done
echo "Garage did not become ready; see $log_file" >&2
exit 1
