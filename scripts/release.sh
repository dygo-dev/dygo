#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage: scripts/release.sh <vMAJOR.MINOR.PATCH[-PRERELEASE]> [--push]

Runs release checks, builds and smokes all assets, and creates an annotated tag.
Pass --push to push the tag and trigger the GitHub release workflow.
EOF
}

version="${1:-}"
push_tag=false
if [[ "${2:-}" == "--push" ]]; then
  push_tag=true
elif [[ -n "${2:-}" ]]; then
  usage
  exit 1
fi
if [[ ! "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]]; then
  usage
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

for command in git go npm; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "required command is unavailable: $command" >&2
    exit 1
  fi
done

if [[ -n "$(git status --porcelain)" ]]; then
  echo "release requires a clean worktree" >&2
  exit 1
fi
if [[ "$(git branch --show-current)" != "main" ]]; then
  echo "release tags must be created from main" >&2
  exit 1
fi

git fetch origin main --tags
if [[ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]]; then
  echo "local main must match origin/main before release" >&2
  exit 1
fi
if git rev-parse -q --verify "refs/tags/$version" >/dev/null; then
  echo "tag already exists locally: $version" >&2
  exit 1
fi
if git ls-remote --exit-code --tags origin "refs/tags/$version" >/dev/null 2>&1; then
  echo "tag already exists on origin: $version" >&2
  exit 1
fi

(
  cd apps/studio/ui
  npm ci
  npm test
  npm run build:embed
)
go generate ./internal/frameworkapp
git diff --exit-code -- internal/frameworkapp/bundled/core
go test ./...
go vet ./...
scripts/test-install.sh
scripts/build-release.sh "$version" dist
scripts/smoke-release.sh "$version" dist

if [[ -n "$(git status --porcelain)" ]]; then
  echo "release checks changed tracked files; inspect the worktree before tagging" >&2
  exit 1
fi

git tag -a "$version" -m "dygo $version"
echo "created release tag: $version"
if [[ "$push_tag" == true ]]; then
  git push origin "refs/tags/$version"
  echo "pushed $version; GitHub Actions will publish the release"
else
  echo "review dist/, then run: git push origin refs/tags/$version"
fi
