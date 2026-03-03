package userflow

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SeleniumAdapter implements BrowserAdapter using the W3C WebDriver
// protocol. It communicates with a Selenium Grid or standalone
// server over HTTP. This adapter is generic and works with any
// W3C-compliant WebDriver implementation (Selenium, Selenoid,
// Moon, Aerokube).
//
// Usage:
//
//	adapter := NewSeleniumAdapter("http://localhost:4444")
//	err := adapter.Initialize(ctx, BrowserConfig{
//	    BrowserType: "chrome",
//	    Headless:    true,
//	    WindowSize:  [2]int{1920, 1080},
//	})
type SeleniumAdapter struct {
	serverURL  string
	sessionID  string
	httpClient *http.Client
}

// Compile-time interface check.
var _ BrowserAdapter = (*SeleniumAdapter)(nil)

// NewSeleniumAdapter creates a SeleniumAdapter that connects to a
// Selenium WebDriver server at the given URL (e.g.,
// "http://localhost:4444").
func NewSeleniumAdapter(serverURL string) *SeleniumAdapter {
	return &SeleniumAdapter{
		serverURL: strings.TrimRight(serverURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Initialize creates a new WebDriver session with the given
// browser configuration. Supports Chrome, Firefox, Edge, and
// Safari browser types.
func (a *SeleniumAdapter) Initialize(
	ctx context.Context, config BrowserConfig,
) error {
	caps := map[string]interface{}{
		"browserName": seleniumBrowserName(config.BrowserType),
	}

	// Add browser-specific options.
	opts := make(map[string]interface{})
	var args []string
	if config.Headless {
		args = append(args, "--headless")
	}
	if config.WindowSize[0] > 0 && config.WindowSize[1] > 0 {
		args = append(args, fmt.Sprintf(
			"--window-size=%d,%d",
			config.WindowSize[0], config.WindowSize[1],
		))
	}
	args = append(args, config.ExtraArgs...)

	optKey := seleniumOptionsKey(config.BrowserType)
	if optKey != "" && len(args) > 0 {
		opts["args"] = args
		caps[optKey] = opts
	}

	body := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"alwaysMatch": caps,
		},
	}

	resp, err := a.wdPost(ctx, "/session", body)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	sessionID, ok := resp["sessionId"].(string)
	if !ok {
		// W3C response nests sessionId inside "value".
		if val, ok := resp["value"].(map[string]interface{}); ok {
			sessionID, _ = val["sessionId"].(string)
		}
	}
	if sessionID == "" {
		return fmt.Errorf(
			"no sessionId in response: %v", resp,
		)
	}
	a.sessionID = sessionID
	return nil
}

// Navigate loads the given URL in the browser.
func (a *SeleniumAdapter) Navigate(
	ctx context.Context, url string,
) error {
	_, err := a.wdPost(
		ctx,
		a.sessionPath("/url"),
		map[string]interface{}{"url": url},
	)
	if err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}
	return nil
}

// Click performs a click on the element matching the CSS
// selector.
func (a *SeleniumAdapter) Click(
	ctx context.Context, selector string,
) error {
	elemID, err := a.findElement(ctx, selector)
	if err != nil {
		return fmt.Errorf("click find %s: %w", selector, err)
	}
	_, err = a.wdPost(
		ctx,
		a.sessionPath(
			"/element/"+elemID+"/click",
		),
		map[string]interface{}{},
	)
	if err != nil {
		return fmt.Errorf("click %s: %w", selector, err)
	}
	return nil
}

// Fill types a value into the input matching the CSS selector.
func (a *SeleniumAdapter) Fill(
	ctx context.Context, selector, value string,
) error {
	elemID, err := a.findElement(ctx, selector)
	if err != nil {
		return fmt.Errorf("fill find %s: %w", selector, err)
	}

	// Clear existing value first.
	_, _ = a.wdPost(
		ctx,
		a.sessionPath("/element/"+elemID+"/clear"),
		map[string]interface{}{},
	)

	_, err = a.wdPost(
		ctx,
		a.sessionPath("/element/"+elemID+"/value"),
		map[string]interface{}{"text": value},
	)
	if err != nil {
		return fmt.Errorf("fill %s: %w", selector, err)
	}
	return nil
}

// SelectOption selects an option in a dropdown matching the
// CSS selector by executing JavaScript.
func (a *SeleniumAdapter) SelectOption(
	ctx context.Context, selector, value string,
) error {
	script := fmt.Sprintf(
		`var el = document.querySelector('%s');
		 if (el) { el.value = '%s'; el.dispatchEvent(new Event('change')); }`,
		escapeJSSingle(selector), escapeJSSingle(value),
	)
	_, err := a.executeScript(ctx, script, nil)
	if err != nil {
		return fmt.Errorf(
			"select option %s: %w", selector, err,
		)
	}
	return nil
}

// IsVisible returns whether the element matching the CSS
// selector is currently visible.
func (a *SeleniumAdapter) IsVisible(
	ctx context.Context, selector string,
) (bool, error) {
	elemID, err := a.findElement(ctx, selector)
	if err != nil {
		return false, nil // Element not found = not visible.
	}
	resp, err := a.wdGet(
		ctx,
		a.sessionPath(
			"/element/"+elemID+"/displayed",
		),
	)
	if err != nil {
		return false, fmt.Errorf(
			"is visible %s: %w", selector, err,
		)
	}
	displayed, _ := resp["value"].(bool)
	return displayed, nil
}

// WaitForSelector waits until an element matching the CSS
// selector appears, up to the given timeout.
func (a *SeleniumAdapter) WaitForSelector(
	ctx context.Context,
	selector string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	interval := 200 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err := a.findElement(ctx, selector)
		if err == nil {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf(
		"timeout waiting for selector %s after %v",
		selector, timeout,
	)
}

// GetText returns the text content of the element matching the
// CSS selector.
func (a *SeleniumAdapter) GetText(
	ctx context.Context, selector string,
) (string, error) {
	elemID, err := a.findElement(ctx, selector)
	if err != nil {
		return "", fmt.Errorf(
			"get text find %s: %w", selector, err,
		)
	}
	resp, err := a.wdGet(
		ctx,
		a.sessionPath("/element/"+elemID+"/text"),
	)
	if err != nil {
		return "", fmt.Errorf(
			"get text %s: %w", selector, err,
		)
	}
	text, _ := resp["value"].(string)
	return text, nil
}

// GetAttribute returns the value of the named attribute on the
// element matching the CSS selector.
func (a *SeleniumAdapter) GetAttribute(
	ctx context.Context, selector, attr string,
) (string, error) {
	elemID, err := a.findElement(ctx, selector)
	if err != nil {
		return "", fmt.Errorf(
			"get attr find %s: %w", selector, err,
		)
	}
	resp, err := a.wdGet(
		ctx,
		a.sessionPath(fmt.Sprintf(
			"/element/%s/attribute/%s",
			elemID, attr,
		)),
	)
	if err != nil {
		return "", fmt.Errorf(
			"get attribute %s: %w", attr, err,
		)
	}
	val, _ := resp["value"].(string)
	return val, nil
}

// Screenshot captures the current browser viewport as a PNG
// image.
func (a *SeleniumAdapter) Screenshot(
	ctx context.Context,
) ([]byte, error) {
	resp, err := a.wdGet(
		ctx, a.sessionPath("/screenshot"),
	)
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}
	b64, _ := resp["value"].(string)
	if b64 == "" {
		return nil, fmt.Errorf("empty screenshot data")
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf(
			"decode screenshot: %w", err,
		)
	}
	return data, nil
}

// EvaluateJS executes JavaScript in the browser context and
// returns the result as a string.
func (a *SeleniumAdapter) EvaluateJS(
	ctx context.Context, script string,
) (string, error) {
	result, err := a.executeScript(ctx, script, nil)
	if err != nil {
		return "", fmt.Errorf("evaluate js: %w", err)
	}
	if result == nil {
		return "", nil
	}
	switch v := result.(type) {
	case string:
		return v, nil
	default:
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}

// NetworkIntercept is not natively supported by WebDriver.
// Returns nil to indicate no error but the handler will not
// fire. Use a proxy-based solution for network interception
// with Selenium (e.g., BrowserMob Proxy).
func (a *SeleniumAdapter) NetworkIntercept(
	_ context.Context,
	_ string,
	_ func(req *InterceptedRequest),
) error {
	return nil
}

// Close terminates the WebDriver session and releases resources.
func (a *SeleniumAdapter) Close(
	ctx context.Context,
) error {
	if a.sessionID == "" {
		return nil
	}
	_, err := a.wdDelete(ctx, a.sessionPath(""))
	a.sessionID = ""
	if err != nil {
		return fmt.Errorf("close session: %w", err)
	}
	return nil
}

// Available returns true if the Selenium server is reachable.
func (a *SeleniumAdapter) Available(
	_ context.Context,
) bool {
	resp, err := a.httpClient.Get(
		a.serverURL + "/status",
	)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// --- internal helpers ---

// sessionPath returns the WebDriver path prefixed with the
// current session ID.
func (a *SeleniumAdapter) sessionPath(
	suffix string,
) string {
	return "/session/" + a.sessionID + suffix
}

// findElement finds an element by CSS selector and returns its
// WebDriver element ID.
func (a *SeleniumAdapter) findElement(
	ctx context.Context, selector string,
) (string, error) {
	resp, err := a.wdPost(
		ctx,
		a.sessionPath("/element"),
		map[string]interface{}{
			"using": "css selector",
			"value": selector,
		},
	)
	if err != nil {
		return "", err
	}

	val, ok := resp["value"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf(
			"unexpected element response: %v", resp,
		)
	}
	// W3C element IDs use a well-known key.
	for _, key := range []string{
		"element-6066-11e4-a52e-4f735466cecf",
		"ELEMENT",
	} {
		if id, ok := val[key].(string); ok {
			return id, nil
		}
	}
	return "", fmt.Errorf(
		"no element ID in response: %v", val,
	)
}

// executeScript runs JavaScript via the WebDriver execute
// endpoint.
func (a *SeleniumAdapter) executeScript(
	ctx context.Context,
	script string,
	args []interface{},
) (interface{}, error) {
	if args == nil {
		args = []interface{}{}
	}
	resp, err := a.wdPost(
		ctx,
		a.sessionPath("/execute/sync"),
		map[string]interface{}{
			"script": script,
			"args":   args,
		},
	)
	if err != nil {
		return nil, err
	}
	return resp["value"], nil
}

// wdPost sends a POST request to the WebDriver server.
func (a *SeleniumAdapter) wdPost(
	ctx context.Context,
	path string,
	body map[string]interface{},
) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		a.serverURL+path,
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return a.doRequest(req)
}

// wdGet sends a GET request to the WebDriver server.
func (a *SeleniumAdapter) wdGet(
	ctx context.Context, path string,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		a.serverURL+path, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return a.doRequest(req)
}

// wdDelete sends a DELETE request to the WebDriver server.
func (a *SeleniumAdapter) wdDelete(
	ctx context.Context, path string,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete,
		a.serverURL+path, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return a.doRequest(req)
}

// doRequest executes an HTTP request and parses the JSON
// response.
func (a *SeleniumAdapter) doRequest(
	req *http.Request,
) (map[string]interface{}, error) {
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf(
			"parse response (status %d): %w\nbody: %s",
			resp.StatusCode, err, string(respBody),
		)
	}

	// Check for WebDriver errors.
	if resp.StatusCode >= 400 {
		errMsg := "unknown error"
		if val, ok := result["value"].(map[string]interface{}); ok {
			if msg, ok := val["message"].(string); ok {
				errMsg = msg
			}
		}
		return nil, fmt.Errorf(
			"webdriver error (status %d): %s",
			resp.StatusCode, errMsg,
		)
	}

	return result, nil
}

// seleniumBrowserName maps BrowserConfig.BrowserType to a
// WebDriver browserName capability.
func seleniumBrowserName(browserType string) string {
	switch strings.ToLower(browserType) {
	case "chrome", "chromium":
		return "chrome"
	case "firefox", "gecko":
		return "firefox"
	case "edge", "msedge":
		return "MicrosoftEdge"
	case "safari":
		return "safari"
	default:
		return "chrome"
	}
}

// seleniumOptionsKey returns the browser-specific options key
// for WebDriver capabilities.
func seleniumOptionsKey(browserType string) string {
	switch strings.ToLower(browserType) {
	case "chrome", "chromium":
		return "goog:chromeOptions"
	case "firefox", "gecko":
		return "moz:firefoxOptions"
	case "edge", "msedge":
		return "ms:edgeOptions"
	default:
		return "goog:chromeOptions"
	}
}

// escapeJSSingle escapes single quotes in strings for use in
// JavaScript template literals.
func escapeJSSingle(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}
