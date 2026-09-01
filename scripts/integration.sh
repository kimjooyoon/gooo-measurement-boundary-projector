#!/usr/bin/env bash
set -Eeuo pipefail

bin=${1:?projector binary is required}
root=${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}
output=${INTEGRATION_OUTPUT:?INTEGRATION_OUTPUT is required}
mkdir -p "$output"
"$bin" compile \
	--source "$root/examples/measurement-boundary.gooo" \
	--out "$output/compile"
bash "$output/compile/generated/collect.sh" \
	"$root/fixtures/cases/closed-single-authority-receipt.json" \
	"$output/generated-collection"
"$bin" evaluate \
	--ir "$output/compile/semantic-ir.json" \
	--collection "$output/generated-collection/collection.json" \
	--out "$output/evaluation.json"
"$bin" report \
	--evaluation "$output/evaluation.json" \
	--out "$output/human-report.md"
jq -e '
  .decision == "CLOSED" and .fail_closed == false and
  ([.metrics[] | select(.state != "CLOSED" or .value == null)] | length) == 0 and
  .aggregate_policy == "FORBID_UNSCOPED_SCALAR"
' "$output/evaluation.json" >/dev/null
jq -e '
  .collector.generated == true and .collector.measured_once == true and
  .collector.repository_writes == 0 and .collector.apply_authority == 0 and
  .collector.commit_authority == 0 and .collector.merge_authority == 0 and
  .collector.tag_authority == 0 and .collector.release_authority == 0
' "$output/generated-collection/collection.json" >/dev/null
jq -e '
  ([.metrics[] | .receipt_digests | length] | all(. == 1)) and
  ([.metrics[] | .consumer_artifacts | length] | all(. >= 1))
' "$output/evaluation.json" >/dev/null
python3 - "$output/evaluation.json" "$output/generated-collection/collection.json" "$output/integration-result.json" <<'PY'
import json
import sys

evaluation_path, collection_path, destination = sys.argv[1:]
with open(evaluation_path, encoding="utf-8") as source:
    evaluation = json.load(source)
with open(collection_path, encoding="utf-8") as source:
    collection = json.load(source)
receipts = [metric["receipt_digests"][0] for metric in evaluation["metrics"]]
consumers = []
for metric in evaluation["metrics"]:
    metric_id = metric["measurement_id"]
    consumers.extend(item for item in collection["consumers"] if item["metric_id"] == metric_id)
result = {
    "schema": "gooo/measurement-boundary/integration/v1",
    "generated_collector_ran": True,
    "measured_once_per_metric": collection["collector"]["measured_once"],
    "consumer_receipts_exact": all(item["receipt_digest"] in receipts for item in consumers),
    "receipt_digests": receipts,
    "evaluation_digest": evaluation["collection_digest"],
}
with open(destination, "w", encoding="utf-8") as output:
    json.dump(result, output, indent=2, sort_keys=True)
    output.write("\n")
PY
