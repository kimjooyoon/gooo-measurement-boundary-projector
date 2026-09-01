#!/usr/bin/env bash
set -Eeuo pipefail

bin=${1:?projector binary is required}
root=${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}
output=${V2_CONFORMANCE_OUTPUT:?V2_CONFORMANCE_OUTPUT is required}
mkdir -p "$output"
"$bin" v2-conformance \
	--source "$root/examples/measurement-boundary-v2.gooo" \
	--corpus "$root/fixtures/v2/corpus.json" \
	--out "$output"
jq -e '
  .schema == "gooo/measurement-boundary/conformance/v2" and
  .total == 12 and .closed == 4 and .unknown == 4 and .refuted == 4 and
  ([.tests[] | select(.expected != .observed)] | length) == 0 and
  ([.tests[]] | length) == 12 and
  ([.controlled_pairs[] | select(.before == null or .after == null)] | length) == 0 and
  ([.optional_observations[] | select(.decision != "UNKNOWN" or .required_gate != false or .immutable_input != true)] | length) == 0
' "$output/conformance-summary.json" >/dev/null
