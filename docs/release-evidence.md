# Release evidence contract

The release is PR-first. A work branch opens a pull request, the pull-request workflow must pass, and the merged `main` workflow must pass before an annotated release tag is pushed. The tag workflow checks out the tag, verifies that it is an annotated tag, builds a source archive and manifest, creates the GitHub release as a draft, audits asset digests, publishes it, and verifies `draft=false`, `prerelease=false`, and `immutable=true`.

The workflow preserves failed runs, tags, and draft releases. It never deletes a failed release or rewrites an existing release's assets. All GitHub API writes use `github.token`; no cross-project required gate is used.

The release evidence artifact includes the tag object SHA, commit SHA, run ID, job names, artifact names and digests, release ID, asset IDs, immutable flag, semantic IR digest, conformance distribution, v0.48 conflict result, generated single-authority receipt digest, and the complete zero-authority vector.
