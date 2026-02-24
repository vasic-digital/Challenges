package userflow

import (
	"context"
	"time"
)

// InterceptedRequest represents an HTTP request captured by
// the browser's network interception.
type InterceptedRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// BrowserAdapter defines the interface for browser-based UI
// testing. Implementations may wrap Playwright, Puppeteer,
// Selenium, or other browser automation tools.
type BrowserAdapter interface {
	// Initialize sets up the browser with the given config.
	Initialize(
		ctx context.Context, config BrowserConfig,
	) error

	// Navigate loads the given URL in the browser.
	Navigate(ctx context.Context, url string) error

	// Click performs a click on the element matching the
	// selector.
	Click(ctx context.Context, selector string) error

	// Fill types a value into the input matching the selector.
	Fill(
		ctx context.Context, selector, value string,
	) error

	// SelectOption selects an option value in a dropdown
	// matching the selector.
	SelectOption(
		ctx context.Context, selector, value string,
	) error

	// IsVisible returns whether the element matching the
	// selector is currently visible.
	IsVisible(
		ctx context.Context, selector string,
	) (bool, error)

	// WaitForSelector waits until an element matching the
	// selector appears, up to the given timeout.
	WaitForSelector(
		ctx context.Context,
		selector string,
		timeout time.Duration,
	) error

	// GetText returns the text content of the element
	// matching the selector.
	GetText(
		ctx context.Context, selector string,
	) (string, error)

	// GetAttribute returns the value of the named attribute
	// on the element matching the selector.
	GetAttribute(
		ctx context.Context, selector, attr string,
	) (string, error)

	// Screenshot captures the current browser viewport as
	// a PNG image.
	Screenshot(ctx context.Context) ([]byte, error)

	// EvaluateJS executes JavaScript in the browser context
	// and returns the result as a string.
	EvaluateJS(
		ctx context.Context, script string,
	) (string, error)

	// NetworkIntercept sets up a handler for network requests
	// matching the given URL pattern.
	NetworkIntercept(
		ctx context.Context,
		pattern string,
		handler func(req *InterceptedRequest),
	) error

	// Close shuts down the browser and releases resources.
	Close(ctx context.Context) error

	// Available returns true if the browser automation tool
	// is installed and usable.
	Available(ctx context.Context) bool
}
