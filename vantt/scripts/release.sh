#!/usr/bin/env bash
# Fork release on the host that runs it (since 2026-11-11 GitHub Actions of this fork
# are disabled by GitHub, and the images serve one x86_64 host anyway):
#   1. catchup from origin/main in a throwaway worktree: upstream tag + patches, each
#      patch's test_command, tag v<tag>-vantt.N;
#   2. push the release branch and tag;
#   3. build ghcr.io/vantt/multica-{backend,web}:<version> (linux/amd64) and push them,
#      or only load them into the local Docker with --no-push.
#
#   bash vantt/scripts/release.sh                 # tests need DATABASE_URL (a scratch Postgres)
#   bash vantt/scripts/release.sh --skip-tests    # patches already tested on this exact series
#   bash vantt/scripts/release.sh --no-push       # images only in the local Docker
#   bash vantt/scripts/release.sh --ref v0.7.3    # another upstream tag
#
# Pushing to GHCR needs a token with write:packages:
#   gh auth refresh -h github.com -s write:packages
#   gh auth token | docker login ghcr.io -u <user> --password-stdin
# Deploy afterwards: fgOPC multica-ops references/install-upgrade.md.
set -euo pipefail

push=1; catchup_args=()
while [ $# -gt 0 ]; do
  case "$1" in
    --no-push) push=0 ;;
    --skip-tests) catchup_args+=(--skip-tests) ;;
    --ref) catchup_args+=(--ref "$2"); shift ;;
    *) echo "unknown option $1" >&2; exit 2 ;;
  esac
  shift
done
if [[ " ${catchup_args[*]-} " != *" --skip-tests "* && -z "${DATABASE_URL:-}" ]]; then
  echo "patch tests need DATABASE_URL (a scratch Postgres, never the live one), or pass --skip-tests" >&2
  exit 2
fi

repo="$(git rev-parse --show-toplevel)"
git -C "$repo" fetch --quiet --tags origin
wt="$(mktemp -d "${TMPDIR:-/tmp}/vantt-release.XXXXXX")"
cleanup() { git -C "$repo" worktree remove --force "$wt" 2>/dev/null || true; }
trap cleanup EXIT
git -C "$repo" worktree add --quiet --detach "$wt" origin/main
cd "$wt"
git config user.name "$(git -C "$repo" config user.name)"
git config user.email "$(git -C "$repo" config user.email)"
# Go writes large temporaries; /tmp is often a small tmpfs.
export GOTMPDIR="${GOTMPDIR:-$HOME/.cache/gotmp}"; mkdir -p "$GOTMPDIR"

out="$(node vantt/scripts/catchup.mjs "${catchup_args[@]}" | tail -n 1)"
version="$(node -e 'const o=JSON.parse(process.argv[1]);process.stdout.write(o.version)' "$out")"
branch="$(node -e 'const o=JSON.parse(process.argv[1]);process.stdout.write(o.branch)' "$out")"
existing="$(node -e 'const o=JSON.parse(process.argv[1]);process.stdout.write(String(o.existing))' "$out")"
echo "release $version (branch $branch, existing=$existing)"

git push --quiet --force origin "$branch"
git push --quiet origin "$version"

git checkout --quiet "$version"
commit="$(git rev-parse HEAD)"
mode=(--load); [ "$push" = 1 ] && mode=(--push)
for m in backend:Dockerfile web:Dockerfile.web; do
  name="${m%%:*}"; file="${m#*:}"
  echo "build multica-$name:$version"
  docker buildx build --pull --platform linux/amd64 -f "$file" \
    -t "ghcr.io/vantt/multica-$name:$version" \
    --label org.opencontainers.image.source=https://github.com/vantt/multica \
    --build-arg VERSION="$version" --build-arg COMMIT="$commit" \
    --build-arg REMOTE_API_URL=http://backend:8080 \
    --build-arg NEXT_PUBLIC_APP_VERSION="$version" \
    "${mode[@]}" .
done
echo "done: $version ($([ "$push" = 1 ] && echo pushed to GHCR || echo local Docker only))"
