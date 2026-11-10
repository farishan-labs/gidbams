## Context

See proposal.md — Why. Current `.github/workflows/release.yml` triggers on `v*` tag pushes, runs `go test ./...` and `go build -o gidbams ./cmd/gidbams` on ubuntu/macos, then calls `softprops/action-gh-release@v2`. Every such run completes in 0s with no jobs and GitHub reports "workflow file issue". The CI workflow (`.github/workflows/ci.yml`) is green, so Go toolchain and tests are fine.

## Goals / Non-Goals

**Goals:**
- Make tag pushes produce a published GitHub Release with linux + macos binaries.
- Keep the workflow minimal and consistent with the renamed project (Go 1.26, binary `gidbams`, entry `./cmd/gidbams`).

**Non-Goals:**
- No goreleaser / multi-arch (arm64) matrix in this change; keep ubuntu+macos as today.
- No changes to CI workflow or application code.
- No automatic tag creation; tags are still pushed manually.

## Decisions

- **Add `permissions: contents: write`** to the release job. `softprops/action-gh-release` needs write permission to create a release; the default `GITHUB_TOKEN` permissions on this repo appear to be read-only, which explains the 0s failure with no jobs. Alternative considered: use a PAT, but the built-in token with explicit permissions is the standard approach and avoids secret management.
- **Keep trigger as `push: tags: ['v*']`**. Simple and matches existing practice (many v0.1.x tags). Alternative: also trigger on workflow_dispatch for manual reruns — add `workflow_dispatch` as a secondary trigger so a broken tag can be re-run without re-pushing. Decision: add `workflow_dispatch` for operability.
- **Verify build commands against HEAD** before tagging: `go test ./...`, `go build -o gidbams ./cmd/gidbams`. Already correct in current release.yml.
- **Attach SHA256SUMS** generated in a step (`sha256sum gidbams > SHA256SUMS` on linux; use `shasum -a 256` on macos or run checksum generation on ubuntu only). Decision: generate checksums on ubuntu and upload both; keep macos build for the macos binary only. Simpler alternative: let softprops upload raw binaries; checksums are a nice-to-have. Decision: include checksums for integrity, generated in the ubuntu job.

## Risks / Trade-offs

- [Workflow still fails due to an unrelated cause] → Mitigation: after editing, push a test tag (e.g. `v0.2.0-rc.1`) or use `workflow_dispatch` to validate before promoting; inspect Actions logs.
- [macos runner binary not code-signed] → Acceptable for v0.x; document in README install section.
- [Old tags (v0.1.x) pointed at pre-rename commits with stale workflow] → Those releases remain unavailable; only new tags get releases. Acceptable.

## Migration Plan

1. Edit `.github/workflows/release.yml` (permissions, workflow_dispatch, checksums).
2. Commit and push to main.
3. Trigger via `workflow_dispatch` (or push a tag) and confirm a release is created.
4. Push `v0.2.0` tag to publish the first real release.
5. Rollback: revert the workflow commit; no runtime impact.

## Open Questions

- None blocking. (Whether to move to goreleaser later is tracked separately.)