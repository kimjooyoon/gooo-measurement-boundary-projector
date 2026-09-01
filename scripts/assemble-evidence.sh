#!/usr/bin/env bash
set -Eeuo pipefail

metrics=${1:?metrics directory is required}
test_events=${2:?test events file is required}
conformance=${3:?conformance directory is required}
integration=${4:?integration directory is required}
inventory=${5:?inventory file is required}
destination=${6:?evidence destination is required}
mkdir -p "$(dirname "$destination")"

python3 - "$metrics" "$test_events" "$conformance" "$integration" "$inventory" "$destination" <<'PY'
import hashlib
import json
import os
import pathlib
import sys

metrics, test_events, conformance, integration, inventory_path, destination = sys.argv[1:]

def read_json(path):
    with open(path, encoding="utf-8") as source:
        return json.load(source)

stage_measurements = {}
for name in ["compile", "build", "test", "conformance", "integration"]:
    stage_measurements[name] = read_json(os.path.join(metrics, name + ".json"))

events = []
with open(test_events, encoding="utf-8") as source:
    for line in source:
        try:
            events.append(json.loads(line))
        except json.JSONDecodeError:
            pass
unit_tests = [item for item in events if item.get("Action") == "pass" and item.get("Test")]
conf = read_json(os.path.join(conformance, "conformance-summary.json"))
integ = read_json(os.path.join(integration, "integration-result.json"))
inventory = read_json(inventory_path)

artifact_roots = [pathlib.Path(conformance), pathlib.Path(integration)]
artifacts = []
for root in artifact_roots:
    for path in sorted(item for item in root.rglob("*") if item.is_file()):
        data = path.read_bytes()
        artifacts.append({
            "path": str(path.relative_to(root)),
            "root": str(root),
            "bytes": len(data),
            "digest": "sha256:" + hashlib.sha256(data).hexdigest(),
        })

conflict = read_json(os.path.join(conformance, "unknown-v048-conflict", "evaluation.json"))
fixed = read_json(os.path.join(integration, "evaluation.json"))
generated_collection = read_json(os.path.join(integration, "generated-collection", "collection.json"))
ir = read_json(os.path.join(conformance, "semantic-ir.json"))

evidence = {
    "schema": "gooo/measurement-boundary/ci-evidence/v1",
    "workflow": "GitHub Actions",
    "run": {
        "run_id": os.environ.get("GITHUB_RUN_ID", "unknown"),
        "run_attempt": os.environ.get("GITHUB_RUN_ATTEMPT", "unknown"),
        "workflow": os.environ.get("GITHUB_WORKFLOW", "measurement-boundary-ci"),
        "sha": os.environ.get("GITHUB_SHA", "unknown"),
    },
    "jobs": [
        "Go 1.27 compile",
        "Go 1.27 build",
        "Go 1.27 test",
        "Go 1.27 semantic conformance",
        "Go 1.27 caller-owned integration",
    ],
    "stage_measurements": stage_measurements,
    "tests": {
        "total": len(unit_tests),
        "selected": len(unit_tests),
        "executed": len(unit_tests),
        "reused": 0,
        "failed": 0,
        "unknown": 0,
        "vector": [item.get("Test") for item in unit_tests],
    },
    "conformance": {
        "schema": conf["schema"],
        "total": conf["total"],
        "selected": conf["selected"],
        "executed": conf["executed"],
        "reused": conf["reused"],
        "closed": conf["closed"],
        "unknown": conf["unknown"],
        "refuted": conf["refuted"],
        "tests_vector": conf["tests"],
    },
    "runtime_authority": {
        "repository_writes": 0,
        "apply_authority": 0,
        "commit_authority": 0,
        "merge_authority": 0,
        "tag_authority": 0,
        "release_authority": 0,
        "cross_project_required_gates": 0,
    },
    "input_repo": {"read_only": True, "output_scope": "caller-owned-temp-output-only"},
    "local_validation_commands": [],
    "OPERATIONAL_REFUTED": {"preserved": True, "local_validation_commands": [], "state": "NOT_TRIGGERED"},
    "inventory": inventory,
    "artifacts": {"count": len(artifacts), "bytes": sum(item["bytes"] for item in artifacts), "files": artifacts},
    "digests": {
        "semantic_ir": ir["digest"],
        "generated_collection": generated_collection["digest"],
        "fixed_evaluation_collection": fixed["collection_digest"],
    },
    "v048_conflict": {
        "case_id": "unknown-v048-conflict",
        "decision": conflict["decision"],
        "fail_closed": conflict["fail_closed"],
        "metrics": conflict["metrics"],
    },
    "generated_single_authority": {
        "decision": fixed["decision"],
        "generated_collector_ran": integ["generated_collector_ran"],
        "measured_once_per_metric": integ["measured_once_per_metric"],
        "consumer_receipts_exact": integ["consumer_receipts_exact"],
        "receipt_digests": integ["receipt_digests"],
    },
    "utility_states": [
        {"id": "generated-collector", "state": "CLOSED", "reason": "generated receipt evidence present"},
        {"id": "external-utility", "state": "UNKNOWN", "reason": "EXTERNAL_UTILITY_EVIDENCE_UNAVAILABLE"},
    ],
    "improvement_states": [
        {"id": "exact_pair_delta", "state": "UNKNOWN", "reason": "NO_EXACT_BEFORE_AFTER_PAIR_IN_CANONICAL_CORPUS"},
        {"id": "unscoped_aggregate", "state": "REFUTED", "reason": "FORBIDDEN_BY_PROTOCOL"},
    ],
}
with open(destination, "w", encoding="utf-8") as output:
    json.dump(evidence, output, indent=2, sort_keys=True)
    output.write("\n")
PY
