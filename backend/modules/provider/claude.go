package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Claude struct {
	workDir string
}

func NewClaude(workDir string) *Claude {
	return &Claude{workDir: workDir}
}

func (c *Claude) Name() string      { return "claude" }
func (c *Claude) Category() string   { return "cli" }
func (c *Claude) IsAvailable() bool  { return CLIBinaryAvailable("claude") }

func (c *Claude) Models() []Model {
	return []Model{
		// CLI models (local binary)
		{ID: "cli-cc/claude-opus-4-6", Name: "Claude Opus 4.6", OwnedBy: "anthropic"},
		{ID: "cli-cc/claude-opus-4-6:low", Name: "Claude Opus 4.6 (Low)", OwnedBy: "anthropic"},
		{ID: "cli-cc/claude-opus-4-6:medium", Name: "Claude Opus 4.6 (Medium)", OwnedBy: "anthropic"},
		{ID: "cli-cc/claude-opus-4-6:max", Name: "Claude Opus 4.6 (Max)", OwnedBy: "anthropic"},
		{ID: "cli-cc/claude-sonnet-4-6", Name: "Claude Sonnet 4.6", OwnedBy: "anthropic"},
		{ID: "cli-cc/claude-sonnet-4-6:low", Name: "Claude Sonnet 4.6 (Low)", OwnedBy: "anthropic"},
		{ID: "cli-cc/claude-sonnet-4-6:medium", Name: "Claude Sonnet 4.6 (Medium)", OwnedBy: "anthropic"},
		{ID: "cli-cc/claude-sonnet-4-6:max", Name: "Claude Sonnet 4.6 (Max)", OwnedBy: "anthropic"},
		{ID: "cli-cc/claude-haiku-4-5", Name: "Claude Haiku 4.5", OwnedBy: "anthropic"},
		// OAuth models (subscription token)
		{ID: "cc/claude-opus-4-6", Name: "Claude Opus 4.6", OwnedBy: "anthropic"},
		{ID: "cc/claude-opus-4-6:low", Name: "Claude Opus 4.6 (Low)", OwnedBy: "anthropic"},
		{ID: "cc/claude-opus-4-6:medium", Name: "Claude Opus 4.6 (Medium)", OwnedBy: "anthropic"},
		{ID: "cc/claude-opus-4-6:max", Name: "Claude Opus 4.6 (Max)", OwnedBy: "anthropic"},
		{ID: "cc/claude-sonnet-4-6", Name: "Claude Sonnet 4.6", OwnedBy: "anthropic"},
		{ID: "cc/claude-sonnet-4-6:low", Name: "Claude Sonnet 4.6 (Low)", OwnedBy: "anthropic"},
		{ID: "cc/claude-sonnet-4-6:medium", Name: "Claude Sonnet 4.6 (Medium)", OwnedBy: "anthropic"},
		{ID: "cc/claude-sonnet-4-6:max", Name: "Claude Sonnet 4.6 (Max)", OwnedBy: "anthropic"},
		{ID: "cc/claude-haiku-4-5", Name: "Claude Haiku 4.5", OwnedBy: "anthropic"},
	}
}

// Execute implements Provider.Execute — spawns claude CLI with stream-json output
func (c *Claude) Execute(ctx context.Context, req *Request) (<-chan Event, error) {
	// Extract prompt from OpenAI RawBody
	systemPrompt, prompt := ExtractPrompt(req.RawBody)
	model := req.Model
	effort := req.Effort

	args := []string{
		"--print",
		"--model", model,
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
		"--verbose",
	}

	if effort != "" {
		args = append(args, "--effort", effort)
	}

	if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}

	log.Printf("[CLAUDE] Spawning: claude %s", strings.Join(args, " "))
	log.Printf("[CLAUDE] CWD: %s, Prompt: %d chars", c.workDir, len(prompt))

	cmd := exec.CommandContext(ctx, "claude", args...)

	if c.workDir != "" {
		cmd.Dir = c.workDir
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}

	// Send prompt as plain text via stdin
	stdin.Write([]byte(prompt))
	stdin.Close()

	// Capture stderr for error detection
	var stderrLines []string
	go func() {
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 0, 64*1024), 1*1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if line != "" {
				stderrLines = append(stderrLines, line)
				log.Printf("[CLAUDE stderr] %s", truncate(line, 300))
			}
		}
	}()

	events := make(chan Event, 128)
	go func() {
		defer close(events)
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 4*1024*1024)

		var lastText string
		var sentAnyText bool
		var sawToolUse bool
		seenToolIDs := make(map[string]bool) // track emitted tool_use blocks (content is cumulative)
		var editedFiles []string
		var createdFiles []string
		editedSet := make(map[string]bool)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var msg streamMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				log.Printf("[CLAUDE] Non-JSON line: %.200s", line)
				continue
			}

			switch msg.Type {
			case "system":
				log.Printf("[CLAUDE] System: subtype=%s", msg.Subtype)

			case "assistant":
				if msg.Message == nil {
					continue
				}

				for _, block := range msg.Message.Content {
					switch block.Type {
					case "text":
						if block.Text == "" {
							continue
						}

						newText := block.Text

						if strings.HasPrefix(newText, lastText) {
							// Same turn - send only the new delta
							delta := newText[len(lastText):]
							if delta != "" {
								lastText = newText
								sentAnyText = true
								events <- Event{Type: "text", Text: delta}
							}
						} else {
							// New turn after tool use - text reset
							// Add separator so Cursor formats it properly
							log.Printf("[CLAUDE] New turn (prev=%d chars, new=%d chars)", len(lastText), len(newText))
							prefix := ""
							if sentAnyText {
								prefix = "\n\n"
							}
							lastText = newText
							sentAnyText = true
							events <- Event{Type: "text", Text: prefix + newText}
						}

					case "tool_use":
						sawToolUse = true
						if seenToolIDs[block.ID] {
							continue // already emitted this tool_use
						}
						seenToolIDs[block.ID] = true
						log.Printf("[CLAUDE] Tool: %s (id=%s)", block.Name, block.ID)

						// Track edited files
						if block.Name == "Edit" || block.Name == "Write" {
							var fp struct {
								FilePath string `json:"file_path"`
							}
							if json.Unmarshal(block.Input, &fp) == nil && fp.FilePath != "" && !editedSet[fp.FilePath] {
								editedSet[fp.FilePath] = true
								if block.Name == "Write" {
									createdFiles = append(createdFiles, fp.FilePath)
								} else {
									editedFiles = append(editedFiles, fp.FilePath)
								}
							}
						}

						// Emit tool_use details as formatted text so Cursor shows what changed
						if toolText := formatToolUse(block.Name, block.Input); toolText != "" {
							prefix := ""
							if sentAnyText {
								prefix = "\n\n"
							}
							events <- Event{Type: "text", Text: prefix + toolText}
							sentAnyText = true
							// Reset lastText so next text block doesn't try prefix matching
							lastText = ""
						}

					case "tool_result":
						log.Printf("[CLAUDE] Tool result for %s", block.ToolUseID)
					}
				}

			case "result":
				if !sentAnyText {
					errMsg := "Claude returned empty response."
					if len(stderrLines) > 0 {
						errMsg += " stderr: " + stderrLines[len(stderrLines)-1]
					}
					if msg.CostUSD == 0 {
						errMsg += " (Possible usage limit reached)"
					}
					events <- Event{Type: "text", Text: errMsg}
				}

				// Summary of edited files
				if len(editedFiles) > 0 || len(createdFiles) > 0 {
					var summary strings.Builder
					total := len(editedFiles) + len(createdFiles)
					summary.WriteString(fmt.Sprintf("\n\n---\n**%d file(s) modified:**\n", total))
					for _, f := range createdFiles {
						summary.WriteString(fmt.Sprintf("- `%s` *(new)*\n", shortPath(f)))
					}
					for _, f := range editedFiles {
						summary.WriteString(fmt.Sprintf("- `%s` *(edited)*\n", shortPath(f)))
					}
					summary.WriteString("---")
					events <- Event{Type: "text", Text: summary.String()}
				}

				log.Printf("[CLAUDE] Done. Cost: $%.4f, Tools: %v, TextSent: %v, Files: %d",
					msg.CostUSD, sawToolUse, sentAnyText, len(editedFiles)+len(createdFiles))
				events <- Event{Type: "done", Cost: msg.CostUSD}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("[CLAUDE] Scanner error: %v", err)
			events <- Event{Type: "error", Text: err.Error()}
		}

		// If stream ended without result event and no text was sent
		if !sentAnyText {
			errMsg := "Claude process ended unexpectedly."
			if len(stderrLines) > 0 {
				errMsg += " " + stderrLines[len(stderrLines)-1]
			}
			events <- Event{Type: "text", Text: errMsg}
		}
	}()

	return events, nil
}

// Stream-json message types from Claude CLI
type streamMessage struct {
	Type    string        `json:"type"`
	Subtype string        `json:"subtype,omitempty"`
	Message *assistantMsg `json:"message,omitempty"`
	Result  string        `json:"result,omitempty"`
	CostUSD float64       `json:"cost_usd,omitempty"`
}

type assistantMsg struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

// formatToolUse converts a tool_use block into formatted text for the response.
// Format inspired by the Claude Code VSCode extension.
func formatToolUse(name string, input json.RawMessage) string {
	switch name {
	case "Edit":
		var inp struct {
			FilePath  string `json:"file_path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		}
		if json.Unmarshal(input, &inp) == nil && inp.FilePath != "" {
			fname := shortPath(inp.FilePath)
			added, removed := countDiffLines(inp.OldString, inp.NewString)
			stats := diffStats(added, removed)

			old := prefixDiffLines(inp.OldString, "- ")
			new := prefixDiffLines(inp.NewString, "+ ")

			// Code block with path that Cursor can recognize for "Apply"
			lang := detectLang(inp.FilePath)
			var b strings.Builder
			b.WriteString(fmt.Sprintf("**Edit** `%s` %s\n", fname, stats))
			b.WriteString(fmt.Sprintf("```diff\n%s%s```\n", old, new))
			// Block with filepath label for Cursor to detect
			b.WriteString(fmt.Sprintf("```%s:%s\n%s\n```", lang, inp.FilePath, inp.NewString))
			return b.String()
		}

	case "Write":
		var inp struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		if json.Unmarshal(input, &inp) == nil && inp.FilePath != "" {
			fname := shortPath(inp.FilePath)
			lines := strings.Count(inp.Content, "\n") + 1
			lang := detectLang(inp.FilePath)
			// Keep code block with filepath label for Cursor file detection
			return fmt.Sprintf("**Write** `%s` *(%d lines)*\n```%s:%s\n%s\n```", fname, lines, lang, inp.FilePath, inp.Content)
		}

	case "Read":
		var inp struct {
			FilePath string `json:"file_path"`
			Offset   int    `json:"offset"`
			Limit    int    `json:"limit"`
		}
		if json.Unmarshal(input, &inp) == nil && inp.FilePath != "" {
			fname := shortPath(inp.FilePath)
			if inp.Offset > 0 || inp.Limit > 0 {
				return fmt.Sprintf("*Read `%s` lines %d-%d*", fname, inp.Offset, inp.Offset+inp.Limit)
			}
			return fmt.Sprintf("*Read `%s`*", fname)
		}

	case "Bash":
		var inp struct {
			Command     string `json:"command"`
			Description string `json:"description"`
		}
		if json.Unmarshal(input, &inp) == nil && inp.Command != "" {
			cmd := inp.Command
			if len(cmd) > 300 {
				cmd = cmd[:300] + "..."
			}
			if inp.Description != "" {
				return fmt.Sprintf("**%s**\n> `%s`", inp.Description, cmd)
			}
			return fmt.Sprintf("> `%s`", cmd)
		}

	case "Grep":
		var inp struct {
			Pattern    string `json:"pattern"`
			Path       string `json:"path"`
			Glob       string `json:"glob"`
		}
		if json.Unmarshal(input, &inp) == nil && inp.Pattern != "" {
			path := shortPath(inp.Path)
			if path == "" {
				path = "."
			}
			extra := ""
			if inp.Glob != "" {
				extra = fmt.Sprintf(" `%s`", inp.Glob)
			}
			return fmt.Sprintf("*Grep `%s` in `%s`%s*", inp.Pattern, path, extra)
		}

	case "Glob":
		var inp struct {
			Pattern string `json:"pattern"`
			Path    string `json:"path"`
		}
		if json.Unmarshal(input, &inp) == nil && inp.Pattern != "" {
			path := shortPath(inp.Path)
			if path == "" {
				path = "."
			}
			return fmt.Sprintf("*Glob `%s` in `%s`*", inp.Pattern, path)
		}

	case "Task", "Agent":
		var inp struct {
			Description  string `json:"description"`
			Prompt       string `json:"prompt"`
			SubagentType string `json:"subagent_type"`
		}
		if json.Unmarshal(input, &inp) == nil {
			desc := inp.Description
			if desc == "" && len(inp.Prompt) > 80 {
				desc = inp.Prompt[:80] + "..."
			} else if desc == "" {
				desc = inp.Prompt
			}
			agent := inp.SubagentType
			if agent == "" {
				agent = "agent"
			}
			return fmt.Sprintf("**Task** (%s): %s", agent, desc)
		}

	case "TodoWrite":
		var inp struct {
			Todos []struct {
				Content    string `json:"content"`
				ActiveForm string `json:"activeForm"`
				Status     string `json:"status"`
			} `json:"todos"`
		}
		if json.Unmarshal(input, &inp) == nil && len(inp.Todos) > 0 {
			var b strings.Builder
			b.WriteString("**Plan:**\n")
			for _, t := range inp.Todos {
				switch t.Status {
				case "completed":
					b.WriteString("- [x] ~~" + t.Content + "~~\n")
				case "in_progress":
					b.WriteString("- [ ] **" + t.Content + "** *(in progress)*\n")
				default:
					b.WriteString("- [ ] " + t.Content + "\n")
				}
			}
			return b.String()
		}

	case "WebSearch":
		var inp struct {
			Query string `json:"query"`
		}
		if json.Unmarshal(input, &inp) == nil && inp.Query != "" {
			return fmt.Sprintf("*Web Search: %q*", inp.Query)
		}

	case "WebFetch":
		var inp struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(input, &inp) == nil && inp.URL != "" {
			return fmt.Sprintf("*Fetch: %s*", inp.URL)
		}

	default:
		var generic map[string]any
		if json.Unmarshal(input, &generic) == nil {
			if fp, ok := generic["file_path"].(string); ok {
				return fmt.Sprintf("*%s: `%s`*", name, shortPath(fp))
			}
			if cmd, ok := generic["command"].(string); ok {
				if len(cmd) > 100 {
					cmd = cmd[:100] + "..."
				}
				return fmt.Sprintf("*%s: `%s`*", name, cmd)
			}
			return fmt.Sprintf("*%s*", name)
		}
	}
	return ""
}

// shortPath extracts a short name from the path (last 2 segments)
func shortPath(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}

// countDiffLines counts added and removed lines
func countDiffLines(oldStr, newStr string) (added, removed int) {
	oldLines := strings.Split(oldStr, "\n")
	newLines := strings.Split(newStr, "\n")
	removed = len(oldLines)
	added = len(newLines)
	if oldStr == "" {
		removed = 0
	}
	if newStr == "" {
		added = 0
	}
	return
}

// diffStats generates a descriptive text of the changes
func diffStats(added, removed int) string {
	if removed == 0 && added > 0 {
		return fmt.Sprintf("*+%d lines*", added)
	}
	if added == 0 && removed > 0 {
		return fmt.Sprintf("*-%d lines*", removed)
	}
	if added > 0 && removed > 0 {
		return fmt.Sprintf("*+%d/-%d lines*", added, removed)
	}
	return ""
}

// detectLang returns a language for syntax highlighting
func detectLang(filePath string) string {
	if idx := strings.LastIndex(filePath, "."); idx >= 0 {
		ext := filePath[idx+1:]
		switch ext {
		case "ts", "tsx":
			return "typescript"
		case "js", "jsx", "mjs":
			return "javascript"
		case "py":
			return "python"
		case "go":
			return "go"
		case "vue":
			return "vue"
		case "yml", "yaml":
			return "yaml"
		case "sh", "bash", "zsh":
			return "bash"
		case "sql":
			return "sql"
		case "proto":
			return "protobuf"
		case "json":
			return "json"
		case "css", "scss":
			return ext
		default:
			return ext
		}
	}
	return ""
}

// prefixDiffLines adds a prefix to each line for diff formatting
func prefixDiffLines(s, prefix string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
