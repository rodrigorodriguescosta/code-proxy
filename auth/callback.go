package auth

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

// CallbackResult is the captured result from an OAuth callback
type CallbackResult struct {
	Code  string
	State string
	Error string
}

// CallbackServer is a temporary local HTTP server for capturing OAuth callbacks
type CallbackServer struct {
	port     int
	resultCh chan CallbackResult
	server   *http.Server
}

// NewCallbackServer creates a callback server on the specified port
func NewCallbackServer(port int) *CallbackServer {
	return &CallbackServer{
		port:     port,
		resultCh: make(chan CallbackResult, 1),
	}
}

// Start starts the callback server
func (s *CallbackServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)
	mux.HandleFunc("/auth/callback", s.handleCallback)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("callback server listen: %w", err)
	}

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("[AUTH] Callback server error: %v", err)
		}
	}()

	log.Printf("[AUTH] Callback server listening on port %d", s.port)
	return nil
}

// handleCallback processes the OAuth provider redirect
func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errParam := r.URL.Query().Get("error")

	if errParam != "" {
		s.resultCh <- CallbackResult{Error: errParam}
		fmt.Fprintf(w, "<html><body><h2>Authentication Error</h2><p>%s</p><p>You can close this window.</p></body></html>", errParam)
		return
	}

	if code == "" {
		s.resultCh <- CallbackResult{Error: "no code in callback"}
		fmt.Fprintf(w, "<html><body><h2>Error</h2><p>No authorization code received.</p></body></html>")
		return
	}

	s.resultCh <- CallbackResult{Code: code, State: state}
	fmt.Fprintf(w, "<html><body><h2>Authentication Successful!</h2><p>You can close this window and return to the application.</p></body></html>")
}

// WaitForResult waits for the callback or a timeout
func (s *CallbackServer) WaitForResult(timeout time.Duration) (*CallbackResult, error) {
	select {
	case result := <-s.resultCh:
		if result.Error != "" {
			return nil, fmt.Errorf("OAuth error: %s", result.Error)
		}
		return &result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("OAuth callback timeout after %s", timeout)
	}
}

// SubmitCallback allows manual submission of the callback URL (remote/paste mode)
func (s *CallbackServer) SubmitCallback(callbackURL string) error {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return fmt.Errorf("invalid callback URL: %w", err)
	}

	code := u.Query().Get("code")
	state := u.Query().Get("state")
	errParam := u.Query().Get("error")

	if errParam != "" {
		s.resultCh <- CallbackResult{Error: errParam}
		return fmt.Errorf("OAuth error: %s", errParam)
	}

	if code == "" {
		// Try to extract code from the fragment (some providers use #code=...)
		if fragment := u.Fragment; fragment != "" {
			vals, _ := url.ParseQuery(fragment)
			code = vals.Get("code")
			if state == "" {
				state = vals.Get("state")
			}
		}
	}

	if code == "" {
		return fmt.Errorf("no authorization code found in URL")
	}

	s.resultCh <- CallbackResult{Code: code, State: state}
	return nil
}

// Stop shuts down the callback server
func (s *CallbackServer) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}
}
