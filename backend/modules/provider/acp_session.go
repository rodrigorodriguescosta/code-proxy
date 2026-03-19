package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	acp "github.com/coder/acp-go-sdk"
)

// acpSession represents an active ACP session with event routing
type acpSession struct {
	sessionID acp.SessionId
	eventCh   chan Event
	mu        sync.Mutex
}

// acpManager manages the ACP subprocess and the session pool
type acpManager struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	conn        *acp.ClientSideConnection
	client      *acpClient
	sessions    map[acp.SessionId]*acpSession
	workDir     string
	acpCommand  string
	acpArgs     []string
	initialized bool
}

// newACPManager creates a new ACP manager
func newACPManager(workDir, acpCommand, acpArgs string) *acpManager {
	cmd := acpCommand
	if cmd == "" {
		cmd = "claude-code-acp"
	}

	var args []string
	if acpArgs != "" {
		args = strings.Fields(acpArgs)
	}

	return &acpManager{
		workDir:    workDir,
		acpCommand: cmd,
		acpArgs:    args,
		sessions:   make(map[acp.SessionId]*acpSession),
	}
}

// ensureProcess ensures the ACP subprocess is running, starting it if needed
func (m *acpManager) ensureProcess(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized && m.isAlive() {
		return nil
	}

	// Clear previous state if it exists
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
	}

	log.Printf("[ACP] Starting subprocess: %s %s", m.acpCommand, strings.Join(m.acpArgs, " "))

	cmd := exec.CommandContext(ctx, m.acpCommand, m.acpArgs...)
	cmd.Dir = m.workDir
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ACP subprocess: %w", err)
	}

	m.cmd = cmd
	m.client = &acpClient{
		workDir: m.workDir,
		manager: m,
	}

	m.conn = acp.NewClientSideConnection(m.client, stdin, stdout)

	// Initialize handshake
	_, err = m.conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
			Terminal: true,
		},
	})
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		m.conn = nil
		return fmt.Errorf("ACP initialize: %w", err)
	}

	m.initialized = true
	m.sessions = make(map[acp.SessionId]*acpSession)

	log.Printf("[ACP] Subprocess initialized (PID: %d)", cmd.Process.Pid)

	// Monitor process termination
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		m.conn = nil
		m.initialized = false
		// Close all active session channels
		for _, sess := range m.sessions {
			sess.mu.Lock()
			if sess.eventCh != nil {
				sess.eventCh <- Event{Type: "error", Text: "ACP subprocess exited"}
			}
			sess.mu.Unlock()
		}
		m.sessions = make(map[acp.SessionId]*acpSession)
		m.mu.Unlock()
		log.Printf("[ACP] Subprocess exited: %v", err)
	}()

	return nil
}

// isAlive checks whether the ACP process is running (must be called with mu locked)
func (m *acpManager) isAlive() bool {
	if m.cmd == nil || m.cmd.Process == nil {
		return false
	}
	// Checks whether the process still exists
	return m.cmd.ProcessState == nil
}

// getSession creates a new ACP session
func (m *acpManager) getSession(ctx context.Context) (*acpSession, error) {
	if err := m.ensureProcess(ctx); err != nil {
		return nil, err
	}

	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("ACP connection not available")
	}

	resp, err := conn.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        m.workDir,
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		return nil, fmt.Errorf("ACP new session: %w", err)
	}

	sess := &acpSession{
		sessionID: resp.SessionId,
		eventCh:   make(chan Event, 128),
	}

	m.mu.Lock()
	m.sessions[resp.SessionId] = sess
	m.mu.Unlock()

	log.Printf("[ACP] New session: %s", resp.SessionId)
	return sess, nil
}

// releaseSession removes a session from the map
func (m *acpManager) releaseSession(sessionID acp.SessionId) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	log.Printf("[ACP] Released session: %s", sessionID)
}

// shutdown terminates the ACP subprocess
func (m *acpManager) shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		log.Printf("[ACP] Shutting down subprocess")
		m.cmd.Process.Kill()
		m.cmd.Wait()
	}
	m.conn = nil
	m.initialized = false
}
