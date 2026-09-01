#!/usr/bin/env bash
set -Eeuo pipefail

conformance=${1:?v2 conformance directory is required}
integration=${2:?v2 integration directory is required}
metrics=${3:?stage metrics directory is required}
inventory=${4:?inventory file is required}
destination=${5:?evidence destination is required}
mkdir -p "$(dirname "$destination")"

python3 - "$conformance" "$integration" "$metrics" "$inventory" "$destination" <<'PY'
import hashlib
import json
import os
import pathlib
import sys

conformance, integration, metrics, inventory_path, destination = sys.argv[1:]

def read_json(path):
    with open(path, encoding="utf-8") as source:
        return json.load(source)

summary = read_json(os.path.join(conformance, "conformance-summary.json"))
integration_result = read_json(os.path.join(integration, "integration-result.json"))
evaluation = read_json(os.path.join(integration, "evaluation.json"))
inventory = read_json(inventory_path)
stages = {name: read_json(os.path.join(metrics, name + ".json")) for name in ["build", "conformance", "integration"]}
artifacts = []
for root in [pathlib.Path(conformance), pathlib.Path(integration)]:
    for path in sorted(item for item in root.rglob("*") if item.is_file()):
        data = path.read_bytes()
        artifacts.append({
            "root": str(root),
            "path": str(path.relative_to(root)),
            "bytes": len(data),
            "digest": "sha256:" + hashlib.sha256(data).hexdigest(),
        })

evidence = {
    "schema": "gooo/measurement-boundary/ci-evidence/v2",
    "workflow": "GitHub Actions",
    "run": {
        "run_id": os.environ.get("GITHUB_RUN_ID", "unknown"),
        "run_attempt": os.environ.get("GITHUB_RUN_ATTEMPT", "unknown"),
        "workflow": os.environ.get("GITHUB_WORKFLOW", "measurement-boundary-v2-ci"),
        "sha": os.environ.get("GITHUB_SHA", "unknown"),
    },
    "jobs": ["Go 1.27.x v2 build", "Go 1.27.x v2 conformance", "Go 1.27.x v2 generated integration"],
    "stage_measurements": stages,
    "denominator": {"total": summary["total"], "CLOSED": summary["closed"], "UNKNOWN": summary["unknown"], "REFUTED": summary["refuted"]},
    "tests_vector": summary["tests"],
    "controlled_pairs": summary["controlled_pairs"] + integration_result["controlled_pairs"],
    "optional_observations": summary["optional_observations"],
    "runtime_authority": {
        "repository_writes": 0,
        "apply_authority": 0,
        "commit_authority": 0,
        "merge_authority": 0,
        "tag_authority": 0,
        "release_authority": 0,
        "local_test_executions": 0,
        "cross_project_required_gates": 0,
        "operator_authority": "authoring-only",
    },
    "input_repo": {"read_only": True, "output_scope": "caller-owned-temp-output-only", "root_readme_excluded": inventory["root_readme_excluded"]},
    "local_validation_commands": [],
    "v1_history": {"contract_preserved": True, "denominator": {"total": 10, "CLOSED": 3, "UNKNOWN": 4, "REFUTED": 3}, "OPERATIONAL_REFUTED": {"exact_count": 1, "preserved": True}},
    "artifacts": {"count": len(artifacts), "bytes": sum(item["bytes"] for item in artifacts), "files": artifacts},
    "digests": {"semantic_ir": read_json(os.path.join(conformance, "semantic-ir.json"))["digest"], "generated_collection": read_json(os.path.join(integration, "generated-collection", "collection.json"))["digest"], "evaluation_collection": evaluation["collection_digest"]},
    "integration": {"decision": evaluation["decision"], "same_ci_job": integration_result["same_ci_job"], "pair_vectors": integration_result["controlled_pairs"]},
    "inventory": inventory,
}
with open(destination, "w", encoding="utf-8") as output:
    json.dump(evidence, output, indent=2, sort_keys=True)
    output.write("\n")
PY
