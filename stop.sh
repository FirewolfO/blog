#!/usr/bin/env bash
set -euo pipefail
project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for name in blog-backend blog-frontend; do
  pid_file="$project_dir/.runtime/$name.pid"
  if [[ -f "$pid_file" ]]; then
    pid="$(cat "$pid_file")"
    if kill -0 "$pid" 2>/dev/null; then kill "$pid" 2>/dev/null || true; fi
    rm -f "$pid_file"
  fi
done
"$project_dir/storage/stop.sh"
echo "Blog services stopped"
