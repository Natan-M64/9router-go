### ci(ghcr): multi-arch GHCR images published on release tags, plus local CI gates

- New GHCR workflow builds `linux/amd64` and `linux/arm64` natively (`ubuntu-24.04-arm` for arm), runs as the non-root `9router` user with `CMD ["serve"]`, lints with `hadolint` and `actionlint`, and shares one `web/dist` artifact across jobs. `docker-compose.yml`, `docker-stack.yml` and `.env.example` are trimmed to a minimal bootstrap pointing at the fork image.
- Images are published only for `v*` tags, not for every commit to `main`. `:latest` moves only for stable tags: a tag containing a hyphen (a prerelease such as `v1.2.3-exp.1`) never takes it.
- A push to `main` that only touches Markdown, `docs/`, `wiki/`, `screenshots/` or `site/` no longer starts the CI pipeline. Pull requests are not filtered.
- `make ci-local` runs the gates that do not need a CI runner, in CI order and stopping at the first failure: oxlint, the svelte-check ratchet, frontend unit tests, `go vet` and `go test -count=1 ./...`.
- Tests that depended on the platform now skip with a stated reason instead of failing on macOS: the Wayland display tests off Linux, and the unix-socket tests, which use a short temp directory because macOS caps a socket path at 104 bytes.
