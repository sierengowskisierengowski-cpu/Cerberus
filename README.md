# Cerberus

Cerberus is an autonomous, on-premise Linux XDR prototype that combines three local components:

- **C/eBPF kernel sensor** for low-level execution telemetry
- **Rust endpoint agent** for ingesting kernel events and forwarding them locally
- **Go command tower** for local event intake and AI-assisted analysis through Ollama

The project is designed around a strict local-first model: telemetry remains on the host, analysis is performed against loopback services, and no cloud dependency is required for the current architecture.

## Professional overview

Cerberus is a Linux-focused endpoint security prototype that observes `execve` activity, forwards structured telemetry through a local pipeline, and applies offline model-driven analysis to help classify suspicious execution behavior. In its current form, the repository demonstrates a promising architecture for host-based telemetry collection and local response experimentation, especially for users interested in combining eBPF visibility with offline AI-assisted security reasoning.

### What the tool is

Cerberus is a **local endpoint detection and response prototype** for Linux systems. It is intended to:

- capture process execution events from the kernel
- normalize those events into a shared payload
- ship them between local services only
- evaluate activity through local language models served by Ollama
- provide a foundation for future autonomous response controls

### What it does today

At the current stage, the repository includes:

1. an **eBPF sensor** attached to `tracepoint/syscalls/sys_enter_execve`
2. a **Rust agent** that reads ring buffer events and forwards them to a local HTTP endpoint
3. a **Go control service** that receives telemetry and queries locally hosted AI models for analysis
4. experimental deception scripts for a local maze or honeypot-style layer

### How it is made

Cerberus is composed of three implementation layers:

- **Kernel telemetry layer (C/eBPF):** collects execution metadata such as PID, PPID, UID, command name, and target filename.
- **Agent layer (Rust):** loads the compiled eBPF object, attaches it to the kernel tracepoint, consumes events from a ring buffer, serializes them, and forwards them to the command service.
- **Analysis layer (Go):** exposes a loopback HTTP endpoint, accepts JSON telemetry, and sends prompts to Ollama-hosted local models for consensus-style security reasoning.

This separation is a strong architectural decision because it keeps kernel capture, user-space transport, and AI analysis clearly distinct.

## Current audit summary

### Strengths

- Clear local-first security concept
- Good separation between kernel capture, agent processing, and command logic
- Practical use of eBPF ring buffers for low-overhead event transport
- Straightforward Rust-to-Go telemetry handoff
- Strong thematic identity and project direction

### Issues identified and corrected in this audit

- Added a professional README where none existed
- Added a dedicated disclaimer section describing prototype status and safe-use boundaries
- Replaced repository backup clutter by removing the committed `main.go.save` artifact
- Added a `.gitignore` rule to avoid committing editor backups and temporary save files again
- Standardized the public project description and professional positioning

### Issues still present in the codebase

1. **Hard-coded filesystem paths in deception scripts**
   - `internal/deception/deploy_maze.sh` uses a user-specific absolute path.
   - This should be made configurable or relative.

2. **Unsafe response automation heuristic**
   - `cmd/agent/src/main.rs` kills processes using broad string matching (`/tmp/` or `nc`).
   - This is too aggressive for production and can terminate legitimate processes.

3. **No build orchestration at repository root**
   - There is no top-level build, setup, or developer workflow file.

4. **No installation or operational instructions before this audit**
   - The project needed documentation for prerequisites, build flow, and component responsibilities.

5. **Generated kernel header committed without context**
   - `internal/kernel/vmlinux.h` is present, but there was no explanation of how it was generated or when it should be regenerated.

6. **No explicit license file**
   - The repository currently has no top-level license, which should be addressed before broader distribution.

## Disclaimer

**Important:** Cerberus should currently be treated as a **security research and prototype project**, not as a production-ready security control.

The repository contains kernel-space telemetry code, autonomous process-killing logic, and deception-oriented shell scripts. These capabilities can disrupt a host if deployed without validation. Run only in controlled lab environments, test systems, or explicitly authorized environments.

This project should **not** be marketed as guaranteed protection, a complete XDR platform, or a drop-in enterprise defense product in its current state.

## Safe-use notice

Before using Cerberus, ensure that you:

- understand the operational impact of attaching eBPF programs
- validate kernel compatibility on the target host
- review and constrain all automated response behavior
- test on non-production systems first
- verify that local Ollama models and prompts align with your intended decision policy

## Builder credit

**Builder credit:** Joseph Sierengowski

Cerberus has a strong original vision and identity. Credit for the project concept, architecture direction, and implementation belongs with **Joseph Sierengowski**, whose authorship should remain visible in repository documentation, release notes, and any project presentation material.

## Recommended next steps

1. Replace hard-coded paths in deception scripts with configurable variables.
2. Move autonomous kill logic behind a policy flag, quarantine mode, or allowlist/denylist model.
3. Add a top-level build script or Makefile.
4. Add a LICENSE file.
5. Add architecture diagrams and sample event flow documentation.
6. Add tests or at minimum smoke-test instructions for the Go and Rust services.
7. Explain how `sensor.bpf.o` and `vmlinux.h` are produced.

## Suggested repository description

**Cerberus is an autonomous on-premise Linux XDR prototype that combines a C/eBPF kernel sensor, a Rust endpoint agent, and a Go command tower with local AI-assisted analysis through Ollama. It is built for local-only telemetry, offline security experimentation, and research into autonomous host defense workflows.**

## My direct assessment

My full view is that this is a **seriously interesting prototype with a strong identity**, especially for an early-stage security project. The architecture is conceptually sound: kernel sensor in C/eBPF, event handling in Rust, orchestration in Go, and local model inference over loopback. That is a credible technical direction for a privacy-preserving host defense tool.

What keeps it from feeling fully professional yet is not the vision — the vision is strong — but the operational polish. Right now the repository needs better guardrails, cleaner packaging, clearer setup instructions, and tighter safety boundaries around automated response behavior. Once those are improved, the project will present much more convincingly to security engineers, collaborators, and evaluators.

In short: **good concept, compelling architecture, promising build, not yet production-safe, but absolutely worth refining.**
