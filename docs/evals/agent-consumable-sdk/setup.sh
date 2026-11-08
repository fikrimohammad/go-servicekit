#!/usr/bin/env bash
# setup.sh copies a sample fixture to a temporary directory, points every SDK
# module at this working tree, downloads the fixture's dependencies, and prints
# the working directory.
#
# Usage: setup.sh <sample>
#
# The sample is the fixture directory name and the sample-<name>.md prompt next
# to this script. Requires Go 1.26.5 and network access for go mod download.
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
root=$(cd "$here/../../.." && pwd)
sample=${1-}

if [ -z "$sample" ] || [ ! -d "$here/fixtures/$sample" ]; then
  printf 'usage: %s <sample>\n' "$(basename "${BASH_SOURCE[0]}")" >&2
  printf 'samples:\n' >&2
  for dir in "$here"/fixtures/*/; do
    [ -d "$dir" ] && printf '  %s\n' "$(basename "$dir")" >&2
  done
  exit 2
fi

work=$(mktemp -d)
cp -R "$here/fixtures/$sample/." "$work/"

# Replace every SDK module, including transitive siblings that the fixture does
# not require directly. A dependency's own replace directives are ignored, so
# the fixture needs all of them.
while IFS= read -r modfile; do
  module=$(awk '$1 == "module" { print $2; exit }' "$modfile")
  rel=${modfile#"$root"/}
  dir=$(dirname "$rel")
  (cd "$work" && GOWORK=off go mod edit -replace "$module=$root/$dir")
done < <(find "$root" -maxdepth 2 -name go.mod -not -path '*/vendor/*' | sort)

(cd "$work" && GOWORK=off go mod download)
printf '%s\n' "$work"
