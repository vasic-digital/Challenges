package userflow

import (
	"bytes"
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

// CypressCLIAdapter implements BrowserAdapter by executing
// Cypress CLI commands. Each browser action generates a minimal
// Cypress spec file, writes it to a temporary directory, and
// invokes `npx cypress run` to execute the spec. Results are
// parsed from Cypress stdout.
//
// Usage:
//
//	adapter := NewCypressCLIAdapter("/path/to/project")
//	err := adapter.Initialize(ctx, BrowserConfig{
//	    BrowserType: "chrome",
//	    Headless:    true,
//	})
type CypressCLIAdapter struct {
	projectDir string
	baseURL    string
	headless   bool
	browser    string
	tempDir    string
}

// Compile-time interface check.
var _ BrowserAdapter = (*CypressCLIAdapter)(nil)

// NewCypressCLIAdapter creates a CypressCLIAdapter that runs
// Cypress specs from the given project directory. The directory
// should contain a valid Cypress configuration
// (cypress.config.js or similar).
func NewCypressCLIAdapter(
	projectDir string,
) *CypressCLIAdapter {
	return &CypressCLIAdapter{
		projectDir: projectDir,
		headless:   true,
		browser:    "chrome",
	}
}

// Initialize sets up the adapter with the given browser config.
// Creates a temporary directory for generated spec files.
func (a *CypressCLIAdapter) Initialize(
	ctx context.Context, config BrowserConfig,
) error {
	if config.BrowserType != "" {
		a.browser = cypressBrowserName(config.BrowserType)
	}
	a.headless = config.Headless

	dir, err := os.MkdirTemp("", "cypress-specs-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	a.tempDir = dir
	return nil
}

// Navigate loads the given URL in the browser by generating
// and executing a Cypress spec that calls cy.visit().
func (a *CypressCLIAdapter) Navigate(
	ctx context.Context, url string,
) error {
	a.baseURL = url
	spec := fmt.Sprintf(`
describe('navigate', () => {
  it('visits url', () => {
    cy.visit('%s');
  });
});
`, escapeJSSingle(url))

	_, err := a.runSpec(ctx, spec)
	if err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}
	return nil
}

// Click performs a click on the element matching the CSS
// selector by generating a Cypress spec with cy.get().click().
func (a *CypressCLIAdapter) Click(
	ctx context.Context, selector string,
) error {
	spec := a.wrapSpec(fmt.Sprintf(
		"cy.get('%s').click();",
		escapeJSSingle(selector),
	))

	_, err := a.runSpec(ctx, spec)
	if err != nil {
		return fmt.Errorf("click %s: %w", selector, err)
	}
	return nil
}

// Fill types a value into the input matching the CSS selector.
// Clears the field first using cy.clear(), then types with
// cy.type().
func (a *CypressCLIAdapter) Fill(
	ctx context.Context, selector, value string,
) error {
	spec := a.wrapSpec(fmt.Sprintf(
		"cy.get('%s').clear().type('%s');",
		escapeJSSingle(selector),
		escapeJSSingle(value),
	))

	_, err := a.runSpec(ctx, spec)
	if err != nil {
		return fmt.Errorf("fill %s: %w", selector, err)
	}
	return nil
}

// SelectOption selects an option in a dropdown matching the
// CSS selector using cy.select().
func (a *CypressCLIAdapter) SelectOption(
	ctx context.Context, selector, value string,
) error {
	spec := a.wrapSpec(fmt.Sprintf(
		"cy.get('%s').select('%s');",
		escapeJSSingle(selector),
		escapeJSSingle(value),
	))

	_, err := a.runSpec(ctx, spec)
	if err != nil {
		return fmt.Errorf(
			"select option %s: %w", selector, err,
		)
	}
	return nil
}

// IsVisible returns whether the element matching the CSS
// selector is currently visible. Parses Cypress stdout for
// the visibility result.
func (a *CypressCLIAdapter) IsVisible(
	ctx context.Context, selector string,
) (bool, error) {
	spec := fmt.Sprintf(`
describe('visibility', () => {
  it('checks visibility', () => {
    cy.visit('%s');
    cy.get('body').then(($body) => {
      const el = $body.find('%s');
      const visible = el.length > 0 && el.is(':visible');
      cy.task('log', JSON.stringify({ visible }));
    });
  });
});
`, escapeJSSingle(a.baseURL), escapeJSSingle(selector))

	out, err := a.runSpec(ctx, spec)
	if err != nil {
		// Element not found or spec failed = not visible.
		return false, nil
	}

	// Parse the task log output for the visibility result.
	if strings.Contains(out, `"visible":true`) {
		return true, nil
	}
	return false, nil
}

// WaitForSelector waits until an element matching the CSS
// selector appears, up to the given timeout. Uses Cypress
// retry with a custom timeout.
func (a *CypressCLIAdapter) WaitForSelector(
	ctx context.Context,
	selector string,
	timeout time.Duration,
) error {
	spec := a.wrapSpec(fmt.Sprintf(
		"cy.get('%s', { timeout: %d }).should('exist');",
		escapeJSSingle(selector),
		timeout.Milliseconds(),
	))

	_, err := a.runSpec(ctx, spec)
	if err != nil {
		return fmt.Errorf(
			"wait for %s: %w", selector, err,
		)
	}
	return nil
}

// GetText returns the text content of the element matching
// the CSS selector. Logs the text via cy.task and parses
// stdout.
func (a *CypressCLIAdapter) GetText(
	ctx context.Context, selector string,
) (string, error) {
	spec := fmt.Sprintf(`
describe('getText', () => {
  it('reads text', () => {
    cy.visit('%s');
    cy.get('%s').invoke('text').then((text) => {
      cy.task('log', JSON.stringify({ text }));
    });
  });
});
`, escapeJSSingle(a.baseURL), escapeJSSingle(selector))

	out, err := a.runSpec(ctx, spec)
	if err != nil {
		return "", fmt.Errorf(
			"get text %s: %w", selector, err,
		)
	}

	return a.parseTaskLogString(out, "text")
}

// GetAttribute returns the value of the named attribute on
// the element matching the CSS selector.
func (a *CypressCLIAdapter) GetAttribute(
	ctx context.Context, selector, attr string,
) (string, error) {
	spec := fmt.Sprintf(`
describe('getAttribute', () => {
  it('reads attribute', () => {
    cy.visit('%s');
    cy.get('%s').invoke('attr', '%s').then((val) => {
      cy.task('log', JSON.stringify({ value: val || '' }));
    });
  });
});
`, escapeJSSingle(a.baseURL),
		escapeJSSingle(selector),
		escapeJSSingle(attr))

	out, err := a.runSpec(ctx, spec)
	if err != nil {
		return "", fmt.Errorf(
			"get attribute %s: %w", attr, err,
		)
	}

	return a.parseTaskLogString(out, "value")
}

// Screenshot captures the current browser viewport as a PNG
// image using Cypress screenshot capabilities.
func (a *CypressCLIAdapter) Screenshot(
	ctx context.Context,
) ([]byte, error) {
	screenshotName := fmt.Sprintf(
		"capture-%d", time.Now().UnixNano(),
	)
	spec := a.wrapSpec(fmt.Sprintf(
		"cy.screenshot('%s');",
		escapeJSSingle(screenshotName),
	))

	_, err := a.runSpec(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}

	// Cypress saves screenshots to cypress/screenshots/.
	pattern := filepath.Join(
		a.projectDir,
		"cypress", "screenshots", "**",
		screenshotName+".png",
	)
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		// Try a simpler path pattern.
		pattern = filepath.Join(
			a.projectDir,
			"cypress", "screenshots",
			"*", screenshotName+".png",
		)
		matches, _ = filepath.Glob(pattern)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf(
			"screenshot file not found: %s",
			screenshotName,
		)
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		return nil, fmt.Errorf(
			"read screenshot: %w", err,
		)
	}
	return data, nil
}

// EvaluateJS executes JavaScript in the browser context using
// cy.window().then() and returns the result as a string. The
// script is base64-encoded to avoid quoting issues in the
// generated spec.
func (a *CypressCLIAdapter) EvaluateJS(
	ctx context.Context, script string,
) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(
		[]byte(script),
	)
	spec := fmt.Sprintf(`
describe('evaluateJS', () => {
  it('runs script', () => {
    cy.visit('%s');
    cy.window().then((win) => {
      const code = atob('%s');
      const fn = new win.Function(code);
      const result = fn.call(win);
      cy.task('log', JSON.stringify({ result }));
    });
  });
});
`, escapeJSSingle(a.baseURL), encoded)

	out, err := a.runSpec(ctx, spec)
	if err != nil {
		return "", fmt.Errorf("evaluate js: %w", err)
	}

	return a.parseTaskLogString(out, "result")
}

// NetworkIntercept is not natively supported by the Cypress
// CLI adapter. Returns nil to indicate no error but the
// handler will not fire. Use cy.intercept() inside spec files
// for network interception.
func (a *CypressCLIAdapter) NetworkIntercept(
	_ context.Context,
	_ string,
	_ func(req *InterceptedRequest),
) error {
	return nil
}

// Close removes the temporary spec directory and releases
// resources.
func (a *CypressCLIAdapter) Close(
	_ context.Context,
) error {
	if a.tempDir != "" {
		os.RemoveAll(a.tempDir)
		a.tempDir = ""
	}
	return nil
}

// Available returns true if Cypress is installed and runnable
// via npx.
func (a *CypressCLIAdapter) Available(
	_ context.Context,
) bool {
	cmd := exec.Command("npx", "cypress", "--version")
	err := cmd.Run()
	return err == nil
}

// --- internal helpers ---

// wrapSpec wraps a Cypress command string into a full
// describe/it block that visits the base URL first.
func (a *CypressCLIAdapter) wrapSpec(
	command string,
) string {
	return fmt.Sprintf(`
describe('action', () => {
  it('performs action', () => {
    cy.visit('%s');
    %s
  });
});
`, escapeJSSingle(a.baseURL), command)
}

// runSpec writes a spec file to the temp directory and
// executes it via `npx cypress run`.
func (a *CypressCLIAdapter) runSpec(
	ctx context.Context, spec string,
) (string, error) {
	specFile := filepath.Join(
		a.tempDir,
		fmt.Sprintf(
			"spec_%d.cy.js", time.Now().UnixNano(),
		),
	)
	if err := os.WriteFile(
		specFile, []byte(spec), 0644,
	); err != nil {
		return "", fmt.Errorf(
			"write spec file: %w", err,
		)
	}
	defer os.Remove(specFile)

	args := []string{
		"cypress", "run",
		"--spec", specFile,
		"--browser", a.browser,
	}
	if a.baseURL != "" {
		args = append(
			args,
			"--config",
			fmt.Sprintf("baseUrl=%s", a.baseURL),
		)
	}
	if a.headless {
		args = append(args, "--headless")
	}

	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Dir = a.projectDir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf(
			"cypress run: %w\noutput: %s",
			err, buf.String(),
		)
	}
	return buf.String(), nil
}

// parseTaskLogString extracts a string field from a
// JSON-formatted cy.task('log', ...) output line in Cypress
// stdout.
func (a *CypressCLIAdapter) parseTaskLogString(
	output, field string,
) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, field) {
			continue
		}
		// Try to find JSON in the line.
		start := strings.Index(line, "{")
		end := strings.LastIndex(line, "}")
		if start < 0 || end < start {
			continue
		}
		jsonStr := line[start : end+1]
		var result map[string]interface{}
		if err := json.Unmarshal(
			[]byte(jsonStr), &result,
		); err != nil {
			continue
		}
		if val, ok := result[field].(string); ok {
			return val, nil
		}
		// Handle non-string values.
		if val, ok := result[field]; ok {
			b, _ := json.Marshal(val)
			return string(b), nil
		}
	}
	return "", nil
}

// cypressBrowserName maps BrowserConfig.BrowserType to a
// Cypress-compatible browser name.
func cypressBrowserName(browserType string) string {
	switch strings.ToLower(browserType) {
	case "chrome", "chromium":
		return "chrome"
	case "firefox", "gecko":
		return "firefox"
	case "edge", "msedge":
		return "edge"
	case "electron":
		return "electron"
	default:
		return "chrome"
	}
}
