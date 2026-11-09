package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gustawdaniel/antigravity-remote/pkg/antigravity"
	"github.com/gustawdaniel/antigravity-remote/pkg/approvals"
	"github.com/gustawdaniel/antigravity-remote/pkg/hub"
	"github.com/gustawdaniel/antigravity-remote/pkg/models"
	"github.com/gustawdaniel/antigravity-remote/pkg/transcript"
)

var (
	finishedMu       sync.Mutex
	lastFinishedTime = make(map[string]time.Time)
)

func broadcastAgentFinished(appHub *hub.Hub, ws string, source string) {
	finishedMu.Lock()
	last, ok := lastFinishedTime[ws]
	if ok && time.Since(last) < 10*time.Second {
		finishedMu.Unlock()
		log.Printf("[Finished] Suppressing duplicate finish notification for [%s] (source: %s, elapsed: %v)", ws, source, time.Since(last))
		return
	}
	lastFinishedTime[ws] = time.Now()
	finishedMu.Unlock()

	appHub.Broadcast(models.WSMessage{
		Type: "agent_finished",
		Payload: map[string]interface{}{
			"workspace": ws,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
	log.Printf("[Finished] Agent in [%s] finished task (source: %s). Broadcasted agent_finished.", ws, source)
	sendNtfy("Antigravity Gotowe", fmt.Sprintf("Antigravity w [%s] skończył pracę i odpowiedział.", ws), "white_check_mark,robot")
}

func main() {
	port := flag.Int("port", 8765, "HTTP server port")
	staticDir := flag.String("static", "./frontend/build", "Path to frontend static assets")
	certDir := flag.String("certs", "./certs", "Path to TLS certificates directory")
	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" {
		fmt.Sscanf(envPort, "%d", port)
	}

	appHub := hub.NewHub()
	go appHub.Run()

	mgr := approvals.NewManager()

	agyClient := antigravity.NewClient()
	go func() {
		// Initial discovery in background
		agyClient.DiscoverInstances()
	}()

	mux := http.NewServeMux()

	// ----------------------------------------------------
	// HOOK ENDPOINTS (Called by Antigravity hook scripts)
	// ----------------------------------------------------

	// PreToolUse: Auto-approve immediately without blocking!
	mux.HandleFunc("POST /api/hook/pre-tool", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var input models.AntigravityHookInput
		if err := json.Unmarshal(body, &input); err != nil {
			log.Printf("[Hook] Unmarshal error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(models.HookDecisionOutput{Decision: "allow"})
			return
		}

		app := mgr.CreateApproval(input)
		log.Printf("[Hook] PreToolUse [%s] %s (Step %d, ID: %s, AutoApproved: %v)", app.Workspace, app.ToolName, app.StepIdx, app.ID, app.AutoApproved)

		output := models.HookDecisionOutput{Decision: "allow"}

		// Check if user submitted a message from phone that should be injected
		if msg, ok := mgr.PopQueuedMessage(app.Workspace); ok {
			output.InjectSteps = []models.InjectStep{
				{UserMessage: msg},
			}
			log.Printf("[Hook] Injected user message for %s into PreToolUse: %s", app.Workspace, msg)
		}

		// Update session status and broadcast quietly
		sess := mgr.UpdateSessionFromHook(input, "running")
		appHub.Broadcast(models.WSMessage{
			Type:    "session_updated",
			Payload: sess,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(output)
	})

	// PostToolUse: Tool completed
	mux.HandleFunc("POST /api/hook/post-tool", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var input models.AntigravityHookInput
		if err := json.Unmarshal(body, &input); err == nil {
			sess := mgr.UpdateSessionFromHook(input, "running")
			appHub.Broadcast(models.WSMessage{
				Type:    "session_updated",
				Payload: sess,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	// Stop: Execution loop paused / ended
	mux.HandleFunc("POST /api/hook/stop", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var input models.AntigravityHookInput
		var output struct {
			Decision string `json:"decision,omitempty"`
			Reason   string `json:"reason,omitempty"`
		}

		if err := json.Unmarshal(body, &input); err == nil {
			sess := mgr.UpdateSessionFromHook(input, "idle")
			appHub.Broadcast(models.WSMessage{
				Type:    "session_updated",
				Payload: sess,
			})

			// If there's a queued message from mobile, continue turn!
			ws := sess.Workspace
			if msg, ok := mgr.PopQueuedMessage(ws); ok {
				output.Decision = "continue"
				output.Reason = msg
				log.Printf("[Hook] Stop continuation dispatched for %s: %s", ws, msg)
			} else {
				if isAgentReallyFinished(input.TranscriptPath) {
					broadcastAgentFinished(appHub, ws, "hook")
				} else {
					log.Printf("[Hook] Agent in [%s] stop hook triggered, but agent is still executing tools/generating answer. Skipping premature agent_finished notification.", ws)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(output)
	})

	// ----------------------------------------------------
	// REST API (Called by SvelteKit Mobile PWA)
	// ----------------------------------------------------

	// Send message/prompt to agent from mobile phone
	mux.HandleFunc("POST /api/sessions/{workspace}/message", func(w http.ResponseWriter, r *http.Request) {
		workspace := r.PathValue("workspace")
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
			http.Error(w, "message required", http.StatusBadRequest)
			return
		}

		trimmed := strings.TrimSpace(req.Message)
		log.Printf("[API] Mobile message for [%s]: %s", workspace, trimmed)

		// 1. Resolve workspace directory and active cascade ID
		var wsPath string
		var activeCascadeID string
		if sess, ok := mgr.GetSession(workspace); ok {
			wsPath = sess.WorkspacePath
			activeCascadeID = sess.ID
		} else {
			sessions := mgr.ListSessions()
			for _, s := range sessions {
				if s.Workspace == workspace {
					wsPath = s.WorkspacePath
					activeCascadeID = s.ID
					break
				}
			}
		}

		if wsPath == "" {
			home, _ := os.UserHomeDir()
			wsPath = filepath.Join(home, "pro", workspace)
		}

		// 2. Dispatch directly to Antigravity Language Server
		cascadeID, notice, err := agyClient.SendMessage(workspace, wsPath, activeCascadeID, trimmed)
		if err != nil {
			log.Printf("[API] Failed to dispatch prompt directly for [%s]: %v. Falling back to hook queue.", workspace, err)
			mgr.QueueMessage(workspace, trimmed)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":   false,
				"workspace": workspace,
				"error":     fmt.Sprintf("Brak aktywnego okna Antigravity dla '%s'. Upewnij się, że projekt jest otwarty w IDE na komputerze.", workspace),
				"message":   trimmed,
			})
			return
		}
		// Update session in memory
		if sess, ok := mgr.GetSession(workspace); ok {
			sess.ID = cascadeID
			sess.Status = "running"
			sess.LastActive = time.Now()
			appHub.Broadcast(models.WSMessage{
				Type:    "session_updated",
				Payload: sess,
			})
		}

		// Record message in history so mobile UI never loses it on refresh/switch
		mgr.RecordUserMessage(workspace, cascadeID, trimmed)

		// 4. Broadcast to connected WebSocket clients
		appHub.Broadcast(models.WSMessage{
			Type: "user_message_sent",
			Payload: map[string]interface{}{
				"workspace":  workspace,
				"cascade_id": cascadeID,
				"message":    trimmed,
				"timestamp":  time.Now().Format(time.RFC3339),
			},
		})

		if notice != "" {
			appHub.Broadcast(models.WSMessage{
				Type: "agent_notice",
				Payload: map[string]interface{}{
					"workspace": workspace,
					"message":   notice,
					"timestamp": time.Now().Format(time.RFC3339),
				},
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    err == nil,
			"workspace":  workspace,
			"cascade_id": cascadeID,
			"message":    trimmed,
			"notice":     notice,
		})
	})

	// List approvals
	mux.HandleFunc("GET /api/approvals", func(w http.ResponseWriter, r *http.Request) {
		ws := r.URL.Query().Get("workspace")
		apps := mgr.ListApprovals(ws)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apps)
	})

	// Decide approval (Approve / Deny)
	mux.HandleFunc("POST /api/approvals/{id}/decide", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var req struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		app, err := mgr.Decide(id, req.Decision, req.Reason)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		appHub.Broadcast(models.WSMessage{
			Type:    "approval_resolved",
			Payload: app,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})

	// List active workspaces / sessions
	mux.HandleFunc("GET /api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		sessions := mgr.ListSessions()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	})

	// Read transcript steps for session (supports both workspace name or session ID)
	mux.HandleFunc("GET /api/sessions/{id}/transcript", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		sessions := mgr.ListSessions()
		var transcriptPath string
		workspaceName := id
		for _, s := range sessions {
			if s.ID == id || s.Workspace == id {
				transcriptPath = s.TranscriptPath
				workspaceName = s.Workspace
				break
			}
		}

		if transcriptPath == "" {
			home, _ := os.UserHomeDir()
			transcriptPath = filepath.Join(home, ".gemini", "antigravity-ide", "brain", id, ".system_generated", "logs", "transcript.jsonl")
			if _, err := os.Stat(transcriptPath); err != nil {
				transcriptPath = filepath.Join(home, ".gemini", "antigravity-ide", "brain", id, ".system_generated", "logs", "transcript_full.jsonl")
			}
		}

		steps, err := transcript.ReadRecentSteps(transcriptPath, 300)
		if err != nil {
			steps = []transcript.TranscriptStep{}
		}

		// Merge recent mobile user messages that may not have been written to disk yet
		recentMsgs := mgr.GetRecentUserMessages(workspaceName, 15*time.Minute)
		lastIdx := 0
		if len(steps) > 0 {
			lastIdx = steps[len(steps)-1].StepIndex
		}
		for _, um := range recentMsgs {
			umNorm := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(um.Content, "\r\n", "\n"), "\r", "\n")))
			if umNorm == "" {
				continue
			}
			alreadyPresent := false
			for _, st := range steps {
				if st.Type == "USER_INPUT" {
					stClean := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(st.CleanText, "\r\n", "\n"), "\r", "\n")))
					stRaw := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(st.Content, "\r\n", "\n"), "\r", "\n")))
					if (stClean != "" && (strings.Contains(stClean, umNorm) || strings.Contains(umNorm, stClean))) ||
						(stRaw != "" && (strings.Contains(stRaw, umNorm) || strings.Contains(umNorm, stRaw))) {
						alreadyPresent = true
						break
					}
				}
			}
			if !alreadyPresent {
				lastIdx++
				newStep := transcript.TranscriptStep{
					StepIndex: lastIdx,
					Source:    "USER_EXPLICIT",
					Type:      "USER_INPUT",
					Status:    "DONE",
					Content:   um.Content,
					CleanText: um.Content,
					CreatedAt: um.Timestamp.UTC().Format(time.RFC3339Nano),
				}
				steps = append(steps, newStep)
			}
		}

		// ALWAYS sort all steps strictly chronologically by timestamp!
		sort.SliceStable(steps, func(i, j int) bool {
			ti, okI := parseStepTime(steps[i].CreatedAt)
			tj, okJ := parseStepTime(steps[j].CreatedAt)
			if okI && okJ && !ti.Equal(tj) {
				return ti.Before(tj)
			}
			if okI && !okJ {
				return true
			}
			if !okI && okJ {
				return false
			}
			return steps[i].StepIndex < steps[j].StepIndex
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(steps)
	})

	// Mode Settings
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.ServerSettings{Mode: mgr.GetMode()})
	})

	mux.HandleFunc("POST /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var req models.ServerSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Mode != "" {
			mgr.SetMode(req.Mode)
			appHub.Broadcast(models.WSMessage{
				Type:    "settings_updated",
				Payload: models.ServerSettings{Mode: mgr.GetMode()},
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.ServerSettings{Mode: mgr.GetMode()})
	})

	// ----------------------------------------------------
	// WEBSOCKET (Live updates to phone)
	// ----------------------------------------------------
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		appHub.ServeWS(w, r, func() []models.WSMessage {
			return []models.WSMessage{
				{
					Type:    "workspaces",
					Payload: mgr.ListSessions(),
				},
				{
					Type:    "settings_updated",
					Payload: models.ServerSettings{Mode: mgr.GetMode()},
				},
			}
		})
	})

	// Background status watcher: detects when any session becomes idle and broadcasts updates.
	// Transcript mtime alone lies when the agent hangs on a step (no writes, but still RUNNING),
	// so the language server's run status is authoritative before announcing "finished".
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		prevStatuses := make(map[string]string)
		stuckNotified := make(map[string]bool)
		for range ticker.C {
			sessions := mgr.ListSessions()
			for _, s := range sessions {
				prev, exists := prevStatuses[s.Workspace]
				if exists && prev == "running" && s.Status == "idle" && s.ID != "" {
					if inst, err := agyClient.GetInstanceForWorkspace(s.Workspace, s.WorkspacePath); err == nil {
						if st, err := agyClient.GetCascadeState(inst, s.ID); err == nil && st.Running() {
							if stuck := st.StuckFor(); stuck > 0 {
								if !stuckNotified[s.ID] {
									stuckNotified[s.ID] = true
									msg := fmt.Sprintf("Agent w [%s] wisi od %s na kroku %s. Wyślij wiadomość, a przerwę go i przekażę ją dalej.", s.Workspace, stuck.Round(time.Minute), st.LastStepType)
									log.Printf("[Monitor] %s", msg)
									appHub.Broadcast(models.WSMessage{
										Type: "agent_notice",
										Payload: map[string]interface{}{
											"workspace": s.Workspace,
											"message":   msg,
											"timestamp": time.Now().Format(time.RFC3339),
										},
									})
									sendNtfy("Antigravity zawieszony", msg, "warning,robot")
								}
							} else {
								// Still genuinely working (long generation / tool) — keep it running.
								s.Status = "running"
								s.LastActive = time.Now()
								prevStatuses[s.Workspace] = s.Status
								continue
							}
						} else {
							delete(stuckNotified, s.ID)
						}
					}
				}
				if exists && prev == "running" && s.Status == "idle" {
					log.Printf("[Monitor] Session [%s] finished work, transitioned to idle.", s.Workspace)
					appHub.Broadcast(models.WSMessage{
						Type:    "session_updated",
						Payload: s,
					})
					broadcastAgentFinished(appHub, s.Workspace, "monitor")
				}
				prevStatuses[s.Workspace] = s.Status
			}
		}
	}()

	// ----------------------------------------------------
	// PODCAST FEED & PLAYER REVERSE PROXY
	// ----------------------------------------------------
	podcastBackend, errPod := url.Parse("http://127.0.0.1:8777")
	if errPod != nil {
		log.Fatalf("Invalid podcast URL: %v", errPod)
	}
	podcastProxy := httputil.NewSingleHostReverseProxy(podcastBackend)
	origPodDirector := podcastProxy.Director
	podcastProxy.Director = func(req *http.Request) {
		origPodDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
		if req.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
	}
	podcastProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		log.Printf("[PodcastProxy] Error forwarding to %s: %v", r.URL.Path, proxyErr)
		http.Error(w, "Brain Podcast Engine is currently offline or starting up.", http.StatusBadGateway)
	}

	mux.Handle("GET /podcast", podcastProxy)
	mux.Handle("GET /podcast/{path...}", podcastProxy)
	mux.Handle("POST /podcast/{path...}", podcastProxy)

	// ----------------------------------------------------
	// STATIC SPA / PWA SERVING
	// ----------------------------------------------------
	absStatic, _ := filepath.Abs(*staticDir)
	fileServer := http.FileServer(http.Dir(absStatic))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/ws" || strings.HasPrefix(r.URL.Path, "/podcast") {
			http.NotFound(w, r)
			return
		}

		path := filepath.Join(absStatic, r.URL.Path)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		indexPath := filepath.Join(absStatic, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			http.ServeFile(w, r, indexPath)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Antigravity Remote</title><meta name="viewport" content="width=device-width, initial-scale=1"></head>
<body style="font-family: sans-serif; background: #0f172a; color: #f8fafc; padding: 2rem; text-align: center;">
  <h2>🚀 Antigravity Remote Backend Active</h2>
  <p>Port: %d</p>
  <p>Frontend static files not found in <code>%s</code>. Run <code>pnpm run build</code> in <code>frontend/</code>.</p>
</body></html>`, *port, absStatic)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("==================================================")
	log.Printf("📱 Antigravity Remote Server listening on %s (Dual HTTP & HTTPS)", addr)
	log.Printf("📁 Static directory: %s", absStatic)
	log.Printf("==================================================")

	var tlsConfig *tls.Config
	leCert, errLE := tls.LoadX509KeyPair(filepath.Join(*certDir, "server.crt"), filepath.Join(*certDir, "server.key"))
	selfCert, errSelf := tls.LoadX509KeyPair(filepath.Join(*certDir, "selfsigned.crt"), filepath.Join(*certDir, "selfsigned.key"))

	if errLE == nil || errSelf == nil {
		tlsConfig = &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				if errLE == nil {
					return &leCert, nil
				}
				if errSelf == nil {
					return &selfCert, nil
				}
				return nil, fmt.Errorf("no cert available")
			},
		}
		log.Printf("🔒 TLS certificates loaded successfully from %s", *certDir)
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Listen error: %v", err)
	}
	defer l.Close()

	dual := &dualListener{Listener: l, tlsConfig: tlsConfig}
	srv := &http.Server{
		Handler: corsMiddleware(mux),
	}

	if err := srv.Serve(dual); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

type dualListener struct {
	net.Listener
	tlsConfig *tls.Config
}

type peekConn struct {
	net.Conn
	peeked []byte
}

func (c *peekConn) Read(b []byte) (int, error) {
	if len(c.peeked) > 0 {
		n := copy(b, c.peeked)
		c.peeked = c.peeked[n:]
		return n, nil
	}
	return c.Conn.Read(b)
}

func (l *dualListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		c.SetReadDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 1)
		n, err := io.ReadFull(c, buf)
		c.SetReadDeadline(time.Time{})

		if err != nil || n == 0 {
			c.Close()
			continue
		}

		wrapped := &peekConn{Conn: c, peeked: buf}

		if buf[0] == 0x16 && l.tlsConfig != nil {
			return tls.Server(wrapped, l.tlsConfig), nil
		}

		return wrapped, nil
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseStepTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse("2006-01-02T15:04:05.999999999", s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

func isAgentReallyFinished(transcriptPath string) bool {
	if transcriptPath == "" {
		return false
	}
	steps, err := transcript.ReadRecentSteps(transcriptPath, 15)
	if err != nil || len(steps) == 0 {
		return false
	}
	last := steps[len(steps)-1]
	// If the last step is an active tool execution, command, or file change, agent is NOT finished!
	if last.Type == "RUN_COMMAND" || last.Type == "CODE_ACTION" || last.Type == "VIEW_FILE" ||
		last.Type == "GREP_SEARCH" || last.Type == "LIST_DIRECTORY" || last.Type == "MCP_TOOL" {
		return false
	}
	// If the last step has pending tool calls, agent is NOT finished!
	if len(last.ToolCalls) > 0 {
		return false
	}
	// If any recent step is still in RUNNING state (background task), agent is NOT finished!
	for _, s := range steps {
		if s.Status == "RUNNING" {
			return false
		}
	}
	// If the last step is a PLANNER_RESPONSE with actual content:
	if last.Type == "PLANNER_RESPONSE" && strings.TrimSpace(last.Content) != "" {
		contentLower := strings.ToLower(last.Content)
		// If the response is merely an intermediate waiting message, do not mark as finished!
		if strings.Contains(contentLower, "waiting for") ||
			strings.Contains(contentLower, "i will wait") ||
			strings.Contains(contentLower, "czekam na") {
			return false
		}
		return true
	}
	return false
}

func sendNtfy(title, msg, tags string) {
	topic := os.Getenv("AGY_NTFY_TOPIC")
	if topic == "" {
		home, _ := os.UserHomeDir()
		if data, err := os.ReadFile(filepath.Join(home, ".config/antigravity-remote/ntfy_topic")); err == nil {
			topic = strings.TrimSpace(string(data))
		}
	}
	if topic == "" {
		return
	}
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", "https://ntfy.sh/"+topic, strings.NewReader(msg))
		if err != nil {
			return
		}
		req.Header.Set("Title", title)
		req.Header.Set("Tags", tags)
		req.Header.Set("Priority", "default")
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
			log.Printf("[ntfy] Push notification sent to ntfy.sh/%s", topic)
		}
	}()
}
