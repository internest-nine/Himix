# Himix

<p align="center">
  <img src="support/icon.ico" alt="Himix icon" width="128" height="128"><br><br>
  <a href="README.md">简体中文</a> | <a href="README_EN.md">English</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Version-0.1-2F6BFF?style=flat-square" alt="Version 0.1">
  <img src="https://img.shields.io/badge/Base-HypoMux%20v2.8.2-0078D4?style=flat-square" alt="Based on HypoMux v2.8.2">
  <img src="https://img.shields.io/badge/Core-Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Desktop-Wails%20v3-CB3837?style=flat-square" alt="Wails v3">
  <img src="https://img.shields.io/badge/UI-React%20%2B%20Fluent%20UI-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React and Fluent UI">
  <img src="https://img.shields.io/badge/Platform-Windows%2010%20%2F%2011-0078D4?style=flat-square&logo=windows" alt="Windows 10 and 11">
</p>

Himix is an open-source multi-adapter aggregation and traffic-splitting tool for Windows, derived from [HypoMux v2.8.2](https://github.com/Hypostasis-Cat/HypoMux) (AGPL-3.0).

It distributes multi-connection downloads across several active adapters, so Ethernet, Wi-Fi, a phone hotspot or USB tethering can all carry traffic at the same time.

**What sets it apart from upstream: fully offline, a rebuilt interface, less fiddly routing, and a bundled Windows installer.**
See [Differences from HypoMux](#differences-from-hypomux) below.

> It aggregates many independent connections; it does not split a single TCP connection into several paths. That makes it a good fit for Steam, IDM, game-platform updaters and large browser downloads. A single-connection transfer is still limited by that connection's own link.

---

## Contents

- [Differences from HypoMux](#differences-from-hypomux)
- [New in this fork](#new-in-this-fork)
- [Core features](#core-features)
- [Quick start](#quick-start)
- [How it works](#how-it-works)
- [Technical boundaries](#technical-boundaries)
- [Building from source](#building-from-source)
- [License](#license)

---

## Differences from HypoMux

Upstream v2.8.2 ships a solid engine, and Himix **keeps every engine capability intact** (all 12 modules under `engine/internal/` are present). The changes are concentrated in the product layer.

| Area | HypoMux v2.8.2 | Himix 0.1 |
| --- | --- | --- |
| **Network behaviour** | Built-in AI assistant (configurable model API), MCP external tools, official update check | **All removed** — no outbound calls, no telemetry at runtime |
| **Interface** | Left-hand icon rail (icons only, with tooltips) | **Single bottom pill navigation with labels**; the main area is left entirely to content |
| **Mode naming** | "System proxy mode" / "Virtual NIC mode" | **"Light" and "Full"**, each stating what it does and what it costs |
| **Rule input** | Pick the match type by hand; pasting a URL raises a format error | **Auto-detects** URL / IP / process, so you can paste directly |
| **Outbound choice** | Choose an adapter manually | **Probes** which adapter can reach the target; rewrites the rule when it cannot |
| **Link loss** | Rules pointing at a dropped adapter break with it | **Fails over** to a usable line and switches back silently once it returns |
| **Installation** | Obtain an installer yourself | **Ships a NSIS installer**; installs and uninstalls cleanly |
| **Installer signing** | SignPath signature (verifies a certificate chain online) | **Unsigned**, auditable |
| **Updates** | In-app update check | **Replace the installer manually**; version managed by hand |

### Each difference in one sentence

**Fully offline.** Upstream's AI assistant can talk to any model API, and MCP can expose data to external AI clients; the update check polls an official update channel. Himix **deletes** all of it rather than hiding it behind a switch — 37 frontend files (248 KB of components) and 15 backend files (3,687 lines) removed outright, with zero provider domains left. The `pixi.js` and `pixi-live2d-display` Live2D dependency declarations are gone too: upstream used them to render the assistant's avatar, and nothing here references them. The `pixi`, `cubism4`, `display` and `AIAssistant` chunks no longer exist in the bundle. The trade-off is that you manage versions yourself; the benefit is that **not a single byte leaves your machine at runtime**.

> Upstream's `updater.go` (738 lines) is **kept but never registered** in this fork — no production code path calls it, so the update check does not exist in practice. It is retained so the release pipeline can validate the installer naming contract (`Himix_Setup_<version>.exe`), and tests still cover its logic.

**The interface makes room for content.** Upstream switches pages with a left icon rail (`CompactNavigation`, icons plus tooltips only). Himix moves this to a single labelled pill bar along the bottom, leaving the main area to cards. The title bar is unchanged from upstream, brand text included.

**Routing is less work.** Upstream asks you to decide whether a new rule matches on process or domain before you can add it. Himix decides from what you paste. Once a rule exists you never pick an adapter by hand either — it probes which line reaches the target.

**Installing is easier.** Himix ships a NSIS installer: double-click, install to `C:\Program Files\Himix`, and it registers the core service and creates shortcuts for you. Uninstalling leaves nothing behind.

---

## New in this fork

### Rule input that recognises what you paste

Upstream's rule editor opens on the Process tab, so pasting a URL submits it as a process name and the engine answers with a format error. Himix instead **decides the rule type from the value itself**:

```
https://www.example.com/item/123?id=5   →  domain rule example.com (covers every site under it)
192.168.1.0/24                          →  IP/CIDR rule
D:\Games\steam.exe                      →  process rule
```

Edges covered by 21 unit tests: bare domains, userinfo, trailing dots, internationalised domains converted to punycode, ports, paths, CIDR prefixes, bare IPs, Chinese domains, invalid input, and paths reduced to the executable name.

### Outbound probing that picks the line for you

After a rule is added, Himix **probes each adapter to see whether it can reach the target** and shows the result. Two situations are decided automatically:

- **The chosen line provably cannot reach the target** → switch to the line that can
- **Aggregation is selected but only one line can reach it** → pin the rule to that line

The second case matters most: a dead link inside an aggregation pool makes an intranet site **fail intermittently**, which is worse than pinning it to the single line that works.

### Automatic failover when a link drops

When an adapter disappears, rules pointing at it should not go down with it. Himix:

- **keeps** the adapter the saved rule chose
- projects the rule onto the aggregation pool while that adapter is gone — and by then the pool only holds links that still work
- **restores** the original routing when the link returns or the app restarts, with no user action

This path reuses the normal rule-publication flow (`writeSingBoxRuleSetPlanLocked`) rather than adding a second writer, so behaviour is identical to the ordinary path.

### A rebuilt interface

- **Mode choice moved from jargon to the user's point of view**: upstream says "system proxy mode" / "virtual NIC mode"; Himix says "Light" and "Full", each card stating what will happen and what it costs — Light promises "no driver needed" and "easy to undo" against the caveat "games and some clients may bypass it"; Full promises "covers games and clients" and "needs administrator rights" against "may conflict with other VPNs"
- **Shorter navigation labels**: upstream uses "Home / Routing rules / Network check / System settings"; Himix uses "Boost / Routing / Check-up / Settings"
- **The rule inputs show examples**: upstream only says "e.g. cs2.exe" and "e.g. example.com (includes subdomains)"; Himix says "program name or full path, e.g. cs2.exe" and "just paste a URL, e.g. https://www.example.com/item/123"
- **Brand accent** `#2F6BFF`, 20px card radius, decorative dividers and glow removed
- When no usable adapter is found it gives an actionable hint ("check that an adapter is connected and has a usable IPv4 address, then scan again") plus a rescan button, instead of leaving the list empty

### Windows installer

Packed with NSIS, a four-page Chinese wizard:

- installs to `C:\Program Files\Himix` and registers the `HimixCore` service
- creates desktop and Start-menu shortcuts
- **install transaction**: snapshots files and registry first, then applies changes; any failed step rolls back, so a half-installed state is impossible
- the core service lives in `%ProgramData%\Himix\Core`, with an ACL that grants write access only to LocalSystem and Administrators; ordinary users get read-only
- uninstalling removes the install directory, the registry entry and both shortcuts

### Release pipeline fixes

Upstream's rename and release scripts disagreed in several places; this fork fixes them:

- **Installer name matches the updater**: published as `Himix_Setup_<version>.exe`
- **Engine CI points at a real directory**: fixed the build failure caused by a reference to the retired path
- **Pre-release checks are no longer skipped**: fixed mismatched environment-variable names that made tests call `t.Skip()`
- Added cross-side contract tests comparing the directory name, executable names, service name, pipe name and registry key across the NSIS script, the engine and the desktop

### Two-component version numbers

The version is `0.1`. A missing patch component counts as zero in every comparison (`0.1` and `0.1.0` compare identically), while the string keeps its original spelling so that **release tag, metadata and installer name always agree**.

---

## Core features

The following capabilities come from upstream v2.8.2 in full; Himix removes none of them:

- **System proxy mode**: starts local HTTP/HTTPS and SOCKS5 services; takes over the Windows system proxy by default, or you can turn that off and only hand a local port to apps you configure yourself
- **Virtual adapter mode**: routes broader system traffic through Wintun and sing-box, with WFP, DNS and routing rules for fine-grained splitting
- **Multi-adapter connection scheduling**: picks an outbound adapter per new connection and pins the socket to the real physical link using source-address binding and `IP_UNICAST_IF`
- **Advanced routing rules**: match by process, domain or IP/CIDR, and send traffic to aggregation, direct, Ethernet, Wi-Fi or one specific adapter
- **Per-link blocked domains**: remembers domains a particular link cannot reach so later connections stop being assigned to it
- **Encrypted DNS**: DoH and custom DoT, multiple servers, IPv4/IPv6, designated DNS egress; validates server certificates and never silently falls back to plaintext
- **Live status and diagnostics**: per-adapter rate, connection count and aggregate throughput, plus packet loss, latency, jitter, DNS, gateway and source-binding checks
- **Least-privilege architecture**: the UI does not run as administrator; only the separate Core service holds the network privileges it needs
- **Steam download optimisation**, **hotspot panel**, **rule-set panel**, **connection viewer**, **MTU / NAT probing**, **blocked domains** and other tools

### Choosing between the two modes

| Mode | Coverage | Privileges and compatibility | Recommended for |
| --- | --- | --- | --- |
| Light | Apps that follow the Windows system proxy, or that you point at the local HTTP/SOCKS5 port | Lighter; the system-proxy takeover can be turned off; creates no virtual adapter | IDM, browsers, Steam and other proxy-aware downloads |
| Full | Broader TCP/UDP and traffic that is not proxy-aware | Needs the Core service, Wintun/WFP; cannot take over the default route alongside another TUN | Game-platform downloads, WeGame, complex splitting and full takeover |

---

## Quick start

1. Connect at least two working networks, for example wired broadband plus Wi-Fi, or broadband plus USB/hotspot tethering
2. Install Himix (double-click `Himix_Setup_0.1.exe`) or run `himix.exe` directly
3. On the home page, refresh and tick the adapters you want in the aggregation pool
4. Run "Check-up" first to confirm each link has a usable IPv4 address, gateway, DNS and source-binding capability
5. Pick Light or Full mode. Low-latency games, voice and conferencing apps are best added to the direct rules first
6. Start the aggregation engine, then begin the download. If Steam is already running, restart it as prompted so the proxy settings apply fully
7. When you are done, stop aggregation or quit normally; Himix restores the system network settings it changed

---

## How it works

```text
application connections
   │
   ├─ Light: HTTP/HTTPS 10801 · SOCKS5 10800
   └─ Full:  Wintun + sing-box + WFP/DNS/routing rules
                         │
                  himix-engine.exe
                         │
        pick aggregation / direct / a specific adapter per connection
               ┌─────────┼─────────┐
            adapter 1  adapter 2  adapter 3
               └─────────┴─────────┘
                  combined throughput
```

Light mode writes a local proxy chain into the current user's Windows Internet Settings; Full mode creates and manages TUN, WFP, DNS and routing resources through the separate privileged service. The Go engine chooses an egress per new connection, combining local source-address binding with the Windows interface index so different connections reliably take different physical adapters.

---

## Technical boundaries

- Suited to IDM, Steam, Epic Games Launcher, EA App, Xbox, WeGame, Chrome, Edge, Firefox and any download that opens many concurrent connections
- Multi-adapter aggregation is **connection-level load distribution**. It will not push a single TCP connection past its own link's speed, and it does not turn several lines into a link-aggregation protocol with one public IP
- Aggregation targets throughput, not lower latency. Competitive games, voice and video conferencing are better served by direct rules or by pausing aggregation
- Himix does not read game memory, inject DLLs or modify a game's private protocol packets; third-party platforms and anti-cheat rules differ, so users must still follow the relevant terms of service
- Use it only on devices and networks you own or are authorised to use. Do not use it to bypass access controls, security measures or platform restrictions you are not entitled to bypass
- **This build's installer is not code-signed**, so Windows SmartScreen may warn about an unknown publisher

### Important notice and compliance disclaimer

Himix is a transparent, open-source network tool, intended only for devices and network connections you own or are authorised to use. It should not be used to circumvent third-party access controls, network restrictions, platform rules or any unauthorised security measures.

Before using it, make sure you understand these behaviours:

1. **System settings are modified**: while running, Himix may adjust Windows system proxy and/or routing settings to steer traffic into the acceleration core
2. **Local security proxy**: once acceleration is on, the traffic being accelerated passes through the local core for splitting, proxying and multiplexing
3. **Automatic restoration**: stopping the tool or uninstalling restores the system proxy and network settings that were changed
4. **Games and routing rules**: for latency-sensitive titles such as competitive online games, add them to the **direct/bypass rule list**, or pause the tool while playing

---

## Building from source

```bash
# frontend dependencies
cd desktop/frontend && pnpm install

# development mode (hot reload)
cd desktop && wails3 task dev

# production build → bin/himix.exe
cd desktop && wails3 task build

# package the NSIS installer (install NSIS and set MAKENSIS first)
export MAKENSIS="/c/Program Files (x86)/NSIS/makensis.exe"
cd desktop && wails3 task windows:package

# per-user install, no administrator rights required
cd desktop && wails3 task windows:package INSTALL_SCOPE=user
```

**Requirements**: Go 1.26+ · Node 22+ · pnpm · Wails v3 · NSIS 3.13+

**Tests**:

```bash
cd desktop && go test ./... -count=1
cd desktop/frontend && npx tsc --noEmit && npx vitest run
cd engine && go test ./... -count=1
```

### Repository layout

```
src/
├── desktop/          Wails desktop app (Go backend + React frontend)
│   ├── frontend/     React + TypeScript + Fluent UI
│   ├── internal/
│   │   ├── services/     business services (routing, connections, settings, diagnostics…)
│   │   ├── engineclient/ core communication, including the elevated handshake
│   │   ├── platform/     platform adapters and tray
│   │   └── releaseversion/ version parsing and metadata sync
│   ├── build/windows/nsis/  installer scripts
│   └── cmd/          release-version / update-manifest-sign
├── engine/           standalone Go network engine (TUN, WFP, DNS, proxy, scheduling)
├── protocol/         protocol definitions
└── .github/workflows/ CI: build, sign, publish, release smoke tests
```

### Version convention

The version lives in `desktop/VERSION`. Run `go run ./cmd/release-version -write` to propagate it to all 12 metadata locations
(Taskfile, package.json, product.ts, updater.go, config.yml, version.nsh, info.json, manifest, msix…).
**Do not edit individual files by hand** — something will be missed.

Both `X.Y` and `X.Y.Z` spellings are accepted. A missing patch component counts as zero in comparisons while the string keeps its original spelling, so the release tag, metadata and installer name stay consistent.

---

## License

Himix is derived from [HypoMux](https://github.com/Hypostasis-Cat/HypoMux) and released under the **AGPL-3.0** license (see [`LICENSE`](LICENSE)).

This project is a derivative work based on upstream [v2.8.2](https://github.com/Hypostasis-Cat/HypoMux/tree/v2.8.2) (commit `7667cbf9`). The complete upstream source is available at that tag; this repository does not bundle a copy.

> **What AGPL-3.0 actually requires**: private and internal use is unrestricted. As soon as you distribute it — including handing a build to a colleague or sharing binaries inside a team — the derivative work must also be released under AGPL-3.0 with complete source code.

Upstream's sponsors, signing service and acknowledgements are listed in the upstream README and are not repeated here.

---

## Acknowledgements

- **[Hypostasis-Cat](https://github.com/Hypostasis-Cat)** — author and maintainer of upstream HypoMux. This project is based on the v2.8.2 release, and the vast majority of the engine work is upstream's.
- **[SignPath.io](https://signpath.io/)** — provided free Windows code signing for upstream. This fork's installer is unsigned, so it does not apply here.
- Everyone who contributed code, reported issues or gave feedback upstream.

<p align="center">
  <a href="https://github.com/Hypostasis-Cat/HypoMux"><img src="https://img.shields.io/badge/upstream-HypoMux%20v2.8.2-0078D4?style=flat-square" alt="Upstream"></a>
</p>
