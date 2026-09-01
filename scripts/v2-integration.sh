#!/usr/bin/env bash
set -Eeuo pipefail

bin=${1:?projector binary is required}
root=${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}
output=${V2_INTEGRATION_OUTPUT:?V2_INTEGRATION_OUTPUT is required}
mkdir -p "$output"
"$bin" v2-compile \
	--source "$root/examples/measurement-boundary-v2.gooo" \
	--out "$output/compile"
bash "$output/compile/generated/collect.sh" \
	"$root/fixtures/v2/cases/closed-same-scope-matched-pair.json" \
	"$output/generated-collection"
"$bin" v2-evaluate \
	--ir "$output/compile/semantic-ir.json" \
	--collection "$output/generated-collection/collection.json" \
	--out "$output/evaluation.json"
"$bin" v2-report \
	--evaluation "$output/evaluation.json" \
	--out "$output/human-report.md"
jq -e '
  .decision == "CLOSED" and .fail_closed == false and
  .aggregate_policy == "FORBID_UNSCOPED_SCALAR" and
  .closed_count == 2 and .unknown_count == 0 and .refuted_count == 0 and
  ([.metrics[] | select(.state != "CLOSED")] | length) == 0 and
  ([.metrics[] | select(.improvement.state != "CLOSED" or .improvement.before == null or .improvement.after == null or .improvement.delta == null)] | length) == 0 and
  ([.metrics[] | select(.value != null)] | length) == 0 and
  (has("score") | not) and (has("percentage") | not) and (has("average") | not)
' "$output/evaluation.json" >/dev/null
jq -e '
  .schema == "gooo/measurement-boundary/collection/v2" and
  .collector.generated == true and .collector.measured_once == true and
  .collector.operator_authority == "authoring-only" and
  .collector.runtime_authority.repository_writes == 0 and
  .collector.runtime_authority.local_test_executions == 0 and
  .collector.runtime_authority.cross_project_required_gates == 0 and
  ([.observations[] | select(.stage_id != "stage:product-integration" or (.covered_operations | length) != 2 or .rss_process_tree_scope != "process-tree")] | length) == 0
' "$output/generated-collection/collection.json" >/dev/null
python3 - "$output/evaluation.json" "$output/generated-collection/collection.json" "$output/integration-result.json" <<'PY'
import json
import os
import sys

evaluation_path, collection_path, destination = sys.argv[1:]
with open(evaluation_path, encoding="utf-8") as source:
    evaluation = json.load(source)
with open(collection_path, encoding="utf-8") as source:
    collection = json.load(source)
pairs = []
for metric in evaluation["metrics"]:
    pair = metric["improvement"]
    pairs.append({
        "metric_id": metric["measurement_id"],
        "pair_id": pair["pair_id"],
        "scenario_id": "scenario:main-lock",
        "input_digest": "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
        "contract_digest": "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
        "fixture_digest": "sha256:9999999999999999999999999999999999999999999999999999999999999999",
        "toolchain": "go1.27.x",
        "runner": os.environ.get("RUNNER_OS", "GitHub-hosted-runner"),
        "job": os.environ.get("GITHUB_JOB", "v2-integration"),
        "before": pair["before"],
        "after": pair["after"],
    })
result = {
    "schema": "gooo/measurement-boundary/integration/v2",
    "same_ci_job": True,
    "controlled_pairs": pairs,
    "runtime_authority": collection["collector"]["runtime_authority"],
    "local_validation_commands": [],
}
with open(destination, "w", encoding="utf-8") as output:
    json.dump(result, output, indent=2, sort_keys=True)
    output.write("\n")
PY
