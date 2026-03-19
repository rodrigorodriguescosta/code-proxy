package provider

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	acp "github.com/coder/acp-go-sdk"
)

// acpClient implements the acp.Client interface.
// Handles bidirectional requests from the ACP agent (file I/O, permissions, terminals).
type acpClient struct {
	workDir  string
	manager  *acpManager
	mu       sync.Mutex
	terms    map[string]*terminal
	termSeq  int
}

type terminal struct {
	cmd    *exec.Cmd
	output strings.Builder
	mu     sync.Mutex
	done   chan struct{}
	exit   *int
}

var _ acp.Client = (*acpClient)(nil)

// ReadTextFile reads a file from disk
func (c *acpClient) ReadTextFile(_ context.Context, p acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	absPath := p.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(c.workDir, absPath)
	}

	b, err := os.ReadFile(absPath)
	if err != nil {
		return acp.ReadTextFileResponse{}, fmt.Errorf("read %s: %w", absPath, err)
	}

	content := string(b)

	// Support for line/limit
	if p.Line != nil || p.Limit != nil {
		lines := strings.Split(content, "\n")
		start := 0
		if p.Line != nil && *p.Line > 1 {
			start = *p.Line - 1
			if start > len(lines) {
				start = len(lines)
			}
		}
		end := len(lines)
		if p.Limit != nil && *p.Limit > 0 && start+*p.Limit < end {
			end = start + *p.Limit
		}
		content = strings.Join(lines[start:end], "\n")
	}

	return acp.ReadTextFileResponse{Content: content}, nil
}

// WriteTextFile writes a file to disk
func (c *acpClient) WriteTextFile(_ context.Context, p acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	absPath := p.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(c.workDir, absPath)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return acp.WriteTextFileResponse{}, fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(p.Content), 0o644); err != nil {
		return acp.WriteTextFileResponse{}, fmt.Errorf("write: %w", err)
	}

	log.Printf("[ACP] Wrote %d bytes to %s", len(p.Content), absPath)
	return acp.WriteTextFileResponse{}, nil
}

// RequestPermission auto-approves all permissions (equivalent to --dangerously-skip-permissions)
func (c *acpClient) RequestPermission(_ context.Context, p acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// Look for "allow_once" or "allow_always" option
	for _, opt := range p.Options {
		if opt.Kind == acp.PermissionOptionKindAllowOnce || opt.Kind == "allow_always" {
			log.Printf("[ACP] Auto-approved: %v (option: %s)", p.ToolCall.Title, opt.Name)
			return acp.RequestPermissionResponse{
				Outcome: acp.NewRequestPermissionOutcomeSelected(opt.OptionId),
			}, nil
		}
	}
	// Fallback: select first option
	if len(p.Options) > 0 {
		log.Printf("[ACP] Auto-approved (fallback): %v", p.ToolCall.Title)
		return acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected(p.Options[0].OptionId),
		}, nil
	}
	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeCancelled(),
	}, nil
}

// SessionUpdate receives streaming events from the agent and routes them to the correct session channel
func (c *acpClient) SessionUpdate(_ context.Context, n acp.SessionNotification) error {
	c.manager.mu.Lock()
	sess, ok := c.manager.sessions[n.SessionId]
	c.manager.mu.Unlock()

	if !ok {
		log.Printf("[ACP] Update for unknown session %s", n.SessionId)
		return nil
	}

	sess.mu.Lock()
	ch := sess.eventCh
	sess.mu.Unlock()
	if ch == nil {
		return nil
	}

	u := n.Update
	switch {
	case u.AgentMessageChunk != nil:
		if u.AgentMessageChunk.Content.Text != nil {
			ch <- Event{Type: "text", Text: u.AgentMessageChunk.Content.Text.Text}
		}

	case u.ToolCall != nil:
		text := formatACPToolCall(u.ToolCall)
		if text != "" {
			ch <- Event{Type: "text", Text: "\n\n" + text + "\n\n"}
		}

	case u.ToolCallUpdate != nil:
		text := formatACPToolCallUpdate(u.ToolCallUpdate)
		if text != "" {
			ch <- Event{Type: "text", Text: "\n" + text}
		}

	case u.AgentThoughtChunk != nil:
		// Include agent reasoning in the response (Cursor renders as text)
		if u.AgentThoughtChunk.Content.Text != nil {
			ch <- Event{Type: "text", Text: u.AgentThoughtChunk.Content.Text.Text}
		}

	case u.Plan != nil:
		if len(u.Plan.Entries) > 0 {
			var items []string
			for _, e := range u.Plan.Entries {
				icon := "[ ]"
				switch string(e.Status) {
				case "completed":
					icon = "[x]"
				case "in_progress":
					icon = "[~]"
				}
				items = append(items, fmt.Sprintf("  %s %s", icon, e.Content))
			}
			ch <- Event{Type: "text", Text: "\n\n**Plan:**\n" + strings.Join(items, "\n") + "\n\n"}
		}
	}

	return nil
}

// CreateTerminal creates a terminal and executes a command
func (c *acpClient) CreateTerminal(_ context.Context, p acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	c.mu.Lock()
	c.termSeq++
	id := fmt.Sprintf("term-%d", c.termSeq)
	c.mu.Unlock()

	cwd := c.workDir
	if p.Cwd != nil {
		cwd = *p.Cwd
	}

	args := p.Args
	cmd := exec.Command(p.Command, args...)
	cmd.Dir = cwd

	// Environment
	cmd.Env = os.Environ()
	for _, e := range p.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", e.Name, e.Value))
	}

	t := &terminal{cmd: cmd, done: make(chan struct{})}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return acp.CreateTerminalResponse{}, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr with stdout

	if err := cmd.Start(); err != nil {
		return acp.CreateTerminalResponse{}, fmt.Errorf("start: %w", err)
	}

	log.Printf("[ACP] Terminal %s: %s %v (cwd: %s)", id, p.Command, args, cwd)

	// Capture output
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1*1024*1024)
		for scanner.Scan() {
			t.mu.Lock()
			t.output.WriteString(scanner.Text())
			t.output.WriteString("\n")
			t.mu.Unlock()
		}
		exitCode := 0
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		t.mu.Lock()
		t.exit = &exitCode
		t.mu.Unlock()
		close(t.done)
	}()

	c.mu.Lock()
	if c.terms == nil {
		c.terms = make(map[string]*terminal)
	}
	c.terms[id] = t
	c.mu.Unlock()

	return acp.CreateTerminalResponse{TerminalId: id}, nil
}

// TerminalOutput returns the current terminal output
func (c *acpClient) TerminalOutput(_ context.Context, p acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	c.mu.Lock()
	t, ok := c.terms[p.TerminalId]
	c.mu.Unlock()
	if !ok {
		return acp.TerminalOutputResponse{}, fmt.Errorf("terminal not found: %s", p.TerminalId)
	}

	t.mu.Lock()
	output := t.output.String()
	var exitStatus *acp.TerminalExitStatus
	if t.exit != nil {
		exitStatus = &acp.TerminalExitStatus{ExitCode: t.exit}
	}
	t.mu.Unlock()

	return acp.TerminalOutputResponse{
		Output:     output,
		ExitStatus: exitStatus,
	}, nil
}

// KillTerminalCommand kills the terminal command
func (c *acpClient) KillTerminalCommand(_ context.Context, p acp.KillTerminalCommandRequest) (acp.KillTerminalCommandResponse, error) {
	c.mu.Lock()
	t, ok := c.terms[p.TerminalId]
	c.mu.Unlock()
	if !ok {
		return acp.KillTerminalCommandResponse{}, fmt.Errorf("terminal not found: %s", p.TerminalId)
	}

	if t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
	return acp.KillTerminalCommandResponse{}, nil
}

// WaitForTerminalExit waits for the terminal to finish
func (c *acpClient) WaitForTerminalExit(_ context.Context, p acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	c.mu.Lock()
	t, ok := c.terms[p.TerminalId]
	c.mu.Unlock()
	if !ok {
		return acp.WaitForTerminalExitResponse{}, fmt.Errorf("terminal not found: %s", p.TerminalId)
	}

	<-t.done

	t.mu.Lock()
	exitCode := t.exit
	t.mu.Unlock()

	return acp.WaitForTerminalExitResponse{ExitCode: exitCode}, nil
}

// ReleaseTerminal releases terminal resources
func (c *acpClient) ReleaseTerminal(_ context.Context, p acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	c.mu.Lock()
	t, ok := c.terms[p.TerminalId]
	if ok {
		delete(c.terms, p.TerminalId)
	}
	c.mu.Unlock()

	if ok && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}

	log.Printf("[ACP] Released terminal %s", p.TerminalId)
	return acp.ReleaseTerminalResponse{}, nil
}
