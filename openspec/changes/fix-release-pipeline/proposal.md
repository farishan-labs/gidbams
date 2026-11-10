## Why

The release workflow (`.github/workflows/release.yml`) has never produced a published release: every tag push since v0.1.x fails at 0s with "workflow file issue" and creates zero jobs. As a result there is no downloadable `gidbams` binary, no GitHub Release, and consumers cannot pin to a released version. This blocks distribution and the "publish as standalone tool" positioning decision.

## What Changes

- Fix `.github/workflows/release.yml` so tag pushes (`v*`) build, test, and publish a GitHub Release with binaries for ubuntu and macos.
- Add the missing `permissions: contents: write` (and any other required) so `softprops/action-gh-release` can create the release.
- Make the workflow's build commands consistent with the renamed project (binary `gidbams`, entry `./cmd/gidbams`, Go 1.26) — current HEAD already reflects this; verify on the tagged commit.
- (Optional) Add a checksum file (`SHA256SUMS`) and attach it to the release.
- Cut a new tag (e.g. `v0.2.0`) after the fix and verify a release publishes end-to-end.

## Capabilities

### New Capabilities
(none — tooling/CI only)

### Modified Capabilities
(none — no spec-level behavior change; `skip_specs: true`)

## Impact

- Affected files: `.github/workflows/release.yml` (and possibly a new checksum step).
- Affected systems: GitHub Actions, GitHub Releases.
- No code/API/runtime changes; no consumer impact.
- Risk: low. Rollback is reverting the workflow file.