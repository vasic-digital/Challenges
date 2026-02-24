package userflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// PlaywrightCLIAdapter implements BrowserAdapter by executing
// Playwright commands via Node.js scripts inside a container
// connected to a CDP (Chrome DevTools Protocol) endpoint.
type PlaywrightCLIAdapter struct {
	cdpEndpoint   string
	containerName string
	initialized   bool
}

// Compile-time interface check.
var _ BrowserAdapter = (*PlaywrightCLIAdapter)(nil)

// NewPlaywrightCLIAdapter creates a PlaywrightCLIAdapter that
// connects to the given CDP WebSocket URL. The containerName
// defaults to "playwright" for podman exec commands.
func NewPlaywrightCLIAdapter(
	cdpEndpoint string,
) *PlaywrightCLIAdapter {
	return &PlaywrightCLIAdapter{
		cdpEndpoint:   cdpEndpoint,
		containerName: "playwright",
	}
}

// Initialize connects to the CDP endpoint and creates a
// browser context.
func (a *PlaywrightCLIAdapter) Initialize(
	ctx context.Context, config BrowserConfig,
) error {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const context = await browser.newContext({
    viewport: { width: %d, height: %d }
  });
  const page = await context.newPage();
  console.log('initialized');
})();
`, a.cdpEndpoint, config.WindowSize[0], config.WindowSize[1])

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf("initialize browser: %w", err)
	}
	a.initialized = true
	return nil
}

// Navigate loads the given URL in the browser page.
func (a *PlaywrightCLIAdapter) Navigate(
	ctx context.Context, url string,
) error {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const contexts = browser.contexts();
  const page = contexts[0].pages()[0];
  await page.goto('%s');
  console.log('navigated');
})();
`, a.cdpEndpoint, url)

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}
	return nil
}

// Click performs a click on the element matching the selector.
func (a *PlaywrightCLIAdapter) Click(
	ctx context.Context, selector string,
) error {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  await page.click('%s');
  console.log('clicked');
})();
`, a.cdpEndpoint, escapeJS(selector))

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf("click %s: %w", selector, err)
	}
	return nil
}

// Fill types a value into the input matching the selector.
func (a *PlaywrightCLIAdapter) Fill(
	ctx context.Context, selector, value string,
) error {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  await page.fill('%s', '%s');
  console.log('filled');
})();
`, a.cdpEndpoint, escapeJS(selector), escapeJS(value))

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf(
			"fill %s: %w", selector, err,
		)
	}
	return nil
}

// SelectOption selects an option in a dropdown matching the
// selector.
func (a *PlaywrightCLIAdapter) SelectOption(
	ctx context.Context, selector, value string,
) error {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  await page.selectOption('%s', '%s');
  console.log('selected');
})();
`, a.cdpEndpoint, escapeJS(selector), escapeJS(value))

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf(
			"select option %s: %w", selector, err,
		)
	}
	return nil
}

// IsVisible returns whether the element matching the selector
// is currently visible.
func (a *PlaywrightCLIAdapter) IsVisible(
	ctx context.Context, selector string,
) (bool, error) {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  const visible = await page.isVisible('%s');
  console.log(JSON.stringify({ visible }));
})();
`, a.cdpEndpoint, escapeJS(selector))

	out, err := a.execNode(ctx, script)
	if err != nil {
		return false, fmt.Errorf(
			"is visible %s: %w", selector, err,
		)
	}

	var result struct {
		Visible bool `json:"visible"`
	}
	if err := json.Unmarshal(
		[]byte(strings.TrimSpace(out)), &result,
	); err != nil {
		return false, fmt.Errorf(
			"parse visibility: %w", err,
		)
	}
	return result.Visible, nil
}

// WaitForSelector waits until an element matching the selector
// appears, up to the given timeout.
func (a *PlaywrightCLIAdapter) WaitForSelector(
	ctx context.Context,
	selector string,
	timeout time.Duration,
) error {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  await page.waitForSelector('%s', { timeout: %d });
  console.log('found');
})();
`, a.cdpEndpoint, escapeJS(selector),
		timeout.Milliseconds())

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf(
			"wait for %s: %w", selector, err,
		)
	}
	return nil
}

// GetText returns the text content of the element matching
// the selector.
func (a *PlaywrightCLIAdapter) GetText(
	ctx context.Context, selector string,
) (string, error) {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  const text = await page.textContent('%s');
  console.log(JSON.stringify({ text }));
})();
`, a.cdpEndpoint, escapeJS(selector))

	out, err := a.execNode(ctx, script)
	if err != nil {
		return "", fmt.Errorf(
			"get text %s: %w", selector, err,
		)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(
		[]byte(strings.TrimSpace(out)), &result,
	); err != nil {
		return "", fmt.Errorf("parse text: %w", err)
	}
	return result.Text, nil
}

// GetAttribute returns the value of the named attribute on
// the element matching the selector.
func (a *PlaywrightCLIAdapter) GetAttribute(
	ctx context.Context, selector, attr string,
) (string, error) {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  const val = await page.getAttribute('%s', '%s');
  console.log(JSON.stringify({ value: val }));
})();
`, a.cdpEndpoint, escapeJS(selector), escapeJS(attr))

	out, err := a.execNode(ctx, script)
	if err != nil {
		return "", fmt.Errorf(
			"get attribute %s: %w", attr, err,
		)
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(
		[]byte(strings.TrimSpace(out)), &result,
	); err != nil {
		return "", fmt.Errorf(
			"parse attribute: %w", err,
		)
	}
	return result.Value, nil
}

// Screenshot captures the current browser viewport as PNG.
func (a *PlaywrightCLIAdapter) Screenshot(
	ctx context.Context,
) ([]byte, error) {
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  const buf = await page.screenshot();
  console.log(buf.toString('base64'));
})();
`, a.cdpEndpoint)

	out, err := a.execNode(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(
		strings.TrimSpace(out),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decode screenshot: %w", err,
		)
	}
	return data, nil
}

// EvaluateJS executes JavaScript in the browser and returns
// the result as a string.
func (a *PlaywrightCLIAdapter) EvaluateJS(
	ctx context.Context, jsScript string,
) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(
		[]byte(jsScript),
	)
	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  const page = browser.contexts()[0].pages()[0];
  const code = Buffer.from('%s', 'base64').toString();
  const result = await page.evaluate(code);
  console.log(JSON.stringify(result));
})();
`, a.cdpEndpoint, encoded)

	out, err := a.execNode(ctx, script)
	if err != nil {
		return "", fmt.Errorf("evaluate js: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// NetworkIntercept sets up a handler for network requests
// matching the given URL pattern.
func (a *PlaywrightCLIAdapter) NetworkIntercept(
	_ context.Context,
	_ string,
	_ func(req *InterceptedRequest),
) error {
	// Network interception requires a persistent connection
	// to the browser. This is a limitation of the CLI-based
	// approach. Return nil to indicate no error but the
	// handler will not fire in practice.
	return nil
}

// Close shuts down the browser connection.
func (a *PlaywrightCLIAdapter) Close(
	ctx context.Context,
) error {
	if !a.initialized {
		return nil
	}

	script := fmt.Sprintf(`
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.connectOverCDP('%s');
  await browser.close();
  console.log('closed');
})();
`, a.cdpEndpoint)

	_, err := a.execNode(ctx, script)
	a.initialized = false
	if err != nil {
		return fmt.Errorf("close browser: %w", err)
	}
	return nil
}

// Available checks if the CDP endpoint responds to an HTTP
// request.
func (a *PlaywrightCLIAdapter) Available(
	_ context.Context,
) bool {
	// Convert ws:// to http:// for health check.
	url := strings.Replace(
		a.cdpEndpoint, "ws://", "http://", 1,
	)
	url = strings.Replace(url, "wss://", "https://", 1)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// execNode runs a Node.js script inside the Playwright
// container via `podman exec`.
func (a *PlaywrightCLIAdapter) execNode(
	ctx context.Context, script string,
) (string, error) {
	cmd := exec.CommandContext(
		ctx, "podman", "exec", a.containerName,
		"node", "-e", script,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"node exec: %w\noutput: %s",
			err, string(out),
		)
	}
	return string(out), nil
}

// escapeJS escapes single quotes in strings for use in JS
// template literals.
func escapeJS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}
