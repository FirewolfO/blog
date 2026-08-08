#!/usr/bin/env bash
set -euo pipefail
project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
pid_file="$project_dir/.runtime/garage.pid"
if [[ ! -f "$pid_file" ]]; then echo "Garage is not running"; exit 0; fi
pid="$(cat "$pid_file")"
if kill -0 "$pid" 2>/dev/null; then kill "$pid"; fi
rm -f "$pid_file"
echo "Garage stopped"
