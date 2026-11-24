#!/usr/bin/env bash
set -euo pipefail

# Installs/manages macOS launchd Socket Activation + smart maintain timer for 9router-go.
# Zero CPU/RAM/battery when idle; wakes on first HTTP request to 127.0.0.1:PORT.

BIN_PATH="${NINEROUTER_BIN:-$(cd "$(dirname "$0")/.." && pwd)/9router-go}"
PORT="${PORT:-20130}"
IDLE_TIMEOUT="${IDLE_TIMEOUT:-15m}"
MAINTAIN_INTERVAL="${MAINTAIN_INTERVAL:-1800}"

AGENTS_DIR="${HOME}/Library/LaunchAgents"
GATEWAY_PLIST="${AGENTS_DIR}/com.9router.gateway.ondemand.plist"
MAINTAIN_PLIST="${AGENTS_DIR}/com.9router.maintain.plist"

usage() {
  echo "Usage: $0 {install|uninstall|status}"
  exit 1
}

case "${1:-}" in
  install)
    if [ ! -x "$BIN_PATH" ]; then
      echo "Binary not found or not executable at: $BIN_PATH" >&2
      echo "Run 'make build' first." >&2
      exit 1
    fi
    mkdir -p "$AGENTS_DIR"

    cat > "$GATEWAY_PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.9router.gateway.ondemand</string>
	<key>ProgramArguments</key>
	<array>
		<string>${BIN_PATH}</string>
		<string>--idle-timeout</string>
		<string>${IDLE_TIMEOUT}</string>
		<string>serve</string>
	</array>
	<key>Sockets</key>
	<dict>
		<key>Listeners</key>
		<dict>
			<key>SockNodeName</key>
			<string>127.0.0.1</string>
			<key>SockServiceName</key>
			<string>${PORT}</string>
		</dict>
	</dict>
	<key>RunAtLoad</key>
	<false/>
	<key>KeepAlive</key>
	<false/>
</dict>
</plist>
EOF

    cat > "$MAINTAIN_PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.9router.maintain</string>
	<key>ProgramArguments</key>
	<array>
		<string>${BIN_PATH}</string>
		<string>maintain</string>
		<string>--quiet</string>
	</array>
	<key>StartInterval</key>
	<integer>${MAINTAIN_INTERVAL}</integer>
	<key>RunAtLoad</key>
	<false/>
</dict>
</plist>
EOF

    launchctl unload -w "$GATEWAY_PLIST" >/dev/null 2>&1 || true
    launchctl unload -w "$MAINTAIN_PLIST" >/dev/null 2>&1 || true
    launchctl load -w "$GATEWAY_PLIST"
    launchctl load -w "$MAINTAIN_PLIST"
    echo "Installed on-demand launchd agents (127.0.0.1:${PORT}, idle=${IDLE_TIMEOUT}, maintain=${MAINTAIN_INTERVAL}s)."
    ;;

  uninstall)
    launchctl unload -w "$GATEWAY_PLIST" >/dev/null 2>&1 || true
    launchctl unload -w "$MAINTAIN_PLIST" >/dev/null 2>&1 || true
    rm -f "$GATEWAY_PLIST" "$MAINTAIN_PLIST"
    echo "Uninstalled on-demand launchd agents."
    ;;

  status)
    launchctl list | grep -E "com\.9router\.(gateway\.ondemand|maintain)" || echo "No on-demand launchd agents loaded."
    ;;

  *)
    usage
    ;;
esac
