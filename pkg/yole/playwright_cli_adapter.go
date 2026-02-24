package yole

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// playwrightCommandFunc is injectable for testing.
var playwrightCommandFunc = exec.CommandContext

// PlaywrightCLIAdapter implements PlaywrightAdapter by
// executing Playwright scripts via Node.js subprocess.
type PlaywrightCLIAdapter struct {
	browserType string
	url         string
	scriptDir   string
	timeout     time.Duration
}

// NewPlaywrightCLIAdapter creates an adapter that runs
// Playwright commands via npx/node subprocess.
func NewPlaywrightCLIAdapter() *PlaywrightCLIAdapter {
	return &PlaywrightCLIAdapter{
		browserType: "chromium",
		timeout:     30 * time.Second,
	}
}

// Available returns true if npx and playwright are installed.
func (p *PlaywrightCLIAdapter) Available(
	ctx context.Context,
) bool {
	_, err := exec.LookPath("npx")
	if err != nil {
		return false
	}
	cmd := playwrightCommandFunc(
		ctx, "npx", "playwright", "--version",
	)
	return cmd.Run() == nil
}

// Initialize sets up the browser type for subsequent
// operations.
func (p *PlaywrightCLIAdapter) Initialize(
	ctx context.Context, browserType string,
) error {
	if browserType != "" {
		p.browserType = browserType
	}
	tmpDir, err := os.MkdirTemp("", "playwright-yole-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	p.scriptDir = tmpDir
	return nil
}

// Navigate opens the specified URL in a headless browser.
func (p *PlaywrightCLIAdapter) Navigate(
	ctx context.Context, url string,
) error {
	p.url = url
	script := fmt.Sprintf(`
const { %s } = require('playwright');
(async () => {
  const browser = await %s.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('%s', { waitUntil: 'networkidle' });
  console.log('NAVIGATE_OK');
  await browser.close();
})();
`, p.browserType, p.browserType,
		strings.ReplaceAll(url, "'", "\\'"),
	)
	_, err := p.runScript(ctx, script)
	return err
}

// Click clicks an element matching the CSS selector.
func (p *PlaywrightCLIAdapter) Click(
	ctx context.Context, selector string,
) error {
	if p.url == "" {
		return fmt.Errorf("no page loaded; call Navigate first")
	}
	script := fmt.Sprintf(`
const { %s } = require('playwright');
(async () => {
  const browser = await %s.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('%s', { waitUntil: 'networkidle' });
  await page.click('%s');
  console.log('CLICK_OK');
  await browser.close();
})();
`, p.browserType, p.browserType,
		strings.ReplaceAll(p.url, "'", "\\'"),
		strings.ReplaceAll(selector, "'", "\\'"),
	)
	_, err := p.runScript(ctx, script)
	return err
}

// ClickByText clicks an element containing the given text.
func (p *PlaywrightCLIAdapter) ClickByText(
	ctx context.Context, text string,
) error {
	if p.url == "" {
		return fmt.Errorf("no page loaded; call Navigate first")
	}
	script := fmt.Sprintf(`
const { %s } = require('playwright');
(async () => {
  const browser = await %s.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('%s', { waitUntil: 'networkidle' });
  await page.getByText('%s').click();
  console.log('CLICK_TEXT_OK');
  await browser.close();
})();
`, p.browserType, p.browserType,
		strings.ReplaceAll(p.url, "'", "\\'"),
		strings.ReplaceAll(text, "'", "\\'"),
	)
	_, err := p.runScript(ctx, script)
	return err
}

// IsVisible checks if an element matching the selector is
// visible on the page.
func (p *PlaywrightCLIAdapter) IsVisible(
	ctx context.Context, selector string,
) (bool, error) {
	if p.url == "" {
		return false, fmt.Errorf(
			"no page loaded; call Navigate first",
		)
	}
	script := fmt.Sprintf(`
const { %s } = require('playwright');
(async () => {
  const browser = await %s.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('%s', { waitUntil: 'networkidle' });
  const visible = await page.isVisible('%s');
  console.log(visible ? 'VISIBLE_TRUE' : 'VISIBLE_FALSE');
  await browser.close();
})();
`, p.browserType, p.browserType,
		strings.ReplaceAll(p.url, "'", "\\'"),
		strings.ReplaceAll(selector, "'", "\\'"),
	)
	out, err := p.runScript(ctx, script)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "VISIBLE_TRUE"), nil
}

// Screenshot captures the current page as PNG bytes.
func (p *PlaywrightCLIAdapter) Screenshot(
	ctx context.Context,
) ([]byte, error) {
	if p.url == "" {
		return nil, fmt.Errorf(
			"no page loaded; call Navigate first",
		)
	}
	script := fmt.Sprintf(`
const { %s } = require('playwright');
(async () => {
  const browser = await %s.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('%s', { waitUntil: 'networkidle' });
  const buf = await page.screenshot({ fullPage: true });
  console.log(buf.toString('base64'));
  await browser.close();
})();
`, p.browserType, p.browserType,
		strings.ReplaceAll(p.url, "'", "\\'"),
	)
	out, err := p.runScript(ctx, script)
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(
		strings.TrimSpace(out),
	)
	if err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}
	return data, nil
}

// Close cleans up temporary files.
func (p *PlaywrightCLIAdapter) Close(
	_ context.Context,
) error {
	if p.scriptDir != "" {
		return os.RemoveAll(p.scriptDir)
	}
	return nil
}

// runScript writes a Node.js script to a temp file and
// executes it, returning stdout.
func (p *PlaywrightCLIAdapter) runScript(
	ctx context.Context, script string,
) (string, error) {
	dir := p.scriptDir
	if dir == "" {
		dir = os.TempDir()
	}
	scriptPath := filepath.Join(dir, "pw_script.js")
	if err := os.WriteFile(
		scriptPath, []byte(script), 0o644,
	); err != nil {
		return "", fmt.Errorf("write script: %w", err)
	}

	cmd := playwrightCommandFunc(ctx, "node", scriptPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf(
			"playwright script failed: %w\nstderr: %s",
			err, stderr.String(),
		)
	}
	return stdout.String(), nil
}
