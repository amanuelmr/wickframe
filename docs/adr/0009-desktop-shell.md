---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0009: Package the desktop app with Wails v3

## Context and Problem Statement

Personal-mode users (P1) expect a double-click desktop app rather than a terminal
command (FR-OPS-07). Which shell do we use to package the Go backend and web UI as
a native desktop application for Windows, macOS and Linux?

## Decision Drivers

* Reuse the Go backend in-process, with no sidecar process to manage.
* Small download and memory footprint.
* Code signing, installers and auto-update for all three operating systems.
* The same web UI as the server edition.

## Considered Options

* Wails v3
* Tauri v2, with the Go backend as a sidecar
* Electron, with the Go backend as a sidecar
* No desktop shell (the user opens a browser at `localhost`)

## Decision Outcome

Chosen option: **Wails v3**. v3.0.0 was released 2026-10-26 and v3.1.0 on
2026-11-01; it requires Go 1.26+. It runs our Go code natively in the same process,
uses the OS webview, and ships built-in code signing, packaging (NSIS/MSIX, DMG,
AppImage/deb/rpm) and an in-app updater with cryptographic verification and
rollback.

Rules:

1. The desktop app is a **thin shell**: `cmd/wickframe-desktop` starts the same
   `internal/app` in personal mode, and serves the same embedded UI. There is no
   desktop-only business logic.
2. The UI talks to the backend over the same HTTP/WebSocket API, on a loopback port
   chosen at random and protected by a per-launch secret token. It does not use
   Wails bindings. Desktop and server then have one code path.
3. **Pin the Wails minor version.** v3 is new and releases often, so upgrades go
   through the desktop smoke tests.
4. Chart performance is tested early on every webview: WebView2 (Windows),
   WKWebView (macOS) and **WebKitGTK (Linux)**, which is the most likely to lag.
5. Updates are delivered only from signed GitHub Releases, and verified before
   install ([ADR-0015](0015-supply-chain-and-release.md)).

### Consequences

* Good, because downloads are small, there is no bundled Chromium, and there is a
  single process.
* Good, because signing and updating are built in.
* Bad, because v3 has only been stable for a few weeks. Mitigated by pinning, and
  by the fact that the desktop build is optional (M3) and the server and browser
  edition does not depend on it.
* Bad, because webview differences can affect rendering. Mitigated by per-platform
  testing.

### Confirmation

* CI builds desktop artifacts for the three operating systems. A smoke test
  launches the app, loads a chart, and checks the backend health endpoint.

## Pros and Cons of the Options

### Tauri v2 (Go sidecar)

* Good, because it is mature, small and secure.
* Bad, because the Go backend must run as a separate sidecar process, which adds
  lifecycle and IPC complexity, and Tauri's own core is Rust.

### Electron (Go sidecar)

* Good, because rendering is consistent everywhere (bundled Chromium).
* Bad, because each app is about 100 MB plus a sidecar process, with higher memory
  use.

### No shell

* Good, because there is zero extra work.
* Bad, because it is a poor experience for non-technical P1 users. It remains
  supported anyway (`wickframe serve` then open the browser).

## More Information

* Wails v3.0.0 release notes: <https://github.com/wailsapp/wails/releases/tag/v3.0.0>
* Tauri sidecars: <https://v2.tauri.app/develop/sidecar/>
* **Revisit when:** Wails v3 shows instability during M3, or WebKitGTK performance
  is unacceptable.
