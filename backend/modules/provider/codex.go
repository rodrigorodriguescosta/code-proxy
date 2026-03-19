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

// Codex is the provider for OpenAI Codex CLI
type Codex struct {
	workDir string
}

// NewCodex creates a Codex CLI provider
func NewCodex(workDir string) *Codex {
	return &Codex{workDir: workDir}
}

func (c *Codex) Name() string      { return "codex-cli" }
func (c *Codex) Category() string   { return "cli" }
func (c *Codex) IsAvailable() bool  { return CLIBinaryAvailable("codex") }

func (c *Codex) Models() []Model {
	return []Model{
		// CLI models (local binary)
		{ID: "cli-codex/5.3", Name: "GPT 5.3 Codex", OwnedBy: "openai"},
		{ID: "cli-codex/5.3-xhigh", Name: "GPT 5.3 Codex (xHigh)", OwnedBy: "openai"},
		{ID: "cli-codex/5.3-high", Name: "GPT 5.3 Codex (High)", OwnedBy: "openai"},
		{ID: "cli-codex/5.3-low", Name: "GPT 5.3 Codex (Low)", OwnedBy: "openai"},
		{ID: "cli-codex/5.3-none", Name: "GPT 5.3 Codex (None)", OwnedBy: "openai"},
		{ID: "cli-codex/5.3-spark", Name: "GPT 5.3 Codex Spark", OwnedBy: "openai"},

		{ID: "cli-codex/5.4", Name: "GPT 5.4 Codex", OwnedBy: "openai"},
		{ID: "cli-codex/5.4-xhigh", Name: "GPT 5.4 Codex (xHigh)", OwnedBy: "openai"},
		{ID: "cli-codex/5.4-high", Name: "GPT 5.4 Codex (High)", OwnedBy: "openai"},
		{ID: "cli-codex/5.4-low", Name: "GPT 5.4 Codex (Low)", OwnedBy: "openai"},
		{ID: "cli-codex/5.4-none", Name: "GPT 5.4 Codex (None)", OwnedBy: "openai"},
		{ID: "cli-codex/5.4-spark", Name: "GPT 5.4 Codex Spark", OwnedBy: "openai"},

		{ID: "cli-codex/5.2-codex", Name: "GPT 5.2 Codex", OwnedBy: "openai"},
		{ID: "cli-codex/5.2-base", Name: "GPT 5.2", OwnedBy: "openai"},

		{ID: "cli-codex/5.1", Name: "GPT 5.1 Codex", OwnedBy: "openai"},
		{ID: "cli-codex/5.1-max", Name: "GPT 5.1 Codex Max", OwnedBy: "openai"},
		{ID: "cli-codex/5.1-mini", Name: "GPT 5.1 Codex Mini", OwnedBy: "openai"},
		{ID: "cli-codex/5.1-mini-high", Name: "GPT 5.1 Codex Mini (High)", OwnedBy: "openai"},
		{ID: "cli-codex/5.1-base", Name: "GPT 5.1", OwnedBy: "openai"},

		{ID: "cli-codex/5-codex", Name: "GPT 5 Codex", OwnedBy: "openai"},
		{ID: "cli-codex/5-mini", Name: "GPT 5 Codex Mini", OwnedBy: "openai"},
		{ID: "cli-codex/o4-mini", Name: "o4 Mini", OwnedBy: "openai"},
		{ID: "cli-codex/o3", Name: "o3", OwnedBy: "openai"},

		// OAuth models (subscription token)
		{ID: "codex/5.3", Name: "GPT 5.3 Codex", OwnedBy: "openai"},
		{ID: "codex/5.3-xhigh", Name: "GPT 5.3 Codex (xHigh)", OwnedBy: "openai"},
		{ID: "codex/5.3-high", Name: "GPT 5.3 Codex (High)", OwnedBy: "openai"},
		{ID: "codex/5.3-low", Name: "GPT 5.3 Codex (Low)", OwnedBy: "openai"},
		{ID: "codex/5.3-spark", Name: "GPT 5.3 Codex Spark", OwnedBy: "openai"},
		{ID: "codex/5.4", Name: "GPT 5.4 Codex", OwnedBy: "openai"},
		{ID: "codex/5.4-xhigh", Name: "GPT 5.4 Codex (xHigh)", OwnedBy: "openai"},
		{ID: "codex/5.4-high", Name: "GPT 5.4 Codex (High)", OwnedBy: "openai"},
		{ID: "codex/5.4-low", Name: "GPT 5.4 Codex (Low)", OwnedBy: "openai"},
		{ID: "codex/5.4-spark", Name: "GPT 5.4 Codex Spark", OwnedBy: "openai"},
		{ID: "codex/5.2-codex", Name: "GPT 5.2 Codex", OwnedBy: "openai"},
		{ID: "codex/5.1", Name: "GPT 5.1 Codex", OwnedBy: "openai"},
		{ID: "codex/5.1-max", Name: "GPT 5.1 Codex Max", OwnedBy: "openai"},
		{ID: "codex/5.1-mini", Name: "GPT 5.1 Codex Mini", OwnedBy: "openai"},
		{ID: "codex/5-codex", Name: "GPT 5 Codex", OwnedBy: "openai"},
		{ID: "codex/5-mini", Name: "GPT 5 Codex Mini", OwnedBy: "openai"},
		{ID: "codex/o4-mini", Name: "o4 Mini", OwnedBy: "openai"},
		{ID: "codex/o3", Name: "o3", OwnedBy: "openai"},
	}
}

func (c *Codex) Execute(ctx context.Context, req *Request) (<-chan Event, error) {
	systemPrompt, userPrompt := ExtractPrompt(req.RawBody)
	prompt := strings.TrimSpace(userPrompt)
	if prompt == "" && strings.TrimSpace(systemPrompt) != "" {
		// Some clients put the main instructions in the "system" role.
		// Codex has no system/user split, so treat system prompt as the input.
		prompt = strings.TrimSpace(systemPrompt)
	} else if prompt != "" && strings.TrimSpace(systemPrompt) != "" {
		// Combine both to preserve context.
		prompt = systemPrompt + "\n\n" + prompt
	}

	model := req.Model
	if model == "" {
		model = "o4-mini"
	}

	cliModel := model
	if len(model) > 0 && model[0] >= '0' && model[0] <= '9' {
		cliModel = mapCodexCLIModel(model)
	}

	log.Printf("[CODEX] Prompt: %d chars", len(prompt))

	if strings.TrimSpace(prompt) == "" {
		// Avoid spawning codex with an empty prompt; return a meaningful error.
		// Debug: help identify the incoming OpenAI payload shape.
		var root map[string]any
		if err := json.Unmarshal(req.RawBody, &root); err == nil {
			log.Printf("[CODEX][DEBUG] ExtractPrompt empty: topLevelKeys=%v", keysOfMap(root))
			if msgsVal, ok := root["messages"]; ok {
				log.Printf("[CODEX][DEBUG] root.messages type=%T", msgsVal)
				if msgsAny, ok := msgsVal.([]any); ok {
					log.Printf("[CODEX][DEBUG] root.messages arrayLen=%d", len(msgsAny))
				}
			} else {
				log.Printf("[CODEX][DEBUG] root.messages missing")
			}
			for _, k := range []string{"input", "prompt", "text"} {
				if v, ok := root[k]; ok {
					log.Printf("[CODEX][DEBUG] root.%s type=%T", k, v)
				}
			}

			// If messages is a proper array, dump first few content shape fields.
			if msgsAny, ok := root["messages"].([]any); ok {
				for i, m := range msgsAny {
					if i >= 3 {
						break
					}
					mm, ok := m.(map[string]any)
					if !ok {
						continue
					}
					role, _ := mm["role"].(string)
					content := mm["content"]
					switch ct := content.(type) {
					case string:
						log.Printf("[CODEX][DEBUG] msg[%d] role=%s content=string len=%d", i, role, len(ct))
					case []any:
						log.Printf("[CODEX][DEBUG] msg[%d] role=%s content=array len=%d", i, role, len(ct))
					case map[string]any:
						log.Printf("[CODEX][DEBUG] msg[%d] role=%s content=object keys=%v", i, role, keysOfMap(ct))
					default:
						log.Printf("[CODEX][DEBUG] msg[%d] role=%s contentType=%T", i, role, content)
					}
				}
			}
		}

		events := make(chan Event, 2)
		go func() {
			defer close(events)
			events <- Event{Type: "text", Text: "(Codex prompt is empty - no instructions provided.)"}
			events <- Event{Type: "done"}
		}()
		return events, nil
	}

	isModelUnsupported := func(stderrStr string) bool {
		// codex emits: "The '<model>' model is not supported when using Codex with a ChatGPT account."
		return strings.Contains(stderrStr, "model is not supported") && strings.Contains(stderrStr, "ChatGPT account")
	}

	events := make(chan Event, 128)

	runOnce := func(includeModel bool) (fullTextLen int, stderrStr string, scanErr error, waitErr error) {
		args := []string{
			"exec",
			"--full-auto",
			"-",
		}
		if includeModel {
			args = []string{"exec", "--model", cliModel, "--full-auto", "-"}
		}

		log.Printf("[CODEX] Spawning: codex %s", strings.Join(args, " "))

		cmd := exec.CommandContext(ctx, "codex", args...)
		if c.workDir != "" {
			cmd.Dir = c.workDir
		}

		var stderrBuf strings.Builder
		cmd.Stderr = &stderrBuf

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return 0, stderrBuf.String(), nil, err
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return 0, stderrBuf.String(), nil, err
		}
		if err := cmd.Start(); err != nil {
			return 0, stderrBuf.String(), nil, err
		}

		_, _ = stdin.Write([]byte(prompt))
		_ = stdin.Close()

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

		var fullText strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var msg struct {
				Type    string `json:"type,omitempty"`
				Content string `json:"content,omitempty"`
				Text    string `json:"text,omitempty"`
			}

			if json.Unmarshal([]byte(line), &msg) == nil {
				if msg.Content != "" {
					// We only return fullTextLen and stderrStr here; the caller decides whether to rerun.
					events <- Event{Type: "text", Text: msg.Content}
					fullText.WriteString(msg.Content)
				} else if msg.Text != "" {
					events <- Event{Type: "text", Text: msg.Text}
					fullText.WriteString(msg.Text)
				} else {
					delta := line + "\n"
					events <- Event{Type: "text", Text: delta}
					fullText.WriteString(delta)
				}
			} else {
				delta := line + "\n"
				events <- Event{Type: "text", Text: delta}
				fullText.WriteString(delta)
			}
		}

		scanErr = scanner.Err()
		waitErr = cmd.Wait()
		stderrStr = strings.TrimSpace(stderrBuf.String())
		fullTextLen = fullText.Len()
		return
	}

	go func() {
		defer close(events)
		fullLen, stderrStr, scanErr, waitErr := runOnce(true)

		// If the chosen model isn't allowed for the current Codex account, retry without --model.
		if fullLen == 0 && isModelUnsupported(stderrStr) {
			// Retry without specifying model; codex will use its configured default.
			_, stderrStr2, scanErr2, waitErr2 := runOnce(false)
			stderrStr = stderrStr2
			scanErr = scanErr2
			waitErr = waitErr2
			fullLen = 1 // indicates second run was attempted; if it still produced nothing, we'll report error below
		}

		if fullLen == 0 {
			msg := "(Codex returned empty response)"
			if scanErr != nil {
				msg += fmt.Sprintf(" scan error: %v", scanErr)
			}
			if waitErr != nil {
				msg += fmt.Sprintf(" exit error: %v", waitErr)
			}
			if stderrStr != "" {
				parts := strings.Split(stderrStr, "\n")
				msg += " stderr: " + strings.TrimSpace(parts[len(parts)-1])
			}
			events <- Event{Type: "text", Text: msg}
		} else if scanErr != nil {
			events <- Event{Type: "text", Text: fmt.Sprintf("\n(Codex stream scan error: %v)\n", scanErr)}
		}

		events <- Event{Type: "done"}
	}()

	return events, nil
}

func mapCodexCLIModel(cleanModel string) string {
	// cleanModel examples (coming from registry.ResolveProvider on "codex/" prefix):
	// - "5.4-xhigh"       -> "gpt-5.4-codex-xhigh"
	// - "5.4"             -> "gpt-5.4-codex"
	// - "5.2-codex"       -> "gpt-5.2-codex"
	// - "5.2-base"        -> "gpt-5.2"
	// - "5.1-mini-high"  -> "gpt-5.1-codex-mini-high"

	if strings.HasSuffix(cleanModel, "-base") {
		base := strings.TrimSuffix(cleanModel, "-base")
		return "gpt-" + base
	}

	if strings.Contains(cleanModel, "-codex") {
		return "gpt-" + cleanModel
	}

	if !strings.Contains(cleanModel, "-") {
		return fmt.Sprintf("gpt-%s-codex", cleanModel)
	}

	parts := strings.SplitN(cleanModel, "-", 2)
	ver := parts[0]
	rest := parts[1]
	return fmt.Sprintf("gpt-%s-codex-%s", ver, rest)
}

func keysOfMap(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
