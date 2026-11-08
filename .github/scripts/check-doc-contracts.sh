#!/usr/bin/env bash
# check-doc-contracts.sh verifies that every public package in the SDK carries
# a self-contained doc.go contract and appears in the README import map.
#
# Usage:
#   check-doc-contracts.sh [import-path...]
#
# Without arguments it checks every public package. With arguments it checks
# only those import paths, so a mid-project task can verify its own packages.
#
# Environment:
#   CHECK_ROOT      Tree to check. Default: the repository root.
#   DOC_ONLY_ROOTS  Space-separated module roots with no importable package.
#                   They need a doc.go but no contract headings, and they must
#                   not appear as rows in the import map.
#
# Exit codes: 0 all checks pass; 1 findings; 2 usage error.
set -euo pipefail

ROOT=${CHECK_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}
if [ -z "${DOC_ONLY_ROOTS+x}" ]; then
  DOC_ONLY_ROOTS="github.com/fikrimohammad/go-servicekit/nats github.com/fikrimohammad/go-servicekit/rest"
fi
HEADINGS=("When to use" "Import" "Minimal example" "Lifecycle" "Common mistakes" "Test recipe")

errors=0
fail() {
  printf '::error::%s\n' "$*" >&2
  errors=$((errors + 1))
}

usage() {
  printf 'usage: %s [import-path...]\n' "$(basename "${BASH_SOURCE[0]}")" >&2
  exit 2
}

case "${1-}" in
  -h | --help)
    sed -n '2,18p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  -*)
    usage
    ;;
esac

if [ ! -f "$ROOT/README.md" ]; then
  fail "README.md is missing"
  exit 1
fi

# map_paths holds every import path listed as a row in the README import map.
mapfile -t map_paths < <(
  awk '
    /^## Import map/ { section = 1; next }
    /^## /          { section = 0 }
    section && /^\|/ {
      line = $0
      while (match(line, /`[^`]*`/)) {
        path = substr(line, RSTART + 1, RLENGTH - 2)
        if (path ~ /^github\.com\//) print path
        line = substr(line, RSTART + RLENGTH)
      }
    }
  ' "$ROOT/README.md" | sort -u
)

in_map() {
  local path
  for path in "${map_paths[@]}"; do
    [ "$path" = "$1" ] && return 0
  done
  return 1
}

is_doc_only() {
  local root
  for root in $DOC_ONLY_ROOTS; do
    [ "$root" = "$1" ] && return 0
  done
  return 1
}

# Discover every public package. Internal and test-only packages are skipped.
declare -A dir_of=() name_of=()
discovery_failed=0
while IFS= read -r mod; do
  if ! listing=$(cd "$mod" && GOWORK=off go list \
    -f '{{.ImportPath}}|{{.Dir}}|{{.Name}}|{{join .GoFiles ","}}' ./... 2>&1); then
    fail "${mod#"$ROOT"/}: go list failed: $listing"
    discovery_failed=1
    continue
  fi
  while IFS='|' read -r path dir name gofiles; do
    [ -n "$path" ] || continue
    case "$path" in
      */internal/*) continue ;;
    esac
    [ -n "$gofiles" ] || continue
    dir_of["$path"]=$dir
    name_of["$path"]=$name
  done <<<"$listing"
done < <(find "$ROOT" -maxdepth 2 -name go.mod -not -path '*/vendor/*' -printf '%h\n' | sort)

if [ "$discovery_failed" -ne 0 ]; then
  printf 'doc contracts: discovery failed; fix go list errors first\n' >&2
  exit 1
fi

if [ "${#dir_of[@]}" -eq 0 ]; then
  fail "no public packages found under $ROOT"
  exit 1
fi

# Every declared doc-only root must exist, stay out of the map, and be named in
# the README so an agent is told where the real packages are.
for root in $DOC_ONLY_ROOTS; do
  if [ -z "${dir_of[$root]+x}" ]; then
    fail "$root: declared doc-only root is not a public package in this tree"
    continue
  fi
  if in_map "$root"; then
    fail "$root: module root must not be an import-map row"
  fi
  if ! grep -Fq -- "$root" "$ROOT/README.md"; then
    fail "$root: README must name the root as not importable"
  fi
done

if [ "$#" -gt 0 ]; then
  selected=("$@")
else
  mapfile -t selected < <(printf '%s\n' "${!dir_of[@]}" | sort)
fi

for path in "${selected[@]}"; do
  if [ -z "${dir_of[$path]+x}" ]; then
    fail "$path: not a public package in this tree"
    continue
  fi
  dir=${dir_of[$path]}
  name=${name_of[$path]}
  doc=$dir/doc.go

  if [ ! -f "$doc" ]; then
    fail "$path: missing doc.go"
    continue
  fi
  if ! head -n 1 "$doc" | grep -q "^// Package $name "; then
    fail "$path: doc.go must start with // Package $name"
  fi
  comments=$(grep -rn "^// Package $name " "$dir" --include='*.go' || true)
  count=0
  if [ -n "$comments" ]; then
    count=$(printf '%s\n' "$comments" | wc -l)
  fi
  if [ "$count" -eq 0 ]; then
    fail "$path: missing package comment"
  elif [ "$count" -gt 1 ]; then
    fail "$path: duplicate package comment (want one, in doc.go)"
  fi

  if is_doc_only "$path"; then
    continue
  fi

  for heading in "${HEADINGS[@]}"; do
    if ! grep -q "^// # $heading$" "$doc"; then
      fail "$path: doc.go is missing the heading '# $heading'"
    fi
  done
  if ! in_map "$path"; then
    fail "$path: missing from the README import map"
  fi
done

# The map must not point at paths that are not public packages.
for path in "${map_paths[@]}"; do
  if [ -z "${dir_of[$path]+x}" ]; then
    fail "README import map lists $path, which is not a public package"
  fi
done

if [ "$errors" -ne 0 ]; then
  printf 'doc contracts: %d finding(s)\n' "$errors" >&2
  exit 1
fi
printf 'doc contracts: ok (%d package(s))\n' "${#selected[@]}"
