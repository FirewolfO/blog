#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime_dir="$project_dir/.runtime"
binary="$runtime_dir/bin/garage"
version="v2.3.0"
download_url="https://garagehq.deuxfleurs.fr/_releases/$version/x86_64-unknown-linux-musl/garage"

mkdir -p "$runtime_dir/bin"
if [[ -x "$binary" ]] && "$binary" --version 2>/dev/null | grep -q "2.3.0"; then
  echo "Garage $version is already installed at $binary"
  exit 0
fi

tmp_file="$(mktemp "$runtime_dir/bin/garage.XXXXXX")"
trap 'rm -f "$tmp_file"' EXIT
echo "Downloading Garage $version for linux/amd64..."
curl --fail --location --retry 3 --output "$tmp_file" "$download_url"
chmod 0755 "$tmp_file"
mv "$tmp_file" "$binary"
trap - EXIT
"$binary" --version
