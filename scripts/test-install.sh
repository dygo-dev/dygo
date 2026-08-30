#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_root="$(mktemp -d)"
cleanup() {
  rm -rf "$test_root"
}
trap cleanup EXIT INT TERM

version="v9.8.7-test"
case "$(uname -s)" in
  Darwin) goos="darwin" ;;
  Linux) goos="linux" ;;
  *) echo "installer test is unsupported on this operating system" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) goarch="amd64" ;;
  arm64 | aarch64) goarch="arm64" ;;
  *) echo "installer test is unsupported on this architecture" >&2; exit 1 ;;
esac

release_dir="$test_root/releases/$version"
install_dir="$test_root/install"
asset="dygo_${version}_${goos}_${goarch}.tar.gz"
mkdir -p "$release_dir/payload"
cat >"$release_dir/payload/dygo" <<EOF
#!/usr/bin/env sh
if [ "\${1:-}" = "version" ]; then
  echo "dygo $version"
  exit 0
fi
exit 1
EOF
chmod 0755 "$release_dir/payload/dygo"
tar -C "$release_dir/payload" -czf "$release_dir/$asset" dygo
(
  cd "$release_dir"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$asset" >checksums.txt
  else
    shasum -a 256 "$asset" >checksums.txt
  fi
)

DYGO_VERSION="$version" \
DYGO_INSTALL_DIR="$install_dir" \
DYGO_DOWNLOAD_BASE_URL="file://$release_dir" \
  "$repo_root/scripts/install.sh" >/dev/null

test "$("$install_dir/dygo" version)" = "dygo $version"
printf '%s' 'corrupt' >>"$release_dir/$asset"
if DYGO_VERSION="$version" DYGO_INSTALL_DIR="$test_root/corrupt-install" DYGO_DOWNLOAD_BASE_URL="file://$release_dir" "$repo_root/scripts/install.sh" >/dev/null 2>&1; then
  echo "installer accepted a corrupted archive" >&2
  exit 1
fi
if DYGO_VERSION="not-a-version" "$repo_root/scripts/install.sh" >/dev/null 2>&1; then
  echo "installer accepted an invalid version" >&2
  exit 1
fi

echo "installer smoke passed"
