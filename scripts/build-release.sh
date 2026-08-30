#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: scripts/build-release.sh <vMAJOR.MINOR.PATCH[-PRERELEASE]> [output-dir]" >&2
}

version="${1:-}"
output_dir="${2:-dist}"
if [[ ! "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]]; then
  usage
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ "$output_dir" != /* ]]; then
  output_dir="$repo_root/$output_dir"
fi
mkdir -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd -P)"
if [[ -z "$output_dir" || "$output_dir" == "/" || "$output_dir" == "$repo_root" ]]; then
  echo "refusing unsafe release output directory: $output_dir" >&2
  exit 1
fi

for command in go tar zip; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "required command is unavailable: $command" >&2
    exit 1
  fi
done

stage_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$stage_dir"
}
trap cleanup EXIT INT TERM

platforms=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
  "windows/arm64"
)

for platform in "${platforms[@]}"; do
  goos="${platform%/*}"
  goarch="${platform#*/}"
  asset_name="dygo_${version}_${goos}_${goarch}"
  build_dir="$stage_dir/build/$asset_name"
  binary="dygo"
  if [[ "$goos" == "windows" ]]; then
    binary="dygo.exe"
  fi

  mkdir -p "$build_dir"
  (
    cd "$repo_root"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags "-s -w -X github.com/hapyco/dygo/internal/cli.version=${version}" \
      -o "$build_dir/$binary" ./cmd/dygo
  )

  if [[ "$goos" == "windows" ]]; then
    (cd "$build_dir" && zip -q "$stage_dir/${asset_name}.zip" "$binary")
  else
    tar -C "$build_dir" -czf "$stage_dir/${asset_name}.tar.gz" "$binary"
  fi
done

cp "$repo_root/scripts/install.sh" "$stage_dir/install.sh"
cp "$repo_root/scripts/install.ps1" "$stage_dir/install.ps1"
cp "$repo_root/LICENSE" "$stage_dir/LICENSE"

(
  cd "$stage_dir"
  artifacts=(LICENSE install.ps1 install.sh dygo_*)
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${artifacts[@]}" >checksums.txt
  else
    shasum -a 256 "${artifacts[@]}" >checksums.txt
  fi
)

find "$output_dir" -maxdepth 1 -type f \( -name 'dygo_*' -o -name 'checksums.txt' -o -name 'install.sh' -o -name 'install.ps1' -o -name 'LICENSE' \) -delete
cp "$stage_dir"/dygo_* "$stage_dir/checksums.txt" "$stage_dir/install.sh" "$stage_dir/install.ps1" "$stage_dir/LICENSE" "$output_dir/"

echo "release assets ready: $output_dir"
