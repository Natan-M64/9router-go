### feat(cli): on-demand gateway on macOS and Linux

- `--idle-timeout` (env `IDLE_TIMEOUT`) shuts the gateway down after that long with no active requests.
- `9router-go maintain` runs one OAuth token refresh and the due quota auto-pings, only when needed, then exits.
- Socket activation for `launchd` (macOS) and `systemd` (Linux), with `scripts/macos-on-demand.sh` to set it up.
