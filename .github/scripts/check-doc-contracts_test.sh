#!/usr/bin/env bash
# Tests for check-doc-contracts.sh. Each case builds a throwaway module tree and
# asserts the script's exit code and message.
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
script=$here/check-doc-contracts.sh
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

tree=$tmp/tree
passed=0
failed=0
ROOTS=

write_contract() {
  local pkg=$1 file=$2
  cat >"$file" <<EOF
// Package $pkg does a thing.
//
// # When to use
//
// Use it.
//
// # Import
//
//\timport "github.com/example/m/$pkg"
//
// # Minimal example
//
//\t$pkg.Do()
//
// # Lifecycle
//
// None.
//
// # Common mistakes
//
//   - None.
//
// # Test recipe
//
// Go test.
package $pkg
EOF
}

new_tree() {
  rm -rf "$tree"
  mkdir -p "$tree/m/thing"
  cat >"$tree/m/go.mod" <<'EOF'
module github.com/example/m

go 1.26.5
EOF
  write_contract thing "$tree/m/thing/doc.go"
  printf 'package thing\n' >"$tree/m/thing/thing.go"
  cat >"$tree/README.md" <<'EOF'
# Test tree

## Import map

| I need | Import | Notes |
| --- | --- | --- |
| A thing | `github.com/example/m/thing` | Fine |
EOF
}

run() {
  CHECK_ROOT=$tree DOC_ONLY_ROOTS=$ROOTS "$script" "$@" >"$tmp/out" 2>&1
}

expect_ok() {
  local name=$1
  shift
  if run "$@"; then
    passed=$((passed + 1))
  else
    failed=$((failed + 1))
    printf 'FAIL %s (expected 0)\n' "$name"
    sed 's/^/    /' "$tmp/out"
  fi
}

expect_fail() {
  local name=$1 needle=$2
  shift 2
  if run "$@"; then
    failed=$((failed + 1))
    printf 'FAIL %s (expected non-zero)\n' "$name"
    sed 's/^/    /' "$tmp/out"
  elif ! grep -Fq -- "$needle" "$tmp/out"; then
    failed=$((failed + 1))
    printf 'FAIL %s (missing %q)\n' "$name" "$needle"
    sed 's/^/    /' "$tmp/out"
  else
    passed=$((passed + 1))
  fi
}

new_tree
expect_ok "clean tree"
expect_ok "focused on one package" github.com/example/m/thing

new_tree
rm "$tree/m/thing/doc.go"
expect_fail "missing doc.go" "missing doc.go"

new_tree
sed -i '/# Lifecycle/d' "$tree/m/thing/doc.go"
expect_fail "missing heading" "missing the heading '# Lifecycle'"

new_tree
printf '// Package thing also does a thing.\npackage thing\n' >"$tree/m/thing/extra.go"
expect_fail "duplicate package comment" "duplicate package comment"

new_tree
sed -i '1s|^// Package thing|// Package other|' "$tree/m/thing/doc.go"
expect_fail "wrong first line" "must start with // Package thing"

new_tree
sed -i '/github.com\/example\/m\/thing/d' "$tree/README.md"
expect_fail "missing import-map row" "missing from the README import map"

new_tree
mkdir -p "$tree/m/internal/hidden"
printf 'package hidden\n' >"$tree/m/internal/hidden/hidden.go"
expect_ok "internal package ignored"

new_tree
mkdir -p "$tree/m/bad"
printf '// Package bad is bad.\npackage bad\n' >"$tree/m/bad/bad.go"
expect_fail "unfocused catches a contractless package" "missing doc.go"
expect_ok "focused skips other packages" github.com/example/m/thing

new_tree
mkdir -p "$tree/m/root"
printf '// Package root has no API.\npackage root\n' >"$tree/m/root/doc.go"
printf '\n`github.com/example/m/root` is not importable.\n' >>"$tree/README.md"
ROOTS=github.com/example/m/root
expect_ok "doc-only root outside the map"
ROOTS=

new_tree
mkdir -p "$tree/m/root"
printf '// Package root has no API.\npackage root\n' >"$tree/m/root/doc.go"
printf '\n| Root | `github.com/example/m/root` | No |\n' >>"$tree/README.md"
ROOTS=github.com/example/m/root
expect_fail "doc-only root in the map" "module root must not be an import-map row"
ROOTS=

new_tree
mkdir -p "$tree/m/root"
printf '// Package root has no API.\npackage root\n' >"$tree/m/root/doc.go"
ROOTS=github.com/example/m/root
expect_fail "doc-only root not named" "README must name the root as not importable"
ROOTS=

new_tree
printf '\n| Dead | `github.com/example/m/gone` | Gone |\n' >>"$tree/README.md"
expect_fail "dead import-map row" "which is not a public package"

new_tree
expect_fail "unknown import path" "not a public package" github.com/example/m/nope

printf '%d passed, %d failed\n' "$passed" "$failed"
[ "$failed" -eq 0 ]
