package approvals

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gustawdaniel/antigravity-remote/pkg/models"
)

// discoveryTTL bounds how often /proc is scanned; ListSessions runs on every API request.
const discoveryTTL = 2 * time.Second

type Manager struct {
	approvals          map[string]*models.Approval
	sessions           map[string]*models.WorkspaceSession // Keyed by workspace name (e.g. "well-done", "brain", "antigravity")
	queuedMessages     map[string][]string                 // Keyed by workspace name
	recentUserMessages map[string][]models.UserMessageRecord // Keyed by workspace name
	mode               string                              // "auto" (default), "smart", "manual"
	lastDiscovery      time.Time
	mu                 sync.RWMutex
}

func NewManager() *Manager {
	m := &Manager{
		approvals:          make(map[string]*models.Approval),
		sessions:           make(map[string]*models.WorkspaceSession),
		queuedMessages:     make(map[string][]string),
		recentUserMessages: make(map[string][]models.UserMessageRecord),
		mode:               "auto", // FULL AUTO BY DEFAULT: Zero blocking, zero notification spam!
	}
	m.DiscoverRunningInstances()
	return m
}

func (m *Manager) GetMode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mode
}

func (m *Manager) SetMode(mode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch strings.ToLower(mode) {
	case "auto", "smart", "manual":
		m.mode = strings.ToLower(mode)
	default:
		m.mode = "auto"
	}
}

func (m *Manager) RecordUserMessage(workspace string, cascadeID string, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recentUserMessages[workspace] = append(m.recentUserMessages[workspace], models.UserMessageRecord{
		Workspace: workspace,
		CascadeID: cascadeID,
		Content:   content,
		Timestamp: time.Now(),
	})
	if len(m.recentUserMessages[workspace]) > 50 {
		m.recentUserMessages[workspace] = m.recentUserMessages[workspace][len(m.recentUserMessages[workspace])-50:]
	}
}

func (m *Manager) GetRecentUserMessages(workspace string, since time.Duration) []models.UserMessageRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cutoff := time.Now().Add(-since)
	var result []models.UserMessageRecord
	for _, msg := range m.recentUserMessages[workspace] {
		if msg.Timestamp.After(cutoff) {
			result = append(result, msg)
		}
	}
	return result
}

func (m *Manager) QueueMessage(workspace string, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queuedMessages[workspace] = append(m.queuedMessages[workspace], msg)
}

func (m *Manager) PopQueuedMessage(workspace string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	queue, ok := m.queuedMessages[workspace]
	if !ok || len(queue) == 0 {
		// Also check "all" or default fallback
		if qAll, okAll := m.queuedMessages["all"]; okAll && len(qAll) > 0 {
			msg := qAll[0]
			m.queuedMessages["all"] = qAll[1:]
			return msg, true
		}
		return "", false
	}
	msg := queue[0]
	m.queuedMessages[workspace] = queue[1:]
	return msg, true
}

func (m *Manager) CreateApproval(input models.AntigravityHookInput) *models.Approval {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateID()
	workspace := "default"
	var wsPath string
	if len(input.WorkspacePaths) > 0 {
		wsPath = input.WorkspacePaths[0]
		workspace = filepath.Base(wsPath)
	}

	app := &models.Approval{
		ID:             id,
		Workspace:      workspace,
		ConversationID: input.ConversationID,
		StepIdx:        input.StepIdx,
		CreatedAt:      time.Now(),
		DecisionChan:   make(chan models.HookDecisionOutput, 1),
	}

	if input.ToolCall != nil {
		app.ToolName = input.ToolCall.Name
		app.ToolArgs = input.ToolCall.Args
		extractToolDetails(app, input.ToolCall)
	}

	isCrit, critReason := isCriticalAction(app)
	app.IsCritical = isCrit

	// In "auto" (default) or "smart" without manual flag, EVERYTHING is auto-approved instantly!
	if m.mode == "auto" || m.mode == "smart" {
		app.Status = "allowed"
		app.AutoApproved = true
		now := time.Now()
		app.DecidedAt = &now
		app.DecisionReason = "Auto-approved"

		// Pre-populate decision chan so hook returns in 0ms without blocking!
		app.DecisionChan <- models.HookDecisionOutput{Decision: "allow"}

		// Session stays running
		m.sessions[workspace] = &models.WorkspaceSession{
			ID:             input.ConversationID,
			Workspace:      workspace,
			WorkspacePath:  wsPath,
			TranscriptPath: input.TranscriptPath,
			LastToolName:   app.ToolName,
			Status:         "running",
			LastActive:     now,
		}
	} else {
		// Manual mode only: wait for phone tap
		app.Status = "pending"
		if isCrit {
			app.Description = "⚠️ [CRITICAL] " + critReason + " — " + app.Description
		}

		m.sessions[workspace] = &models.WorkspaceSession{
			ID:             input.ConversationID,
			Workspace:      workspace,
			WorkspacePath:  wsPath,
			TranscriptPath: input.TranscriptPath,
			LastToolName:   app.ToolName,
			Status:         "waiting_approval",
			LastActive:     time.Now(),
		}
	}

	m.approvals[id] = app
	return app
}

func (m *Manager) Decide(id string, decision string, reason string) (*models.Approval, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, exists := m.approvals[id]
	if !exists {
		return nil, fmt.Errorf("approval %s not found", id)
	}

	if app.Status != "pending" {
		return app, fmt.Errorf("approval %s already resolved as %s", id, app.Status)
	}

	now := time.Now()
	app.DecidedAt = &now
	app.DecisionReason = reason

	var hookDecision models.HookDecisionOutput
	switch strings.ToLower(decision) {
	case "allow", "approve":
		app.Status = "allowed"
		hookDecision = models.HookDecisionOutput{Decision: "allow"}
	case "deny", "reject":
		app.Status = "denied"
		hookDecision = models.HookDecisionOutput{
			Decision: "deny",
			Reason:   reason,
		}
		if reason == "" {
			hookDecision.Reason = "Rejected by user via mobile device"
		}
	default:
		app.Status = "allowed"
		hookDecision = models.HookDecisionOutput{Decision: "allow"}
	}

	// Unblock hook listener
	select {
	case app.DecisionChan <- hookDecision:
	default:
	}

	// Update session status
	if sess, ok := m.sessions[app.Workspace]; ok {
		sess.Status = "running"
		sess.LastActive = now
	}

	return app, nil
}

func (m *Manager) UpdateSessionFromHook(input models.AntigravityHookInput, status string) *models.WorkspaceSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	workspace := "default"
	var wsPath string
	if len(input.WorkspacePaths) > 0 {
		wsPath = input.WorkspacePaths[0]
		workspace = filepath.Base(wsPath)
	}

	lastTool := ""
	if input.ToolCall != nil {
		lastTool = input.ToolCall.Name
	}

	sess := &models.WorkspaceSession{
		ID:             input.ConversationID,
		Workspace:      workspace,
		WorkspacePath:  wsPath,
		TranscriptPath: input.TranscriptPath,
		LastToolName:   lastTool,
		Status:         status,
		LastActive:     time.Now(),
	}
	m.sessions[workspace] = sess
	return sess
}

func (m *Manager) ListApprovals(workspace string) []*models.Approval {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Approval
	for _, a := range m.approvals {
		if workspace == "" || workspace == "all" || a.Workspace == workspace {
			result = append(result, a)
		}
	}
	return result
}

func (m *Manager) GetApproval(id string) (*models.Approval, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.approvals[id]
	return a, ok
}

func (m *Manager) refreshSessionStatus(s *models.WorkspaceSession) {
	if s.Status == "waiting_approval" {
		return
	}
	var lastActive time.Time = s.LastActive
	if s.TranscriptPath != "" {
		if info, err := os.Stat(s.TranscriptPath); err == nil {
			if info.ModTime().After(lastActive) {
				lastActive = info.ModTime()
			}
		}
	}
	// If no activity or file changes in the last 20 seconds, mark as idle
	if time.Since(lastActive) > 20*time.Second {
		s.Status = "idle"
	}
}

func (m *Manager) GetSession(workspace string) (*models.WorkspaceSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[workspace]
	if ok {
		m.refreshSessionStatus(s)
	}
	return s, ok
}

func (m *Manager) ListSessions() []*models.WorkspaceSession {
	m.DiscoverRunningInstances()

	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*models.WorkspaceSession
	for _, s := range m.sessions {
		m.refreshSessionStatus(s)
		result = append(result, s)
	}

	// Sort stably by workspace name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Workspace < result[j].Workspace
	})

	return result
}

func (m *Manager) DiscoverRunningInstances() {
	m.mu.Lock()
	if time.Since(m.lastDiscovery) < discoveryTTL {
		m.mu.Unlock()
		return
	}
	m.lastDiscovery = time.Now()
	m.mu.Unlock()

	entries, err := filepath.Glob("/proc/[0-9]*/cmdline")
	if err != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	activeWorkspaces := make(map[string]bool)

	for _, p := range entries {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		cmdStr := string(data)
		if !strings.Contains(cmdStr, "language_server_linux_x64") || !strings.Contains(cmdStr, "--workspace_id") {
			continue
		}

		parts := strings.Split(cmdStr, "\x00")
		for i, part := range parts {
			if part == "--workspace_id" && i+1 < len(parts) {
				wsID := parts[i+1]
				path := resolveWorkspacePath(wsID)
				wsName := filepath.Base(path)
				if wsName == "." || wsName == "/" || wsName == "" {
					wsName = wsID
				}

				activeWorkspaces[wsName] = true

				existing, exists := m.sessions[wsName]
				if !exists {
					convID, tPath := findLatestTranscriptForWorkspace(path)
					m.sessions[wsName] = &models.WorkspaceSession{
						ID:             convID,
						Workspace:      wsName,
						WorkspacePath:  path,
						TranscriptPath: tPath,
						Status:         "idle",
						LastActive:     time.Now(),
					}
				} else if existing.TranscriptPath == "" {
					convID, tPath := findLatestTranscriptForWorkspace(path)
					if convID != "" {
						existing.ID = convID
					}
					if tPath != "" {
						existing.TranscriptPath = tPath
					}
				}
			}
		}
	}

	// Clean up dead sessions that no longer have a running language_server
	for name := range m.sessions {
		if !activeWorkspaces[name] {
			delete(m.sessions, name)
		}
	}
}

func findLatestTranscriptForWorkspace(wsPath string) (string, string) {
	home, _ := os.UserHomeDir()
	brainDir := filepath.Join(home, ".gemini", "antigravity-ide", "brain")
	entries, err := os.ReadDir(brainDir)
	if err != nil {
		return "", ""
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
		// Read up to first 30KB to verify workspace
		file, err := os.Open(it.path)
		if err != nil {
			continue
		}
		buf := make([]byte, 30000)
		n, _ := file.Read(buf)
		file.Close()

		if n > 0 && strings.Contains(string(buf[:n]), wsPath) {
			return it.id, it.path
		}
	}

	if len(items) > 0 {
		return items[0].id, items[0].path
	}
	return "", ""
}

func isCriticalAction(app *models.Approval) (bool, string) {
	if app.ToolName == "run_command" {
		cmd := strings.ToLower(app.CommandLine)
		if strings.Contains(cmd, "rm -rf") || strings.Contains(cmd, "rm -fr") || strings.Contains(cmd, "rm -r ") {
			return true, "Destructive file removal (rm -r)"
		}
		if strings.Contains(cmd, "git reset --hard") {
			return true, "Destructive git reset --hard"
		}
		if strings.Contains(cmd, "git clean -f") {
			return true, "Destructive git clean"
		}
		if strings.Contains(cmd, "drop database") || strings.Contains(cmd, "drop table") {
			return true, "Database destructive drop operation"
		}
		if strings.Contains(cmd, "mkfs") || strings.Contains(cmd, "dd if=") {
			return true, "Raw disk format/overwrite operation"
		}
		return false, ""
	}

	if app.ToolName == "replace_file_content" || app.ToolName == "write_to_file" || app.ToolName == "multi_replace_file_content" {
		path := strings.ToLower(app.FilePath)
		if strings.HasPrefix(path, "/etc") || strings.Contains(path, "/.ssh") {
			return true, "Modification of system / SSH credentials"
		}
		return false, ""
	}

	return false, ""
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

func extractToolDetails(app *models.Approval, tc *models.ToolCall) {
	args := tc.Args
	if args == nil {
		return
	}

	if cmd, ok := args["CommandLine"].(string); ok {
		app.CommandLine = cmd
	}
	if file, ok := args["TargetFile"].(string); ok {
		app.FilePath = file
	}
	if desc, ok := args["Description"].(string); ok {
		app.Description = desc
	}

	switch tc.Name {
	case "replace_file_content":
		target, _ := args["TargetContent"].(string)
		repl, _ := args["ReplacementContent"].(string)
		if target != "" || repl != "" {
			var sb strings.Builder
			for _, line := range strings.Split(target, "\n") {
				sb.WriteString("- ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
			for _, line := range strings.Split(repl, "\n") {
				sb.WriteString("+ ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
			app.Diff = sb.String()
		}
	case "write_to_file":
		content, _ := args["CodeContent"].(string)
		if content != "" {
			lines := strings.Split(content, "\n")
			if len(lines) > 20 {
				app.Diff = "+ " + strings.Join(lines[:20], "\n+ ") + fmt.Sprintf("\n... (+ %d more lines)", len(lines)-20)
			} else {
				app.Diff = "+ " + strings.Join(lines, "\n+ ")
			}
		}
	}
}

func generateID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}
