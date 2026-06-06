#!/usr/bin/env bash
# Cerberus All-In-One Interactive Download Installer
# Created by Joseph Sierengowski

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

echo -e "${CYAN}${BOLD}"
echo "========================================================================"
echo "          CERBERUS ALL-IN-ONE SYSTEM INSTALLER | BY JOSEPH SIERENGOWSKI"
echo "========================================================================"
echo -e "${NC}"

if [[ "${EUID}" -ne 0 ]]; then
    echo -e "${RED}[!] Error: Cerberus installation requires root privileges. Please run as root (or with sudo).${NC}"
    exit 1
fi

echo -e "${YELLOW}[*] Step 1: Checking system compatibility...${NC}"
KERNEL_VERSION=$(uname -r)
echo -e "[i] Kernel version detected: ${BOLD}${KERNEL_VERSION}${NC}"
MAJOR_MIN=$(echo "${KERNEL_VERSION}" | cut -d. -f1,2)
# Since bash doesn't support floating point comparisons easily, we can parse major and minor
MAJOR=$(echo "${KERNEL_VERSION}" | cut -d. -f1)
MINOR=$(echo "${KERNEL_VERSION}" | cut -d. -f2)

if [ "${MAJOR}" -lt 5 ] || { [ "${MAJOR}" -eq 5 ] && [ "${MINOR}" -lt 8 ]; }; then
    echo -e "${YELLOW}[!] Warning: Cerberus requires Linux kernel >= 5.8 for BPF ring-buffer support. Current: ${KERNEL_VERSION}. It might not function properly.${NC}"
else
    echo -e "${GREEN}[+] Kernel version is compatible.${NC}"
fi

echo -e "\n${YELLOW}[*] Step 2: Checking build dependencies...${NC}"
MISSING_DEPS=()
for cmd in clang llvm-strip go cargo make; do
    if ! command -v "$cmd" &>/dev/null; then
        MISSING_DEPS+=("$cmd")
    fi
done

if [ ${#MISSING_DEPS[@]} -ne 0 ]; then
    echo -e "[i] Missing build tools: ${MISSING_DEPS[*]}"
    if command -v apt-get &>/dev/null; then
        echo -e "${YELLOW}[*] Attempting to auto-install missing packages using apt...${NC}"
        apt-get update -y
        # map binary command to package
        for dep in "${MISSING_DEPS[@]}"; do
            case "$dep" in
                clang) apt-get install -y clang libbpf-dev ;;
                llvm-strip) apt-get install -y llvm ;;
                go) apt-get install -y golang-go ;;
                cargo) apt-get install -y cargo rustc ;;
                make) apt-get install -y make ;;
            esac
        done
    else
        echo -e "${RED}[!] Error: Please install missing build tools: ${MISSING_DEPS[*]}${NC}"
        exit 1
    fi
fi
echo -e "${GREEN}[+] Build dependencies satisfied.${NC}"

echo -e "\n${YELLOW}[*] Step 3: Compiling Cerberus components...${NC}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

make clean || true
make build

echo -e "${GREEN}[+] Compilation complete.${NC}"

echo -e "\n${YELLOW}[*] Step 4: Installing binaries and assets to standard paths...${NC}"
mkdir -p /usr/local/bin
mkdir -p /usr/local/lib/cerberus
mkdir -p /var/log/cerberus
mkdir -p /etc/cerberus

cp cmd/server/cerberus-server /usr/local/bin/cerberus-server
cp cmd/agent/target/release/cerberus-agent /usr/local/bin/cerberus-agent
cp cmd/agent/sensor.bpf.o /usr/local/lib/cerberus/sensor.bpf.o
cp internal/deception/deploy_maze.sh /usr/local/bin/deploy_maze.sh
cp internal/deception/spawn_mirror.sh /usr/local/bin/spawn_mirror.sh
cp cerberus-ctl /usr/local/bin/cerberus-ctl

chmod +x /usr/local/bin/cerberus-server
chmod +x /usr/local/bin/cerberus-agent
chmod +x /usr/local/bin/deploy_maze.sh
chmod +x /usr/local/bin/spawn_mirror.sh
chmod +x /usr/local/bin/cerberus-ctl

# Ensure fallback empty config if env file doesn't exist
if [ ! -f /etc/cerberus/agent.env ]; then
    echo "CERBERUS_ENABLE_AUTOKILL=0" > /etc/cerberus/agent.env
fi

echo -e "${GREEN}[+] Binary installation complete.${NC}"

echo -e "\n${YELLOW}[*] Step 5: Creating high-availability systemd service configurations...${NC}"

# 1. Command Tower Server Service
cat <<'EOF' > /etc/systemd/system/cerberus-server.service
[Unit]
Description=Cerberus Command Tower Go Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/cerberus-server
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 2. Kernel Telemetry Rust Agent Service
cat <<'EOF' > /etc/systemd/system/cerberus-agent.service
[Unit]
Description=Cerberus eBPF Rust Agent
After=cerberus-server.service
Requires=cerberus-server.service

[Service]
Type=simple
WorkingDirectory=/usr/local/lib/cerberus
EnvironmentFile=-/etc/cerberus/agent.env
ExecStart=/usr/local/bin/cerberus-agent
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 3. Deception Honeypot Service
cat <<'EOF' > /etc/systemd/system/cerberus-deception.service
[Unit]
Description=Cerberus Deception Maze Layer
After=network.target

[Service]
Type=oneshot
RemainAfterExit=yes
Environment=CERBERUS_MAZE_DIR=/var/cerberus/shadow_layer
ExecStart=/usr/local/bin/spawn_mirror.sh
ExecStop=umount /var/cerberus/shadow_layer || true
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}[+] Systemd unit files successfully written.${NC}"

echo -e "\n${YELLOW}[*] Step 6: Enabling and launching background daemons...${NC}"
systemctl daemon-reload
systemctl enable cerberus-server.service
systemctl enable cerberus-agent.service

echo -e "${YELLOW}[*] Starting Cerberus services...${NC}"
systemctl restart cerberus-server.service
systemctl restart cerberus-agent.service

echo -e "${GREEN}[+] Background services active, persisting, and crash-resilient.${NC}"

echo -e "\n${CYAN}${BOLD}========================================================================${NC}"
echo -e "${GREEN}${BOLD}  INSTALLATION SUCCESSFUL!${NC}"
echo -e "${CYAN}${BOLD}========================================================================${NC}"
echo -e "Cerberus is now fully operational in the background."
echo -e ""
echo -e "   - Run ${BOLD}cerberus-ctl${NC} anywhere in your terminal to open the TUI menu."
echo -e "   - Status can be verified via standard ${BOLD}systemctl status cerberus-agent${NC}"
echo -e ""
echo -e "Thank you for using Cerberus, designed and created by Joseph Sierengowski."
echo -e "${CYAN}${BOLD}========================================================================${NC}"
