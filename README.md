# Cerberus

> **On-premise Linux XDR prototype with a C/eBPF kernel sensor, Rust endpoint agent, and Go command tower for local-only AI-assisted telemetry analysis.**

**Builder / Creator: Joseph Sierengowski**

---

## What Cerberus Is

Cerberus is an **on-premise Linux endpoint detection and response (EDR/XDR) research prototype** designed for local-only telemetry collection, offline AI-assisted analysis, and autonomous host-defense experimentation. It captures process execution activity at the kernel level, processes events entirely on the local host, and evaluates them through a locally hosted AI model — no cloud dependency required.

Cerberus was designed and built by **Joseph Sierengowski**, whose original concept, architecture direction, and implementation vision are reflected throughout the project.

---

## What It Does

Cerberus operates as a three-component pipeline:

1. **Kernel-layer telemetry capture** — A C/eBPF sensor attaches to the `tracepoint/syscalls/sys_enter_execve` kernel hook and captures execution events including PID, PPID, UID, command name, and target binary path.

2. **Local event transport** — A Rust endpoint agent reads events from the eBPF ring buffer, serializes them, and forwards structured telemetry to the Go command service over loopback HTTP.

3. **Offline AI-assisted analysis** — A Go command tower receives telemetry events and submits them to locally hosted Ollama AI models for consensus-style security reasoning and classification.

Additionally, the project includes an experimental deception layer (`internal/deception/`) for creating lightweight in-memory honeypot-style environments using Linux tmpfs mounts.

---

## How It Is Built

Cerberus is composed of three distinct implementation layers with clear separation of concerns:

| Layer | Language | Location | Role |
|-------|----------|----------|------|
| Kernel sensor | C / eBPF | `internal/kernel/sensor.bpf.c` | Execution telemetry capture |
| Endpoint agent | Rust | `cmd/agent/` | Ring buffer consumption and event forwarding |
| Command tower | Go | `cmd/server/` | Event intake and local AI analysis |

The kernel header `internal/kernel/vmlinux.h` is generated from the target kernel using `bpftool btf dump file /sys/kernel/btf/vmlinux format c`. It must match the kernel version of the build host.

---

## Prerequisites

To build and run Cerberus, you will need:

- Linux kernel 5.8 or later (for BPF ring buffer support)
- `clang` and LLVM (for BPF compilation)
- `libbpf-dev` or equivalent
- Rust toolchain (stable, via `rustup`)
- Go 1.21 or later
- `bpftool` (for regenerating `vmlinux.h` if needed)
- [Ollama](https://ollama.com/) running locally with at least one model loaded (for the AI analysis path)

---

## One-Command Installation & Management

To install, configure, persist, and run Cerberus with a single script, use the all-in-one download installer.

### 1. Install & Launch Cerberus Services
Run the installer script with root privileges:

```bash
sudo ./install.sh
```

This installer automatically:
- Checks kernel compatibility and build tools
- Auto-installs missing dependencies (via `apt` on Debian/Ubuntu systems)
- Compiles the eBPF sensor, Go command tower, and Rust agent
- Installs the binaries and deception scripts to `/usr/local/bin`
- Installs and starts system-resilient, boot-persistent **systemd services** (`cerberus-server`, `cerberus-agent`, and `cerberus-deception`)

### 2. Launch the Control TUI Menu
Once installed, you can monitor and interact with the Cerberus ecosystem from any shell context using our dedicated Terminal User Interface:

```bash
cerberus-ctl
```

Inside the terminal menu, you can:
- **View service status** for the server, agent, and deception maze
- **Toggle autokill mode** on or off dynamically (which automatically updates the agent and reloads systemd)
- **Tails live telemetry logs** from our JSON database
- **Ask Cerberus Security AI** custom natural language questions (e.g., "Is the system running low on RAM?", "Summarize recent suspicious telemetry events", etc.) directly from your terminal.

---

## Quick Start (Manual Method)

### 1. Build all components

```bash
make build
```

This compiles:
- the Go command tower (`cmd/server/cerberus-server`)
- the Rust endpoint agent (`cmd/agent/target/release/`)
- the eBPF sensor object (`cmd/agent/sensor.bpf.o`)

### 2. Start the Go command tower

```bash
cd cmd/server
sudo ./cerberus-server
```

The server listens on `http://127.0.0.1:8080` by default.

### 3. Start the Rust agent

```bash
cd cmd/agent
sudo ./target/release/cerberus-agent
```

The agent loads `sensor.bpf.o`, attaches it to the kernel tracepoint, and begins forwarding execution events.

> **Note:** By default, the agent runs in **observation mode only**. Autonomous process termination is disabled unless you explicitly opt in. See [Autonomous Response](#autonomous-response) below.

### 4. (Optional) Deploy the deception layer

```bash
sudo bash internal/deception/spawn_mirror.sh
```

This mounts a tmpfs-backed honeypot directory using `internal/deception/deploy_maze.sh`. The maze path and size can be overridden with environment variables:

```bash
CERBERUS_MAZE_DIR=/var/cerberus/maze CERBERUS_MAZE_SIZE=128M sudo bash internal/deception/spawn_mirror.sh
```

---

## Autonomous Response

> **Autonomous kill behavior is disabled by default.**

Cerberus ships with observation-first defaults. The agent will log detected execution events and forward them to the command tower, but will not automatically terminate processes unless explicitly enabled.

To enable autonomous response:

```bash
CERBERUS_ENABLE_AUTOKILL=1 sudo ./target/release/cerberus-agent
```

Accepted values: `1`, `true`, `yes` (case-insensitive where applicable).

The prototype heuristic checks a narrow list of suspicious execution indicators. **This is not a production policy engine.** Review and refine before enabling autokill in any real environment.

---

## Build Targets

| Target | Description |
|--------|-------------|
| `make build-server` | Build the Go command tower |
| `make build-agent` | Build the Rust endpoint agent |
| `make build-bpf` | Compile the eBPF sensor object |
| `make build` | Build all components |
| `make all` | Alias for `make build` |
| `make clean` | Remove generated binaries and objects |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         Linux Kernel                        │
│   tracepoint/syscalls/sys_enter_execve                      │
│          │                                                  │
│   ┌──────▼──────┐                                           │
│   │ eBPF Sensor │  C / eBPF — sensor.bpf.c                 │
│   │  (Left Head)│                                           │
│   └──────┬──────┘                                           │
│          │ BPF ring buffer                                  │
└──────────┼──────────────────────────────────────────────────┘
           │
   ┌───────▼────────┐
   │  Rust Agent    │  cmd/agent/src/main.rs
   │ (Center Head)  │  — consumes ring buffer
   │                │  — serializes events
   └───────┬────────┘
           │ HTTP POST → 127.0.0.1:8080/telemetry
   ┌───────▼────────┐
   │  Go Command    │  cmd/server/main.go
   │    Tower       │  — receives telemetry
   │ (Right Head)   │  — queries local Ollama
   └────────────────┘
```

---

## Disclaimer

> **Cerberus is an experimental Linux security research project.**

This repository contains kernel-space telemetry code, autonomous process-termination logic, and deception-oriented shell scripts. These capabilities can disrupt a host if deployed without careful review and validation.

**Cerberus should not be deployed as a production protection platform without additional safety controls, policy enforcement, testing, and operational hardening.**

Deploy only in controlled lab environments, test systems, or explicitly authorized environments.

This project is not marketed as guaranteed protection, a complete XDR platform, or a drop-in enterprise security product. It is a prototype and research tool.

---

## Safe-Use Notice

Before running Cerberus, ensure that you:

- understand the operational impact of attaching eBPF programs to a running kernel
- have validated kernel compatibility on the target host
- have reviewed and constrained all automated response behavior before enabling autokill
- are running on a non-production, lab, or test system
- have verified that Ollama model prompts and responses align with your intended classification policy

---

## License

This project is licensed under the [MIT License](LICENSE).  
Copyright © 2026 Joseph Sierengowski.

---

## Builder Credit

> **Created by Joseph Sierengowski.**
>
> Cerberus reflects Joseph Sierengowski's original concept, architecture direction, and implementation work across kernel telemetry, local agent design, and on-premise autonomous defense orchestration. Builder credit and authorship belong with Joseph Sierengowski and should remain visible in all documentation, release materials, and project presentations derived from this work.

---

## Suggested Repository Description

> On-premise Linux XDR prototype with a C/eBPF kernel sensor, Rust endpoint agent, and Go command tower for local-only AI-assisted telemetry analysis. Built by Joseph Sierengowski.
