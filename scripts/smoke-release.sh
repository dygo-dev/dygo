#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
dist_dir="${2:-dist}"
if [[ -z "$version" ]]; then
  echo "usage: scripts/smoke-release.sh <version> [dist-dir]" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ "$dist_dir" != /* ]]; then
  dist_dir="$repo_root/$dist_dir"
fi

case "$(uname -s)" in
  Darwin) goos="darwin" ;;
  Linux) goos="linux" ;;
  *) echo "release smoke is unsupported on this operating system" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) goarch="amd64" ;;
  arm64 | aarch64) goarch="arm64" ;;
  *) echo "release smoke is unsupported on this architecture" >&2; exit 1 ;;
esac

asset="dygo_${version}_${goos}_${goarch}.tar.gz"
if [[ ! -f "$dist_dir/$asset" ]]; then
  echo "release asset is missing: $dist_dir/$asset" >&2
  exit 1
fi

smoke_root="$(mktemp -d)"
cleanup() {
  rm -rf "$smoke_root"
}
trap cleanup EXIT INT TERM

tar -xzf "$dist_dir/$asset" -C "$smoke_root"
test "$("$smoke_root/dygo" version)" = "dygo $version"
(
  cd "$smoke_root"
  "$smoke_root/dygo" new smoke-app --skip-tidy
)
test -f "$smoke_root/smoke-app/.dygo/apps/core/app.yml"
test -f "$smoke_root/smoke-app/.dygo/apps/studio/app.yml"
test -f "$smoke_root/smoke-app/.dygo/apps/studio/pages/home/home.page.yml"
test -f "$smoke_root/smoke-app/.dygo/apps/studio/access/home.page.access.yml"
test -f "$smoke_root/smoke-app/.dygo/apps/studio/ui/dist/index.html"
test ! -e "$smoke_root/smoke-app/.dygo/apps/core/entities/user/fixtures.yml"
test "$(cd "$smoke_root/smoke-app" && "$smoke_root/dygo" hook sync --dry-run)" = "runner: cmd/dygo/main.go (unchanged)"
(
  cd "$smoke_root/smoke-app"
  git init -q
  ! git check-ignore .dygo/apps/core/app.yml
  git check-ignore .dygo/apps/studio/ui/dist/index.html >/dev/null
)

echo "release smoke passed: $asset"
