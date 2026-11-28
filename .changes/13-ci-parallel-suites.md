### ci: shard the Go suites and gate OpenWrt by path
- The single `test` job is split into `web`, `test-go` and `e2e`, so the browser
  suite no longer waits behind `go test ./...` and vice versa.
- `test-go` and `race` are sharded by package and `integration` by test name,
  through deterministic partition scripts that fail on an empty shard and keep
  the union equal to the whole module.
- Coverage is now merged from the `test-go` shard profiles instead of running the
  suite a fourth time, so the gate stays pre-merge without the extra run.
- The OpenWrt VM test runs on packaging paths for pull requests, on every push to
  `main`, and nightly; a Go-only packaging break is caught at the merge commit
  rather than before it.
- Measured before the change: Race 12.3 min, `test` 10.9 min, Integration 9.9 min,
  Coverage 5.7 min, OpenWrt 9.6 min per PR.
