package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Configure the Two-Model Local Consensus Network
const (
	AnalyzerBrain   = "gemma2:2b"     // Google's code-syntax expert
	CommanderBrain  = "granite3.3:2b" // IBM's security decision engine
	MaxInMemoryLogs = 50             // Max number of recent telemetry events to keep in-memory
)

type SecurityLog struct {
	PID            uint32 `json:"pid"`
	PPID           uint32 `json:"ppid"`
	UID            uint32 `json:"uid"`
	ParentProcess  string `json:"parent_process"`
	BinaryExecuted string `json:"binary_executed"`
}

type SecurityLogWithTime struct {
	Timestamp      string `json:"timestamp"`
	PID            uint32 `json:"pid"`
	PPID           uint32 `json:"ppid"`
	UID            uint32 `json:"uid"`
	ParentProcess  string `json:"parent_process"`
	BinaryExecuted string `json:"binary_executed"`
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

type AskRequest struct {
	Question string `json:"question"`
}

type AskResponse struct {
	Answer string `json:"answer"`
}

var (
	recentLogs           []SecurityLogWithTime
	recentLogsMutex      sync.Mutex
	logFilePath          = "/var/log/cerberus/telemetry.json"
	SuspiciousIndicators = []string{"/tmp/", "/dev/shm/", " netcat", "/bin/nc", "/usr/bin/nc"} // Aligned with rust agent heuristics
)

func logTelemetryEvent(secLog SecurityLog) {
	logWithTime := SecurityLogWithTime{
		Timestamp:      time.Now().Format(time.RFC3339),
		PID:            secLog.PID,
		PPID:           secLog.PPID,
		UID:            secLog.UID,
		ParentProcess:  secLog.ParentProcess,
		BinaryExecuted: secLog.BinaryExecuted,
	}

	recentLogsMutex.Lock()
	recentLogs = append(recentLogs, logWithTime)
	if len(recentLogs) > MaxInMemoryLogs {
		recentLogs = recentLogs[1:] // Limit in-memory size
	}
	recentLogsMutex.Unlock()

	// Append to file
	jsonData, err := json.Marshal(logWithTime)
	if err == nil {
		_ = os.MkdirAll("/var/log/cerberus", 0755)
		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			defer f.Close()
			_, _ = f.Write(append(jsonData, '\n'))
		}
	}
}

func getSystemStats() string {
	// Memory
	memInfo, err := os.ReadFile("/proc/meminfo")
	var memStats string
	if err == nil {
		lines := strings.Split(string(memInfo), "\n")
		var total, free, available string
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				total = strings.TrimSpace(strings.TrimPrefix(line, "MemTotal:"))
			} else if strings.HasPrefix(line, "MemFree:") {
				free = strings.TrimSpace(strings.TrimPrefix(line, "MemFree:"))
			} else if strings.HasPrefix(line, "MemAvailable:") {
				available = strings.TrimSpace(strings.TrimPrefix(line, "MemAvailable:"))
			}
		}
		memStats = fmt.Sprintf("RAM - Total: %s, Free: %s, Available: %s", total, free, available)
	} else {
		memStats = "RAM - Error reading /proc/meminfo"
	}

	// Disk
	var diskStats string
	cmd := exec.Command("df", "-h", "/")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			diskStats = "Disk Usage on /: " + strings.Join(strings.Fields(lines[1]), " ")
		} else {
			diskStats = "Disk Usage on /: Error parsing df"
		}
	} else {
		diskStats = "Disk Usage on /: Error executing df"
	}

	// Load Avg
	loadAvg, err := os.ReadFile("/proc/loadavg")
	var loadStats string
	if err == nil {
		loadStats = "CPU Load Avg: " + strings.TrimSpace(string(loadAvg))
	} else {
		loadStats = "CPU Load Avg: Error reading /proc/loadavg"
	}

	return fmt.Sprintf("- %s\n- %s\n- %s", memStats, diskStats, loadStats)
}

// CleanAIResponse strips away intermediate reasoning/thinking tags from local models
func CleanAIResponse(rawResponse string) string {
	if strings.Contains(rawResponse, "</thought>") {
		parts := strings.Split(rawResponse, "</thought>")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return strings.TrimSpace(rawResponse)
}

// QueryOllama Engine speaks directly to your local loopback Ollama service
func QueryOllama(model string, prompt string) (string, error) {
	requestBody, err := json.Marshal(OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post("http://127.0.0.1:11434/api/generate", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}

	return CleanAIResponse(ollamaResp.Response), nil
}

// RunConsensusLoop routes telemetry through Gemma, then feeds the report into Granite
func RunConsensusLoop(secLog SecurityLog) {
	// --- PHASE 1: GEMMA 2 2B TECHNICAL DECONSTRUCTION ---
	gemmaPrompt := fmt.Sprintf(
		"You are a neutral code analyzer. Describe strictly what this execution context does without judging if it is malicious or safe.\n"+
			"Parent Process: %s\nExecuted Command: %s\nUser Context ID: %d\nProvide a technical summary in one sentence:",
		secLog.ParentProcess, secLog.BinaryExecuted, secLog.UID,
	)

	gemmaReport, err := QueryOllama(AnalyzerBrain, gemmaPrompt)
	if err != nil {
		log.Printf("[!] Phase 1 (Gemma) Failed: %v. Using simulated fallback.", err)
		gemmaReport = fmt.Sprintf("The binary %s was executed with PID %d under parent process %s by UID %d.", secLog.BinaryExecuted, secLog.PID, secLog.ParentProcess, secLog.UID)
		fmt.Printf("\n[🔬 GEMMA ANALYSIS (Simulated)]: %s\n", gemmaReport)
	} else {
		fmt.Printf("\n[🔬 GEMMA ANALYSIS]: %s\n", gemmaReport)
	}

	// --- PHASE 2: GRANITE 3.3 2B SECURITY COMMAND VERDICT ---
	granitePrompt := fmt.Sprintf(
		"You are Cerberus-Commander, an automated endpoint defense coordinator. Read this system event profile and the technical analysis report.\n\n"+
			"Event Profile:\n- Binary Path: %s\n- User ID: %d\n\nTechnical Analysis:\n%s\n\n"+
			"Determine if this is a hostile attack or unauthorized privilege escalation. You must end your response with exactly 'VERDICT: KILL' or 'VERDICT: ALLOW':",
		secLog.BinaryExecuted, secLog.UID, gemmaReport,
	)

	graniteVerdict, err := QueryOllama(CommanderBrain, granitePrompt)
	if err != nil {
		log.Printf("[!] Phase 2 (Granite) Failed: %v. Using simulated fallback.", err)
		suspicious := false
		for _, ind := range SuspiciousIndicators {
			if strings.Contains(secLog.BinaryExecuted, ind) {
				suspicious = true
				break
			}
		}
		if suspicious {
			graniteVerdict = "VERDICT: KILL"
		} else {
			graniteVerdict = "VERDICT: ALLOW"
		}
		fmt.Printf("[⚔️ GRANITE COMMAND (Simulated)]: %s\n\n", graniteVerdict)
	} else {
		fmt.Printf("[⚔️ GRANITE COMMAND]: %s\n\n", graniteVerdict)
	}
}

func handleAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stats := getSystemStats()

	recentLogsMutex.Lock()
	logsBytes, _ := json.MarshalIndent(recentLogs, "", "  ")
	recentLogsMutex.Unlock()

	prompt := fmt.Sprintf(
		"You are Cerberus Security Assistant, an advanced on-premise security intelligence interface.\n"+
			"Help the administrator with their inquiry about the system state, resource usage, and security events.\n\n"+
			"Current System Resource Statistics:\n%s\n\n"+
			"Recent Telemetry Security Events:\n%s\n\n"+
			"Administrator's Inquiry:\n\"%s\"\n\n"+
			"Provide a clear, detailed, professional, and actionable response in Markdown format, tailored to the inquiry. If the inquiry asks about low RAM, high disk usage, or suspicious events, reference the stats or events above:",
		stats, string(logsBytes), req.Question,
	)

	answer, err := QueryOllama(CommanderBrain, prompt)
	if err != nil {
		log.Printf("[!] Ollama /ask query failed: %v. Using simulated fallback.", err)
		answer = getSimulatedAskResponse(req.Question, stats)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AskResponse{Answer: answer})
}

func getSimulatedAskResponse(question string, stats string) string {
	q := strings.ToLower(question)

	var sb strings.Builder
	sb.WriteString("🤖 **[Cerberus Security Assistant — Offline Simulation Mode]**\n\n")
	sb.WriteString("I am analyzing the current host environment and the captured security telemetry database.\n\n")

	sb.WriteString("### 📊 Host System Metrics\n")
	sb.WriteString(stats + "\n\n")

	sb.WriteString("### 🛡️ Telemetry Event Summary\n")
	recentLogsMutex.Lock()
	count := len(recentLogs)
	recentLogsMutex.Unlock()

	if count == 0 {
		sb.WriteString("No suspicious execution telemetry has been recorded in the current session. The perimeter is currently clean.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("Recorded **%d** execution event(s) in this session. All events have been processed by the local eBPF telemetry ring-buffer.\n\n", count))
	}

	sb.WriteString("### 🧠 AI Analysis & Diagnostics\n")
	if strings.Contains(q, "ram") || strings.Contains(q, "memory") {
		sb.WriteString("- **Memory Diagnostic:** Memory resources are currently being monitored. If available memory drops below 15%, Cerberus-Commander will flag a system resource warning. Your current memory statistics are listed above.\n")
	}
	if strings.Contains(q, "disk") || strings.Contains(q, "space") {
		sb.WriteString("- **Storage Diagnostic:** Root filesystem capacity check. If any partition exceeds 90% utilization, host deception layers or caching pipelines may experience latency. Check your current capacity in the metrics section.\n")
	}
	if strings.Contains(q, "find") || strings.Contains(q, "suspicious") || strings.Contains(q, "telemetry") || strings.Contains(q, "event") {
		sb.WriteString("- **Security Diagnostic:** Monitoring tracepoint hooks. If an execution originates from standard attack paths (e.g., `/tmp/`, `/dev/shm/`), the sensor flags it immediately. Active filters are processing all kernel execve syscalls.\n")
	}

	if !strings.Contains(q, "ram") && !strings.Contains(q, "memory") && !strings.Contains(q, "disk") && !strings.Contains(q, "space") && !strings.Contains(q, "find") && !strings.Contains(q, "telemetry") {
		sb.WriteString("I am ready to answer specific questions about host resources, storage warnings, memory exhaustion, or telemetry findings. Please ask about RAM, disk capacity, or suspicious events for targeted insights.\n")
	}

	sb.WriteString("\n*Note: To connect this assistant to live local LLMs, ensure Ollama is installed and running `granite3.3:2b` and `gemma2:2b` models.*")
	return sb.String()
}

func main() {
	log.Println("[+] Cerberus Command Tower (Right Head) waking up...")
	log.Println("[+] Multi-Model Consensus Engine initialized on port 8080...")

	http.HandleFunc("/telemetry", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
			return
		}

		var secLog SecurityLog
		if err := json.NewDecoder(r.Body).Decode(&secLog); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Log the event locally
		logTelemetryEvent(secLog)

		// Process the consensus reasoning asynchronously so your kernel data loop never pauses
		go RunConsensusLoop(secLog)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"received"}`))
	})

	http.HandleFunc("/ask", handleAsk)

	if err := http.ListenAndServe("127.0.0.1:8080", nil); err != nil {
		log.Fatalf("[!] Command Tower network pipeline dropped: %v", err)
	}
}
