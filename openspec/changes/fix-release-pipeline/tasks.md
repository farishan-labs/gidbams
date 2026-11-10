## 1. Fix release workflow

- [ ] 1.1 Add `permissions: contents: write` to the release job in `.github/workflows/release.yml`
- [ ] 1.2 Add `workflow_dispatch` trigger alongside the existing `push: tags` trigger
- [ ] 1.3 Verify build commands match the renamed project: `go test ./...`, `go build -o gidbams ./cmd/gidbams` (Go 1.26)
- [ ] 1.4 Add a SHA256SUMS generation step in the ubuntu job and upload it via softprops

## 2. Validate end-to-end

- [ ] 2.1 Commit and push the workflow fix to main
- [ ] 2.2 Trigger the workflow via `workflow_dispatch` and confirm a run completes green with jobs created
- [ ] 2.3 Push a tag (e.g. `v0.2.0`) and confirm a GitHub Release is published with `gidbams` (linux + macos) and `SHA256SUMS`
- [ ] 2.4 Document the install-from-release steps in README (optional, small)