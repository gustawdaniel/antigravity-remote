package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

type ToolCallDetail struct {
	Name        string                 `json:"name"`
	Args        map[string]interface{} `json:"args,omitempty"`
	CommandLine string                 `json:"command_line,omitempty"`
	FilePath    string                 `json:"file_path,omitempty"`
	ToolSummary string                 `json:"toolSummary,omitempty"`
	ToolAction  string                 `json:"toolAction,omitempty"`
}

type TranscriptStep struct {
	StepIndex   int              `json:"step_index"`
	Source      string           `json:"source"`
	Type        string           `json:"type"`
	Status      string           `json:"status"`
	Content     string           `json:"content"`
	CleanText   string           `json:"clean_text,omitempty"`
	ExitCode    *int             `json:"exit_code,omitempty"`
	CreatedAt   string           `json:"created_at,omitempty"`
	CommandLine string           `json:"command_line,omitempty"`
	ToolName    string           `json:"tool_name,omitempty"`
	ToolCalls   []ToolCallDetail `json:"tool_calls,omitempty"`
	IsTruncated bool             `json:"is_truncated,omitempty"`
}

var userReqRegex = regexp.MustCompile(`(?s)<USER_REQUEST>(.*?)</USER_REQUEST>`)

// ReadRecentSteps reads up to `limit` recent steps from a transcript.jsonl file
func ReadRecentSteps(path string, limit int) ([]TranscriptStep, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var allSteps []TranscriptStep
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 20*1024*1024)

	var lastToolCommand string
	var lastToolName string

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		step := TranscriptStep{}
		if idx, ok := raw["step_index"].(float64); ok {
			step.StepIndex = int(idx)
		}
		if s, ok := raw["source"].(string); ok {
			step.Source = s
		}
		if t, ok := raw["type"].(string); ok {
			step.Type = t
		}
		if st, ok := raw["status"].(string); ok {
			step.Status = st
		}
		if ca, ok := raw["created_at"].(string); ok {
			step.CreatedAt = ca
		}
		if ec, ok := raw["exit_code"].(float64); ok {
			c := int(ec)
			step.ExitCode = &c
		}
		if c, ok := raw["content"].(string); ok {
			step.Content = c
		}

		// Parse tool calls if present (in PLANNER_RESPONSE)
		if tcRaw, ok := raw["tool_calls"].([]interface{}); ok {
			for _, item := range tcRaw {
				if tcMap, ok := item.(map[string]interface{}); ok {
					name, _ := tcMap["name"].(string)
					detail := ToolCallDetail{Name: name}
					if args, ok := tcMap["args"].(map[string]interface{}); ok {
						detail.Args = args
						if cmd, ok := args["CommandLine"].(string); ok {
							// Strip quotes if any
							detail.CommandLine = strings.Trim(cmd, "\"")
							lastToolCommand = detail.CommandLine
						}
						if fp, ok := args["TargetFile"].(string); ok {
							detail.FilePath = strings.Trim(fp, "\"")
						} else if fp, ok := args["AbsolutePath"].(string); ok {
							detail.FilePath = strings.Trim(fp, "\"")
						} else if fp, ok := args["DirectoryPath"].(string); ok {
							detail.FilePath = strings.Trim(fp, "\"")
						}
						if ta, ok := args["toolAction"].(string); ok {
							detail.ToolAction = strings.Trim(ta, "\"")
						}
						if ts, ok := args["toolSummary"].(string); ok {
							detail.ToolSummary = strings.Trim(ts, "\"")
						}
					}
					lastToolName = name
					step.ToolCalls = append(step.ToolCalls, detail)
				}
			}
		}

		// For RUN_COMMAND: associate command line and extract clean output
		if step.Type == "RUN_COMMAND" {
			step.CommandLine = lastToolCommand
			step.ToolName = "run_command"
			step.CleanText = extractCommandOutput(step.Content)
		} else if step.Type == "USER_INPUT" {
			step.CleanText = extractUserPrompt(step.Content)
		} else if step.Type == "PLANNER_RESPONSE" {
			step.CleanText = step.Content
			if len(step.ToolCalls) > 0 {
				step.ToolName = step.ToolCalls[0].Name
				step.CommandLine = step.ToolCalls[0].CommandLine
			}
		} else {
			step.CleanText = step.Content
			if step.CommandLine == "" && lastToolCommand != "" {
				step.CommandLine = lastToolCommand
			}
			if step.ToolName == "" && lastToolName != "" {
				step.ToolName = lastToolName
			}
		}

		// Don't truncate too aggressively - mobile can easily handle 50k
		if len(step.CleanText) > 50000 {
			step.CleanText = step.CleanText[:50000] + "\n... [truncated for mobile]"
			step.IsTruncated = true
		}

		allSteps = append(allSteps, step)
	}

	if len(allSteps) > limit {
		return allSteps[len(allSteps)-limit:], nil
	}
	return allSteps, nil
}

func extractUserPrompt(raw string) string {
	matches := userReqRegex.FindStringSubmatch(raw)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	// Fallback: strip tags
	lines := strings.Split(raw, "\n")
	var kept []string
	inMeta := false
	for _, l := range lines {
		if strings.HasPrefix(l, "<ADDITIONAL_METADATA>") || strings.HasPrefix(l, "<user_information>") {
			inMeta = true
			continue
		}
		if strings.HasPrefix(l, "</ADDITIONAL_METADATA>") || strings.HasPrefix(l, "</user_information>") {
			inMeta = false
			continue
		}
		if !inMeta && !strings.HasPrefix(l, "<USER_REQUEST>") && !strings.HasPrefix(l, "</USER_REQUEST>") {
			kept = append(kept, l)
		}
	}
	res := strings.TrimSpace(strings.Join(kept, "\n"))
	if res != "" {
		return res
	}
	return raw
}

func extractCommandOutput(raw string) string {
	// Look for standard "Output:\n" marker
	if idx := strings.Index(raw, "Output:\n"); idx != -1 {
		return strings.TrimSpace(raw[idx+len("Output:\n"):])
	}
	if idx := strings.Index(raw, "Output:\r\n"); idx != -1 {
		return strings.TrimSpace(raw[idx+len("Output:\r\n"):])
	}
	// If only stdout/stderr
	if strings.Contains(raw, "The command exited with code") {
		lines := strings.Split(raw, "\n")
		var outLines []string
		pastHeader := false
		for _, l := range lines {
			if strings.Contains(l, "Output:") {
				pastHeader = true
				continue
			}
			if pastHeader {
				outLines = append(outLines, l)
			}
		}
		if len(outLines) > 0 {
			return strings.TrimSpace(strings.Join(outLines, "\n"))
		}
	}
	return strings.TrimSpace(raw)
}
