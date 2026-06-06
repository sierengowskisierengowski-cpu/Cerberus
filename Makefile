SHELL := /bin/bash

.PHONY: help build-server build-agent build-bpf build all clean

help:
	@echo "Cerberus build targets:"
	@echo "  make build-server  - Build the Go command tower"
	@echo "  make build-agent   - Build the Rust endpoint agent"
	@echo "  make build-bpf     - Compile the eBPF sensor object"
	@echo "  make build         - Build all components"
	@echo "  make all           - Alias for build"
	@echo "  make clean         - Remove generated binaries and objects"

build-server:
	cd cmd/server && go build -o cerberus-server .

build-agent:
	cd cmd/agent && cargo build --release

build-bpf:
	clang -O2 -g -target bpf -c internal/kernel/sensor.bpf.c -o cmd/agent/sensor.bpf.o

build: build-bpf build-server build-agent

all: build

clean:
	rm -f cmd/server/cerberus-server
	rm -f cmd/agent/sensor.bpf.o
