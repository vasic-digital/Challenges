package userflow

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// TauriCLIAdapter implements DesktopAdapter using WebDriver
// protocol to control a Tauri application.
type TauriCLIAdapter struct {
	binaryPath   string
	cmd          *exec.Cmd
	done         chan struct{}
	sessionID    string
	webDriverURL string
	httpClient   *http.Client
	mu           sync.Mutex
}

// Compile-time interface check.
var _ DesktopAdapter = (*TauriCLIAdapter)(nil)

// NewTauriCLIAdapter creates a TauriCLIAdapter that will
// launch the given binary and connect via WebDriver.
func NewTauriCLIAdapter(
	binaryPath string,
) *TauriCLIAdapter {
	return &TauriCLIAdapter{
		binaryPath: binaryPath,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LaunchApp starts the Tauri binary with TAURI_AUTOMATION=true,
// finds the WebDriver port, and creates a WebDriver session.
func (a *TauriCLIAdapter) LaunchApp(
	ctx context.Context, config DesktopAppConfig,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cmd != nil {
		return fmt.Errorf("app already running")
	}

	// Find a free port for WebDriver.
	port, err := findFreePort()
	if err != nil {
		return fmt.Errorf("find free port: %w", err)
	}
	a.webDriverURL = fmt.Sprintf(
		"http://127.0.0.1:%d", port,
	)

	binaryPath := a.binaryPath
	if config.BinaryPath != "" {
		binaryPath = config.BinaryPath
	}

	args := config.Args
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	if config.WorkDir != "" {
		cmd.Dir = config.WorkDir
	}

	// Set up environment.
	cmd.Env = os.Environ()
	cmd.Env = append(
		cmd.Env, "TAURI_AUTOMATION=true",
	)
	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf("TAURI_WEBDRIVER_PORT=%d", port),
	)
	for k, v := range config.Env {
		cmd.Env = append(
			cmd.Env, fmt.Sprintf("%s=%s", k, v),
		)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch binary: %w", err)
	}

	a.cmd = cmd
	a.done = make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(a.done)
	}()

	return nil
}

// IsAppRunning checks if the Tauri process is still running.
func (a *TauriCLIAdapter) IsAppRunning(
	_ context.Context,
) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cmd == nil || a.cmd.Process == nil {
		return false, nil
	}

	select {
	case <-a.done:
		return false, nil
	default:
		return true, nil
	}
}

// Navigate loads the given URL in the application's webview
// via WebDriver.
func (a *TauriCLIAdapter) Navigate(
	ctx context.Context, url string,
) error {
	body := fmt.Sprintf(`{"url":"%s"}`, url)
	_, err := a.wdPost(
		ctx,
		fmt.Sprintf(
			"/session/%s/url", a.sessionID,
		),
		body,
	)
	if err != nil {
		return fmt.Errorf("navigate: %w", err)
	}
	return nil
}

// Click clicks the first element matching the selector.
func (a *TauriCLIAdapter) Click(
	ctx context.Context, selector string,
) error {
	elemID, err := a.findElement(ctx, selector)
	if err != nil {
		return fmt.Errorf("click find: %w", err)
	}

	_, err = a.wdPost(
		ctx,
		fmt.Sprintf(
			"/session/%s/element/%s/click",
			a.sessionID, elemID,
		),
		"{}",
	)
	if err != nil {
		return fmt.Errorf("click: %w", err)
	}
	return nil
}

// Fill types a value into the input matching the selector.
func (a *TauriCLIAdapter) Fill(
	ctx context.Context, selector, value string,
) error {
	elemID, err := a.findElement(ctx, selector)
	if err != nil {
		return fmt.Errorf("fill find: %w", err)
	}

	// Clear existing value.
	_, _ = a.wdPost(
		ctx,
		fmt.Sprintf(
			"/session/%s/element/%s/clear",
			a.sessionID, elemID,
		),
		"{}",
	)

	body := fmt.Sprintf(`{"text":"%s"}`, value)
	_, err = a.wdPost(
		ctx,
		fmt.Sprintf(
			"/session/%s/element/%s/value",
			a.sessionID, elemID,
		),
		body,
	)
	if err != nil {
		return fmt.Errorf("fill: %w", err)
	}
	return nil
}

// IsVisible checks if the element matching the selector is
// displayed.
func (a *TauriCLIAdapter) IsVisible(
	ctx context.Context, selector string,
) (bool, error) {
	elemID, err := a.findElement(ctx, selector)
	if err != nil {
		return false, nil
	}

	data, err := a.wdGet(
		ctx,
		fmt.Sprintf(
			"/session/%s/element/%s/displayed",
			a.sessionID, elemID,
		),
	)
	if err != nil {
		return false, fmt.Errorf("is visible: %w", err)
	}

	var resp struct {
		Value bool `json:"value"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return false, fmt.Errorf(
			"parse displayed: %w", err,
		)
	}
	return resp.Value, nil
}

// WaitForSelector polls until the element appears.
func (a *TauriCLIAdapter) WaitForSelector(
	ctx context.Context,
	selector string,
	timeout time.Duration,
) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		_, err := a.findElement(ctx, selector)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"wait for selector: %w", ctx.Err(),
			)
		case <-deadline:
			return fmt.Errorf(
				"wait for selector %s: timed out", selector,
			)
		case <-ticker.C:
		}
	}
}

// Screenshot captures the window via the WebDriver screenshot
// endpoint.
func (a *TauriCLIAdapter) Screenshot(
	ctx context.Context,
) ([]byte, error) {
	data, err := a.wdGet(
		ctx,
		fmt.Sprintf(
			"/session/%s/screenshot", a.sessionID,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}

	var resp struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf(
			"parse screenshot: %w", err,
		)
	}

	return base64.StdEncoding.DecodeString(resp.Value)
}

// InvokeCommand executes a Tauri IPC command via JS
// evaluation through WebDriver.
func (a *TauriCLIAdapter) InvokeCommand(
	ctx context.Context,
	command string,
	args ...string,
) (string, error) {
	argsJSON := "{}"
	if len(args) > 0 {
		argsJSON = strings.Join(args, ",")
	}

	script := fmt.Sprintf(
		"return JSON.stringify("+
			"await window.__TAURI__.invoke('%s', %s)"+
			")",
		command, argsJSON,
	)

	return a.executeScript(ctx, script)
}

// WaitForWindow waits until the WebDriver session is
// available.
func (a *TauriCLIAdapter) WaitForWindow(
	ctx context.Context, timeout time.Duration,
) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := a.createSession(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"wait for window: %w", ctx.Err(),
			)
		case <-deadline:
			return fmt.Errorf(
				"wait for window: timed out after %s",
				timeout,
			)
		case <-ticker.C:
		}
	}
}

// Close closes the WebDriver session and stops the process.
func (a *TauriCLIAdapter) Close(
	ctx context.Context,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Delete WebDriver session.
	if a.sessionID != "" {
		_, _ = a.wdDelete(
			ctx,
			fmt.Sprintf(
				"/session/%s", a.sessionID,
			),
		)
		a.sessionID = ""
	}

	// Stop the process.
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill()
		<-a.done
		a.cmd = nil
	}

	return nil
}

// Available returns true if the binary exists.
func (a *TauriCLIAdapter) Available(
	_ context.Context,
) bool {
	_, err := os.Stat(a.binaryPath)
	return err == nil
}

// createSession creates a new WebDriver session.
func (a *TauriCLIAdapter) createSession(
	ctx context.Context,
) error {
	body := `{"capabilities":{}}`
	data, err := a.wdPost(ctx, "/session", body)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	var resp struct {
		Value struct {
			SessionID string `json:"sessionId"`
		} `json:"value"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf(
			"parse session response: %w", err,
		)
	}

	a.mu.Lock()
	a.sessionID = resp.Value.SessionID
	a.mu.Unlock()
	return nil
}

// findElement finds an element by CSS selector and returns
// its WebDriver element ID.
func (a *TauriCLIAdapter) findElement(
	ctx context.Context, selector string,
) (string, error) {
	body := fmt.Sprintf(
		`{"using":"css selector","value":"%s"}`,
		selector,
	)
	data, err := a.wdPost(
		ctx,
		fmt.Sprintf(
			"/session/%s/element", a.sessionID,
		),
		body,
	)
	if err != nil {
		return "", fmt.Errorf("find element: %w", err)
	}

	var resp struct {
		Value map[string]string `json:"value"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf(
			"parse element response: %w", err,
		)
	}

	// WebDriver returns element ID as value in a map.
	for _, id := range resp.Value {
		return id, nil
	}
	return "", fmt.Errorf(
		"element not found: %s", selector,
	)
}

// executeScript runs a JS script via WebDriver and returns
// the result.
func (a *TauriCLIAdapter) executeScript(
	ctx context.Context, script string,
) (string, error) {
	body := fmt.Sprintf(
		`{"script":"%s","args":[]}`,
		strings.ReplaceAll(script, `"`, `\"`),
	)
	data, err := a.wdPost(
		ctx,
		fmt.Sprintf(
			"/session/%s/execute/async",
			a.sessionID,
		),
		body,
	)
	if err != nil {
		return "", fmt.Errorf("execute script: %w", err)
	}

	var resp struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse script: %w", err)
	}
	return string(resp.Value), nil
}

// wdGet performs a GET request to the WebDriver server.
func (a *TauriCLIAdapter) wdGet(
	ctx context.Context, path string,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, a.webDriverURL+path, nil,
	)
	if err != nil {
		return nil, err
	}
	return a.wdDo(req)
}

// wdPost performs a POST request to the WebDriver server.
func (a *TauriCLIAdapter) wdPost(
	ctx context.Context, path, body string,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, a.webDriverURL+path,
		bytes.NewBufferString(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return a.wdDo(req)
}

// wdDelete performs a DELETE request to the WebDriver server.
func (a *TauriCLIAdapter) wdDelete(
	ctx context.Context, path string,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete,
		a.webDriverURL+path, nil,
	)
	if err != nil {
		return nil, err
	}
	return a.wdDo(req)
}

// wdDo executes an HTTP request against the WebDriver server.
func (a *TauriCLIAdapter) wdDo(
	req *http.Request,
) ([]byte, error) {
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"webdriver request: %w", err,
		)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"read webdriver response: %w", err,
		)
	}

	if resp.StatusCode >= 400 {
		return data, fmt.Errorf(
			"webdriver HTTP %d: %s",
			resp.StatusCode, string(data),
		)
	}
	return data, nil
}

// findFreePort finds a free TCP port on the local machine.
func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}
