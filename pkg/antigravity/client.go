package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LanguageServerInstance struct {
	PID           int       `json:"pid"`
	WorkspaceID   string    `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	WorkspacePath string    `json:"workspace_path"`
	CSRFToken     string    `json:"csrf_token"`
	Port          int       `json:"port"`
	LastChecked   time.Time `json:"last_checked"`
}

type Client struct {
	instances map[string]*LanguageServerInstance // Keyed by WorkspaceName
	mu        sync.RWMutex
	client    *http.Client
}

func NewClient() *Client {
	return &Client{
		instances: make(map[string]*LanguageServerInstance),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DiscoverInstances finds all running Antigravity language_server processes and their HTTP Connect ports.
func (c *Client) DiscoverInstances() ([]*LanguageServerInstance, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := filepath.Glob("/proc/[0-9]*/cmdline")
	if err != nil {
		return nil, fmt.Errorf("failed to scan /proc: %w", err)
	}

	var discovered []*LanguageServerInstance

	// Run ss -tlpn once to find listening sockets
	ssCmd := exec.Command("ss", "-tlpn")
	ssOut, err := ssCmd.Output()
	ssText := string(ssOut)

	for _, p := range entries {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		cmdStr := string(data)
		if !strings.Contains(cmdStr, "language_server_linux_x64") || !strings.Contains(cmdStr, "--csrf_token") {
			continue
		}

		parts := strings.Split(cmdStr, "\x00")
		var csrfToken, wsID string
		for i, part := range parts {
			if part == "--csrf_token" && i+1 < len(parts) {
				csrfToken = parts[i+1]
			}
			if part == "--workspace_id" && i+1 < len(parts) {
				wsID = parts[i+1]
			}
		}

		if csrfToken == "" {
			continue
		}

		// Extract PID
		dirParts := strings.Split(p, string(filepath.Separator))
		if len(dirParts) < 3 {
			continue
		}
		pid, err := strconv.Atoi(dirParts[2])
		if err != nil {
			continue
		}

		wsPath := resolveWorkspacePath(wsID)
		wsName := filepath.Base(wsPath)
		if wsName == "." || wsName == "/" || wsName == "" {
			wsName = wsID
		}

		// Find listening ports for this pid
		ports := findListeningPortsForPID(ssText, pid)
		if len(ports) == 0 {
			continue
		}

		// Probe ports for HTTP Connect / Heartbeat
		activePort := probeConnectPort(ports, csrfToken)
		if activePort <= 0 {
			continue
		}

		inst := &LanguageServerInstance{
			PID:           pid,
			WorkspaceID:   wsID,
			WorkspaceName: wsName,
			WorkspacePath: wsPath,
			CSRFToken:     csrfToken,
			Port:          activePort,
			LastChecked:   time.Now(),
		}

		discovered = append(discovered, inst)
		c.instances[wsName] = inst
		if wsPath != "" {
			c.instances[wsPath] = inst
		}
		log.Printf("[Antigravity Client] Discovered workspace [%s] (PID %d, Port %d, Path: %s)", wsName, pid, activePort, wsPath)
	}

	return discovered, nil
}

// GetInstanceForWorkspace returns an active language server instance for a given workspace.
func (c *Client) GetInstanceForWorkspace(workspace string, wsPath string) (*LanguageServerInstance, error) {
	c.mu.RLock()
	inst, ok := c.instances[workspace]
	if !ok && wsPath != "" {
		inst, ok = c.instances[wsPath]
	}
	c.mu.RUnlock()

	// If found, check if still alive
	if ok && inst != nil {
		if c.checkHeartbeat(inst) {
			return inst, nil
		}
	}

	// Rescan
	discovered, err := c.DiscoverInstances()
	if err != nil {
		return nil, err
	}

	for _, d := range discovered {
		if d.WorkspaceName == workspace || (wsPath != "" && d.WorkspacePath == wsPath) {
			return d, nil
		}
	}

	// Fallback: If only 1 instance is running, use it
	if len(discovered) == 1 {
		return discovered[0], nil
	}

	return nil, fmt.Errorf("no active Antigravity language server found for workspace '%s'", workspace)
}

func (c *Client) checkHeartbeat(inst *LanguageServerInstance) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/exa.language_server_pb.LanguageServerService/Heartbeat", inst.Port)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-codeium-csrf-token", inst.CSRFToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// StartCascade creates a new Cascade conversation session in the workspace.
func (c *Client) StartCascade(inst *LanguageServerInstance, wsPath string) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/exa.language_server_pb.LanguageServerService/StartCascade", inst.Port)

	payload := map[string]interface{}{
		"source":         1,    // CORTEX_TRAJECTORY_SOURCE_CASCADE_CLIENT
		"requestedModel": 1318, // MODEL_PLACEHOLDER_M318
		"workspaceUris":  []string{"file://" + wsPath},
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-codeium-csrf-token", inst.CSRFToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("StartCascade returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		CascadeID string `json:"cascadeId"`
	}
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", fmt.Errorf("failed to parse StartCascade response: %w", err)
	}

	log.Printf("[Antigravity Client] Started new cascade %s for workspace %s", res.CascadeID, inst.WorkspaceName)
	return res.CascadeID, nil
}

// SendMessage dispatches a user prompt directly into the Antigravity conversation and wakes up the agent.
// The returned notice is non-empty when the cascade had to be unstuck before the prompt could be delivered.
func (c *Client) SendMessage(workspace string, wsPath string, cascadeID string, prompt string) (string, string, error) {
	inst, err := c.GetInstanceForWorkspace(workspace, wsPath)
	if err != nil {
		return "", "", err
	}

	if wsPath == "" {
		wsPath = inst.WorkspacePath
	}

	// 1. If cascadeID is empty, try to resolve latest active conversation from disk
	if cascadeID == "" {
		cascadeID = FindLatestCascadeID(wsPath)
	}

	// 2. If still empty, start a new cascade
	if cascadeID == "" {
		newID, err := c.StartCascade(inst, wsPath)
		if err != nil {
			return "", "", fmt.Errorf("failed to start cascade: %w", err)
		}
		cascadeID = newID
	}

	// 3. A cascade hanging on a step never picks up new messages (they pile up as "undelivered"),
	// so interrupt it first.
	notice := c.UnstickIfWedged(inst, cascadeID)

	// 4. Send message RPC
	err = c.sendUserCascadeMessageRPC(inst, cascadeID, prompt)
	if err != nil {
		log.Printf("[Antigravity Client] SendUserCascadeMessage failed for cascade %s: %v. Starting fresh cascade...", cascadeID, err)
		// If trajectory was closed or not found, create new cascade and retry
		newID, startErr := c.StartCascade(inst, wsPath)
		if startErr != nil {
			return "", "", fmt.Errorf("retry failed to start cascade: %w", startErr)
		}
		cascadeID = newID
		err = c.sendUserCascadeMessageRPC(inst, cascadeID, prompt)
		if err != nil {
			return "", "", fmt.Errorf("retry SendUserCascadeMessage failed: %w", err)
		}
	}

	log.Printf("[Antigravity Client] Successfully sent prompt to Antigravity Cascade [%s] in %s", cascadeID, inst.WorkspaceName)

	// Focus/refresh conversation in Antigravity desktop IDE window so user sees it live
	go func() {
		if focusErr := c.SmartFocusConversation(inst, cascadeID); focusErr != nil {
			log.Printf("[Antigravity Client] SmartFocusConversation notice: %v", focusErr)
		}
	}()

	return cascadeID, notice, nil
}

// SmartFocusConversation commands the Antigravity desktop IDE to focus and render the conversation.
func (c *Client) SmartFocusConversation(inst *LanguageServerInstance, cascadeID string) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/exa.language_server_pb.LanguageServerService/SmartFocusConversation", inst.Port)
	payload := map[string]interface{}{
		"cascadeId": cascadeID,
	}
	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-codeium-csrf-token", inst.CSRFToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) sendUserCascadeMessageRPC(inst *LanguageServerInstance, cascadeID string, prompt string) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/exa.language_server_pb.LanguageServerService/SendUserCascadeMessage", inst.Port)

	payload := map[string]interface{}{
		"cascadeId": cascadeID,
		"cascadeConfig": map[string]interface{}{
			"plannerConfig": map[string]interface{}{
				"planModel": 1318, // MODEL_PLACEHOLDER_M318
			},
		},
		"items": []map[string]interface{}{
			{
				"text": prompt,
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-codeium-csrf-token", inst.CSRFToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}

func findListeningPortsForPID(ssText string, pid int) []int {
	var ports []int
	re := regexp.MustCompile(fmt.Sprintf(`127\.0\.0\.1:(\d+).*?pid=%d,`, pid))
	matches := re.FindAllStringSubmatch(ssText, -1)
	for _, m := range matches {
		if len(m) > 1 {
			p, err := strconv.Atoi(m[1])
			if err == nil {
				ports = append(ports, p)
			}
		}
	}
	return ports
}

func probeConnectPort(ports []int, csrfToken string) int {
	client := &http.Client{Timeout: 300 * time.Millisecond}
	for _, port := range ports {
		url := fmt.Sprintf("http://127.0.0.1:%d/exa.language_server_pb.LanguageServerService/Heartbeat", port)
		req, err := http.NewRequest("POST", url, bytes.NewReader([]byte("{}")))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-codeium-csrf-token", csrfToken)

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return port
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	return -1
}

func FindLatestCascadeID(wsPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	brainDir := filepath.Join(home, ".gemini", "antigravity-ide", "brain")
	entries, err := os.ReadDir(brainDir)
	if err != nil {
		return ""
	}

	type item struct {
		id      string
		modTime time.Time
		path    string
	}
	var items []item
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(brainDir, e.Name(), ".system_generated", "logs", "transcript.jsonl")
		info, err := os.Stat(p)
		if err == nil {
			items = append(items, item{id: e.Name(), modTime: info.ModTime(), path: p})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime.After(items[j].modTime)
	})

	for _, it := range items {
		file, err := os.Open(it.path)
		if err != nil {
			continue
		}
		buf := make([]byte, 30000)
		n, _ := file.Read(buf)
		file.Close()

		if n > 0 && strings.Contains(string(buf[:n]), wsPath) {
			return it.id
		}
	}

	if len(items) > 0 {
		return items[0].id
	}
	return ""
}

func resolveWorkspacePath(wsID string) string {
	raw := strings.TrimPrefix(wsID, "file_")
	direct := "/" + strings.ReplaceAll(raw, "_", "/")
	if _, err := os.Stat(direct); err == nil {
		return direct
	}

	home, _ := os.UserHomeDir()
	proDir := filepath.Join(home, "pro")
	if entries, err := os.ReadDir(proDir); err == nil {
		for _, e := range entries {
			sanitized := strings.ReplaceAll(e.Name(), "-", "_")
			if strings.HasSuffix(raw, sanitized) {
				return filepath.Join(proDir, e.Name())
			}
		}
	}

	expDir := filepath.Join(home, "exp")
	if entriesExp, err := os.ReadDir(expDir); err == nil {
		for _, e := range entriesExp {
			sanitized := strings.ReplaceAll(e.Name(), "-", "_")
			if strings.HasSuffix(raw, sanitized) {
				return filepath.Join(expDir, e.Name())
			}
		}
	}

	return direct
}

// StuckThreshold is how long a single step may stay RUNNING before the cascade is considered wedged.
// Blocking run_command steps go async after WaitMsBeforeAsync (seconds), so minutes means the IDE lost the process.
const StuckThreshold = 5 * time.Minute

// CascadeState is a lightweight snapshot of a cascade taken from the language server.
type CascadeState struct {
	Status          string    `json:"status"` // CASCADE_RUN_STATUS_IDLE / CASCADE_RUN_STATUS_RUNNING
	StepCount       int       `json:"stepCount"`
	LastStepType    string    `json:"lastStepType"`
	LastStepStatus  string    `json:"lastStepStatus"`
	LastStepStarted time.Time `json:"lastStepStarted"`
	LastStepSummary string    `json:"lastStepSummary"`
}

func (s *CascadeState) Running() bool {
	return s.Status == "CASCADE_RUN_STATUS_RUNNING"
}

// StuckFor returns how long the last step has been hanging, or 0 if it isn't hanging.
func (s *CascadeState) StuckFor() time.Duration {
	if !s.Running() || s.LastStepStarted.IsZero() {
		return 0
	}
	if s.LastStepStatus != "CORTEX_STEP_STATUS_RUNNING" && s.LastStepStatus != "CORTEX_STEP_STATUS_PENDING" {
		return 0
	}
	if d := time.Since(s.LastStepStarted); d > StuckThreshold {
		return d
	}
	return 0
}

func (c *Client) rpc(inst *LanguageServerInstance, method string, payload interface{}, out interface{}) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/exa.language_server_pb.LanguageServerService/%s", inst.Port, method)
	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-codeium-csrf-token", inst.CSRFToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s HTTP %d: %s", method, resp.StatusCode, string(respBytes))
	}
	if out != nil {
		return json.Unmarshal(respBytes, out)
	}
	return nil
}

// GetCascadeState reads run status and the last step without pulling the whole (tens of MB) trajectory.
func (c *Client) GetCascadeState(inst *LanguageServerInstance, cascadeID string) (*CascadeState, error) {
	var all struct {
		TrajectorySummaries map[string]struct {
			Status    string `json:"status"`
			StepCount int    `json:"stepCount"`
		} `json:"trajectorySummaries"`
	}
	if err := c.rpc(inst, "GetAllCascadeTrajectories", map[string]interface{}{}, &all); err != nil {
		return nil, err
	}
	sum, ok := all.TrajectorySummaries[cascadeID]
	if !ok {
		return nil, fmt.Errorf("cascade %s not found in language server", cascadeID)
	}
	state := &CascadeState{Status: sum.Status, StepCount: sum.StepCount}
	if sum.StepCount == 0 {
		return state, nil
	}

	var steps struct {
		Steps []struct {
			Type     string `json:"type"`
			Status   string `json:"status"`
			Metadata struct {
				CreatedAt   time.Time `json:"createdAt"`
				StartedAt   time.Time `json:"startedAt"`
				ToolSummary string    `json:"toolSummary"`
			} `json:"metadata"`
		} `json:"steps"`
	}
	if err := c.rpc(inst, "GetCascadeTrajectorySteps", map[string]interface{}{
		"cascadeId":  cascadeID,
		"stepOffset": sum.StepCount - 1,
	}, &steps); err != nil {
		return state, nil
	}
	if n := len(steps.Steps); n > 0 {
		last := steps.Steps[n-1]
		state.LastStepType = strings.TrimPrefix(last.Type, "CORTEX_STEP_TYPE_")
		state.LastStepStatus = last.Status
		state.LastStepStarted = last.Metadata.StartedAt
		if state.LastStepStarted.IsZero() {
			state.LastStepStarted = last.Metadata.CreatedAt
		}
		state.LastStepSummary = last.Metadata.ToolSummary
	}
	return state, nil
}

// CancelCascade interrupts the current invocation (same as pressing Stop in the IDE).
func (c *Client) CancelCascade(inst *LanguageServerInstance, cascadeID string) error {
	if err := c.rpc(inst, "CancelCascadeInvocation", map[string]interface{}{"cascadeId": cascadeID}, nil); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		st, err := c.GetCascadeState(inst, cascadeID)
		if err == nil && !st.Running() {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("cascade %s still running after cancel", cascadeID)
}

// UnstickIfWedged cancels the cascade when its last step has been hanging longer than StuckThreshold.
// Returns a human-readable notice when it had to intervene.
func (c *Client) UnstickIfWedged(inst *LanguageServerInstance, cascadeID string) string {
	st, err := c.GetCascadeState(inst, cascadeID)
	if err != nil {
		return ""
	}
	stuck := st.StuckFor()
	if stuck == 0 {
		return ""
	}
	what := st.LastStepType
	if st.LastStepSummary != "" {
		what += " (" + st.LastStepSummary + ")"
	}
	log.Printf("[Antigravity Client] Cascade %s wedged on %s for %s. Cancelling invocation.", cascadeID, what, stuck.Round(time.Second))
	if err := c.CancelCascade(inst, cascadeID); err != nil {
		log.Printf("[Antigravity Client] Cancel failed for %s: %v", cascadeID, err)
		return ""
	}
	return fmt.Sprintf("Agent wisiał %s na kroku %s — przerwałem go.", stuck.Round(time.Minute), what)
}
