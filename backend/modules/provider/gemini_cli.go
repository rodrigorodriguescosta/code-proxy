package provider

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// GeminiCLI is the provider for Google Gemini CLI
type GeminiCLI struct {
	workDir string
}

// NewGeminiCLI creates a Gemini CLI provider
func NewGeminiCLI(workDir string) *GeminiCLI {
	return &GeminiCLI{workDir: workDir}
}

func (c *GeminiCLI) Name() string      { return "gemini-cli" }
func (c *GeminiCLI) Category() string   { return "cli" }
func (c *GeminiCLI) IsAvailable() bool  { return CLIBinaryAvailable("gemini") }

func (c *GeminiCLI) Models() []Model {
	return []Model{
		{ID: "gc/gemini-2.5-pro", Name: "Gemini 2.5 Pro (CLI)", OwnedBy: "google"},
		{ID: "gc/gemini-2.5-flash", Name: "Gemini 2.5 Flash (CLI)", OwnedBy: "google"},
		{ID: "gc/gemini-2.0-flash", Name: "Gemini 2.0 Flash (CLI)", OwnedBy: "google"},
	}
}

func (c *GeminiCLI) Execute(ctx context.Context, req *Request) (<-chan Event, error) {
	_, prompt := ExtractPrompt(req.RawBody)

	model := req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	args := []string{
		"--model", model,
		"--sandbox",
		"--yes",
	}

	log.Printf("[GEMINI-CLI] Spawning: gemini %s", strings.Join(args, " "))
	log.Printf("[GEMINI-CLI] Prompt: %d chars", len(prompt))

	cmd := exec.CommandContext(ctx, "gemini", args...)
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

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start gemini: %w", err)
	}

	stdin.Write([]byte(prompt))
	stdin.Close()

	events := make(chan Event, 128)
	go func() {
		defer close(events)
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

		var fullText strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			delta := line + "\n"
			events <- Event{Type: "text", Text: delta}
			fullText.WriteString(delta)
		}

		if fullText.Len() == 0 {
			events <- Event{Type: "text", Text: "(Gemini CLI returned empty response)"}
		}

		events <- Event{Type: "done"}
	}()

	return events, nil
}
