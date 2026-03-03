package userflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PuppeteerOption configures a PuppeteerAdapter.
type PuppeteerOption func(*PuppeteerAdapter)

// WithHeadless sets whether the browser runs in headless mode.
func WithHeadless(headless bool) PuppeteerOption {
	return func(a *PuppeteerAdapter) {
		a.headless = headless
	}
}

// WithBrowserPath sets the path to the browser executable.
func WithBrowserPath(path string) PuppeteerOption {
	return func(a *PuppeteerAdapter) {
		a.browserPath = path
	}
}

// WithContainerName sets the container name for fallback
// execution via podman exec.
func WithContainerName(name string) PuppeteerOption {
	return func(a *PuppeteerAdapter) {
		a.containerName = name
	}
}

// PuppeteerAdapter implements BrowserAdapter by executing
// Puppeteer via Node.js scripts. Each browser action generates
// a Node.js script that uses the puppeteer npm module, writes
// it to a temporary file, and runs it via node. Falls back to
// container execution when local node is unavailable.
//
// Usage:
//
//	adapter := NewPuppeteerAdapter(
//	    WithHeadless(true),
//	    WithContainerName("puppeteer"),
//	)
//	err := adapter.Initialize(ctx, BrowserConfig{
//	    BrowserType: "chrome",
//	    Headless:    true,
//	    WindowSize:  [2]int{1920, 1080},
//	})
type PuppeteerAdapter struct {
	headless      bool
	browserPath   string
	containerName string
	wsEndpoint    string
	initialized   bool
	width         int
	height        int
}

// Compile-time interface check.
var _ BrowserAdapter = (*PuppeteerAdapter)(nil)

// NewPuppeteerAdapter creates a PuppeteerAdapter with the
// given functional options. Defaults to headless mode with
// "puppeteer" as the fallback container name.
func NewPuppeteerAdapter(
	options ...PuppeteerOption,
) *PuppeteerAdapter {
	a := &PuppeteerAdapter{
		headless:      true,
		containerName: "puppeteer",
		width:         1920,
		height:        1080,
	}
	for _, opt := range options {
		opt(a)
	}
	return a
}

// Initialize launches a headless browser via Puppeteer and
// stores the WebSocket endpoint for subsequent operations.
func (a *PuppeteerAdapter) Initialize(
	ctx context.Context, config BrowserConfig,
) error {
	a.headless = config.Headless
	if config.WindowSize[0] > 0 {
		a.width = config.WindowSize[0]
	}
	if config.WindowSize[1] > 0 {
		a.height = config.WindowSize[1]
	}

	launchArgs := a.launchArgs()
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.launch(%s);
  const ep = browser.wsEndpoint();
  console.log(JSON.stringify({ wsEndpoint: ep }));
  await browser.disconnect();
})();
`, launchArgs)

	out, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf("initialize browser: %w", err)
	}

	var result struct {
		WSEndpoint string `json:"wsEndpoint"`
	}
	if err := json.Unmarshal(
		[]byte(strings.TrimSpace(out)), &result,
	); err != nil {
		return fmt.Errorf(
			"parse ws endpoint: %w", err,
		)
	}
	a.wsEndpoint = result.WSEndpoint
	a.initialized = true
	return nil
}

// Navigate loads the given URL in the browser page.
func (a *PuppeteerAdapter) Navigate(
	ctx context.Context, url string,
) error {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0] || await browser.newPage();
  await page.setViewport({ width: %d, height: %d });
  await page.goto('%s');
  console.log('navigated');
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint),
		a.width, a.height,
		escapeJSSingle(url))

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}
	return nil
}

// Click performs a click on the element matching the CSS
// selector.
func (a *PuppeteerAdapter) Click(
	ctx context.Context, selector string,
) error {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  await page.click('%s');
  console.log('clicked');
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint),
		escapeJSSingle(selector))

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf("click %s: %w", selector, err)
	}
	return nil
}

// Fill types a value into the input matching the CSS selector.
// Clears the existing value first by triple-clicking to select
// all, then types the new value.
func (a *PuppeteerAdapter) Fill(
	ctx context.Context, selector, value string,
) error {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  await page.click('%s', { clickCount: 3 });
  await page.type('%s', '%s');
  console.log('filled');
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint),
		escapeJSSingle(selector),
		escapeJSSingle(selector),
		escapeJSSingle(value))

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf("fill %s: %w", selector, err)
	}
	return nil
}

// SelectOption selects an option in a dropdown matching the
// CSS selector by setting the value via page.select().
func (a *PuppeteerAdapter) SelectOption(
	ctx context.Context, selector, value string,
) error {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  await page.select('%s', '%s');
  console.log('selected');
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint),
		escapeJSSingle(selector),
		escapeJSSingle(value))

	_, err := a.execNode(ctx, script)
	if err != nil {
		return fmt.Errorf(
			"select option %s: %w", selector, err,
		)
	}
	return nil
}

// IsVisible returns whether the element matching the CSS
// selector is currently visible.
func (a *PuppeteerAdapter) IsVisible(
	ctx context.Context, selector string,
) (bool, error) {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  const el = await page.$('%s');
  let visible = false;
  if (el) {
    const box = await el.boundingBox();
    visible = box !== null;
  }
  console.log(JSON.stringify({ visible }));
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint),
		escapeJSSingle(selector))

	out, err := a.execNode(ctx, script)
	if err != nil {
		return false, nil
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

// WaitForSelector waits until an element matching the CSS
// selector appears, up to the given timeout.
func (a *PuppeteerAdapter) WaitForSelector(
	ctx context.Context,
	selector string,
	timeout time.Duration,
) error {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  await page.waitForSelector('%s', { timeout: %d });
  console.log('found');
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint),
		escapeJSSingle(selector),
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
// the CSS selector.
func (a *PuppeteerAdapter) GetText(
	ctx context.Context, selector string,
) (string, error) {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  const el = await page.$('%s');
  const text = el
    ? await page.evaluate(e => e.textContent, el)
    : '';
  console.log(JSON.stringify({ text }));
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint),
		escapeJSSingle(selector))

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
// the element matching the CSS selector.
func (a *PuppeteerAdapter) GetAttribute(
	ctx context.Context, selector, attr string,
) (string, error) {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  const el = await page.$('%s');
  const val = el
    ? await page.evaluate(
        (e, a) => e.getAttribute(a), el, '%s'
      )
    : '';
  console.log(JSON.stringify({ value: val || '' }));
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint),
		escapeJSSingle(selector),
		escapeJSSingle(attr))

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

// Screenshot captures the current browser viewport as a PNG
// image encoded in base64, then decodes it.
func (a *PuppeteerAdapter) Screenshot(
	ctx context.Context,
) ([]byte, error) {
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  const buf = await page.screenshot({ encoding: 'base64' });
  console.log(buf);
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint))

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

// EvaluateJS executes JavaScript in the browser context via
// page.evaluate and returns the result as a string. The script
// is base64-encoded to avoid quoting issues.
func (a *PuppeteerAdapter) EvaluateJS(
	ctx context.Context, jsScript string,
) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(
		[]byte(jsScript),
	)
	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  const pages = await browser.pages();
  const page = pages[0];
  const code = Buffer.from('%s', 'base64').toString();
  const result = await page.evaluate(code);
  console.log(JSON.stringify(result));
  await browser.disconnect();
})();
`, escapeJSSingle(a.wsEndpoint), encoded)

	out, err := a.execNode(ctx, script)
	if err != nil {
		return "", fmt.Errorf("evaluate js: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// NetworkIntercept is not supported by the Puppeteer CLI
// adapter because network interception requires a persistent
// connection. Returns nil to indicate no error but the handler
// will not fire in practice.
func (a *PuppeteerAdapter) NetworkIntercept(
	_ context.Context,
	_ string,
	_ func(req *InterceptedRequest),
) error {
	return nil
}

// Close shuts down the browser that was launched during
// Initialize.
func (a *PuppeteerAdapter) Close(
	ctx context.Context,
) error {
	if !a.initialized {
		return nil
	}

	script := fmt.Sprintf(`
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.connect({
    browserWSEndpoint: '%s'
  });
  await browser.close();
  console.log('closed');
})();
`, escapeJSSingle(a.wsEndpoint))

	_, err := a.execNode(ctx, script)
	a.initialized = false
	if err != nil {
		return fmt.Errorf("close browser: %w", err)
	}
	return nil
}

// Available returns true if the puppeteer npm module is
// installed and importable via node.
func (a *PuppeteerAdapter) Available(
	_ context.Context,
) bool {
	cmd := exec.Command(
		"node", "-e", "require('puppeteer')",
	)
	err := cmd.Run()
	return err == nil
}

// --- internal helpers ---

// launchArgs builds the JSON options object for
// puppeteer.launch() based on the adapter configuration.
func (a *PuppeteerAdapter) launchArgs() string {
	opts := map[string]interface{}{
		"headless": a.headless,
		"args": []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			fmt.Sprintf(
				"--window-size=%d,%d",
				a.width, a.height,
			),
		},
	}
	if a.browserPath != "" {
		opts["executablePath"] = a.browserPath
	}

	data, _ := json.Marshal(opts)
	return string(data)
}

// execNode runs a Node.js script locally first, falling back
// to container execution via podman exec if local execution
// fails.
func (a *PuppeteerAdapter) execNode(
	ctx context.Context, script string,
) (string, error) {
	// Write script to temp file to avoid shell escaping
	// issues with -e flag on long scripts.
	tmpFile, err := os.CreateTemp("", "pptr-*.js")
	if err != nil {
		return "", fmt.Errorf(
			"create temp script: %w", err,
		)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(script); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf(
			"write temp script: %w", err,
		)
	}
	tmpFile.Close()

	// Try local execution first.
	cmd := exec.CommandContext(ctx, "node", tmpPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), nil
	}
	fmt.Fprintf(
		os.Stderr,
		"Local node exec failed: %v\nOutput: %s\n",
		err, string(out),
	)

	// Fall back to container execution. Copy the script
	// into the container and execute there.
	containerPath := filepath.Join(
		"/tmp", filepath.Base(tmpPath),
	)
	cpCmd := exec.CommandContext(
		ctx, "podman", "cp",
		tmpPath, a.containerName+":"+containerPath,
	)
	if cpErr := cpCmd.Run(); cpErr != nil {
		// If copy fails, try inline execution.
		cmd = exec.CommandContext(
			ctx, "podman", "exec",
			a.containerName,
			"node", "-e", script,
		)
		out, err = cmd.CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf(
				"node exec: %w\noutput: %s",
				err, string(out),
			)
		}
		return string(out), nil
	}

	cmd = exec.CommandContext(
		ctx, "podman", "exec",
		a.containerName, "node", containerPath,
	)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"node exec: %w\noutput: %s",
			err, string(out),
		)
	}
	return string(out), nil
}
