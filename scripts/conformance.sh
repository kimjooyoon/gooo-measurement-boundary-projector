#!/usr/bin/env bash
set -Eeuo pipefail

bin=${1:?projector binary is required}
root=${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}
output=${CONFORMANCE_OUTPUT:?CONFORMANCE_OUTPUT is required}
mkdir -p "$output"
"$bin" conformance \
	--source "$root/examples/measurement-boundary.gooo" \
	--corpus "$root/fixtures/corpus.json" \
	--out "$output"
jq -e '
  .schema == "gooo/measurement-boundary/conformance/v1" and
  .total == 11 and .selected == 11 and .executed == 11 and .reused == 0 and
  .closed == 3 and .unknown == 5 and .refuted == 3 and
  ([.tests[] | select(.expected != .observed)] | length) == 0
' "$output/conformance-summary.json" >/dev/null
