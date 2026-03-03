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

// AppiumAdapter implements MobileAdapter using the Appium 2.0
// server (W3C WebDriver protocol with Appium extensions). It
// supports both Android and iOS platforms through a unified
// interface. Appium must be running as a separate server.
//
// Usage:
//
//	adapter := NewAppiumAdapter("http://localhost:4723",
//	    AppiumCapabilities{
//	        PlatformName:  "Android",
//	        AutomationName: "UiAutomator2",
//	        App:           "/path/to/app.apk",
//	        DeviceName:    "emulator-5554",
//	    },
//	)
//	err := adapter.Initialize(ctx)
type AppiumAdapter struct {
	serverURL    string
	capabilities AppiumCapabilities
	sessionID    string
	httpClient   *http.Client
	config       MobileConfig
}

// AppiumCapabilities defines the W3C + Appium desired
// capabilities for session creation.
type AppiumCapabilities struct {
	// PlatformName is "Android" or "iOS".
	PlatformName string `json:"platformName"`
	// AutomationName is the automation engine
	// (e.g., "UiAutomator2", "XCUITest", "Espresso").
	AutomationName string `json:"appium:automationName"`
	// DeviceName identifies the target device or emulator.
	DeviceName string `json:"appium:deviceName"`
	// App is the path to the application binary (APK/IPA).
	App string `json:"appium:app,omitempty"`
	// AppPackage is the Android app package name.
	AppPackage string `json:"appium:appPackage,omitempty"`
	// AppActivity is the Android app launch activity.
	AppActivity string `json:"appium:appActivity,omitempty"`
	// BundleID is the iOS app bundle identifier.
	BundleID string `json:"appium:bundleId,omitempty"`
	// PlatformVersion targets a specific OS version.
	PlatformVersion string `json:"appium:platformVersion,omitempty"`
	// NoReset keeps app data between sessions.
	NoReset bool `json:"appium:noReset,omitempty"`
	// FullReset reinstalls the app between sessions.
	FullReset bool `json:"appium:fullReset,omitempty"`
	// NewCommandTimeout is the idle timeout in seconds.
	NewCommandTimeout int `json:"appium:newCommandTimeout,omitempty"`
}

// Compile-time interface check.
var _ MobileAdapter = (*AppiumAdapter)(nil)

// NewAppiumAdapter creates an AppiumAdapter that connects to an
// Appium server at the given URL with the specified capabilities.
func NewAppiumAdapter(
	serverURL string, caps AppiumCapabilities,
) *AppiumAdapter {
	config := MobileConfig{
		PackageName:  caps.AppPackage,
		ActivityName: caps.AppActivity,
		DeviceSerial: caps.DeviceName,
	}
	return &AppiumAdapter{
		serverURL:    strings.TrimRight(serverURL, "/"),
		capabilities: caps,
		config:       config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// initialize creates a new Appium session.
func (a *AppiumAdapter) initialize(ctx context.Context) error {
	capsMap := map[string]interface{}{
		"platformName":          a.capabilities.PlatformName,
		"appium:automationName": a.capabilities.AutomationName,
		"appium:deviceName":     a.capabilities.DeviceName,
	}
	if a.capabilities.App != "" {
		capsMap["appium:app"] = a.capabilities.App
	}
	if a.capabilities.AppPackage != "" {
		capsMap["appium:appPackage"] = a.capabilities.AppPackage
	}
	if a.capabilities.AppActivity != "" {
		capsMap["appium:appActivity"] = a.capabilities.AppActivity
	}
	if a.capabilities.BundleID != "" {
		capsMap["appium:bundleId"] = a.capabilities.BundleID
	}
	if a.capabilities.PlatformVersion != "" {
		capsMap["appium:platformVersion"] = a.capabilities.PlatformVersion
	}
	if a.capabilities.NoReset {
		capsMap["appium:noReset"] = true
	}
	if a.capabilities.FullReset {
		capsMap["appium:fullReset"] = true
	}
	if a.capabilities.NewCommandTimeout > 0 {
		capsMap["appium:newCommandTimeout"] = a.capabilities.NewCommandTimeout
	}

	body := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"alwaysMatch": capsMap,
		},
	}

	resp, err := a.appiumPost(ctx, "/session", body)
	if err != nil {
		return fmt.Errorf("create appium session: %w", err)
	}

	sessionID := extractSessionID(resp)
	if sessionID == "" {
		return fmt.Errorf(
			"no sessionId in appium response: %v", resp,
		)
	}
	a.sessionID = sessionID
	return nil
}

// IsDeviceAvailable checks if the configured device is
// connected by querying the Appium server status.
func (a *AppiumAdapter) IsDeviceAvailable(
	ctx context.Context,
) (bool, error) {
	resp, err := a.appiumGet(ctx, "/status")
	if err != nil {
		return false, fmt.Errorf("device check: %w", err)
	}
	if val, ok := resp["value"].(map[string]interface{}); ok {
		if ready, ok := val["ready"].(bool); ok {
			return ready, nil
		}
	}
	return true, nil
}

// InstallApp installs the application from the given path onto
// the device via the Appium install endpoint.
func (a *AppiumAdapter) InstallApp(
	ctx context.Context, appPath string,
) error {
	if a.sessionID == "" {
		if err := a.initialize(ctx); err != nil {
			return err
		}
	}
	_, err := a.appiumPost(
		ctx,
		a.sessPath("/appium/device/install_app"),
		map[string]interface{}{"appPath": appPath},
	)
	if err != nil {
		return fmt.Errorf("install app: %w", err)
	}
	return nil
}

// LaunchApp starts the configured application on the device.
func (a *AppiumAdapter) LaunchApp(
	ctx context.Context,
) error {
	if a.sessionID == "" {
		if err := a.initialize(ctx); err != nil {
			return err
		}
	}
	_, err := a.appiumPost(
		ctx,
		a.sessPath("/appium/app/launch"),
		map[string]interface{}{},
	)
	if err != nil {
		return fmt.Errorf("launch app: %w", err)
	}
	return nil
}

// StopApp stops the running application on the device.
func (a *AppiumAdapter) StopApp(
	ctx context.Context,
) error {
	if a.sessionID == "" {
		return nil
	}
	_, err := a.appiumPost(
		ctx,
		a.sessPath("/appium/app/close"),
		map[string]interface{}{},
	)
	if err != nil {
		return fmt.Errorf("stop app: %w", err)
	}
	return nil
}

// IsAppRunning checks if the application is currently running
// on the device.
func (a *AppiumAdapter) IsAppRunning(
	ctx context.Context,
) (bool, error) {
	if a.sessionID == "" {
		return false, nil
	}
	bundleOrPkg := a.config.PackageName
	if a.capabilities.BundleID != "" {
		bundleOrPkg = a.capabilities.BundleID
	}
	resp, err := a.appiumPost(
		ctx,
		a.sessPath("/appium/device/app_state"),
		map[string]interface{}{
			"appId": bundleOrPkg,
		},
	)
	if err != nil {
		return false, fmt.Errorf("app state: %w", err)
	}
	// State 4 = running in foreground, 3 = running in background.
	if val, ok := resp["value"].(float64); ok {
		return val >= 3, nil
	}
	return false, nil
}

// TakeScreenshot captures the current device screen as a PNG
// image.
func (a *AppiumAdapter) TakeScreenshot(
	ctx context.Context,
) ([]byte, error) {
	if a.sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}
	resp, err := a.appiumGet(
		ctx, a.sessPath("/screenshot"),
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

// Tap performs a tap gesture at the given screen coordinates.
func (a *AppiumAdapter) Tap(
	ctx context.Context, x, y int,
) error {
	if a.sessionID == "" {
		return fmt.Errorf("no active session")
	}
	action := map[string]interface{}{
		"actions": []interface{}{
			map[string]interface{}{
				"type": "pointer",
				"id":   "finger1",
				"parameters": map[string]interface{}{
					"pointerType": "touch",
				},
				"actions": []interface{}{
					map[string]interface{}{
						"type":     "pointerMove",
						"duration": 0,
						"x":        x,
						"y":        y,
					},
					map[string]interface{}{
						"type":   "pointerDown",
						"button": 0,
					},
					map[string]interface{}{
						"type":   "pointerUp",
						"button": 0,
					},
				},
			},
		},
	}
	_, err := a.appiumPost(
		ctx, a.sessPath("/actions"), action,
	)
	if err != nil {
		return fmt.Errorf(
			"tap at (%d,%d): %w", x, y, err,
		)
	}
	return nil
}

// SendKeys types the given text into the currently focused
// input element.
func (a *AppiumAdapter) SendKeys(
	ctx context.Context, text string,
) error {
	if a.sessionID == "" {
		return fmt.Errorf("no active session")
	}
	// Find the currently active element.
	resp, err := a.appiumGet(
		ctx, a.sessPath("/element/active"),
	)
	if err != nil {
		return fmt.Errorf("find active element: %w", err)
	}
	elemID := extractAppiumElementID(resp)
	if elemID == "" {
		return fmt.Errorf("no active element found")
	}

	_, err = a.appiumPost(
		ctx,
		a.sessPath("/element/"+elemID+"/value"),
		map[string]interface{}{"text": text},
	)
	if err != nil {
		return fmt.Errorf("send keys: %w", err)
	}
	return nil
}

// PressKey sends a key event to the device (e.g.,
// "KEYCODE_BACK", "KEYCODE_HOME").
func (a *AppiumAdapter) PressKey(
	ctx context.Context, keycode string,
) error {
	if a.sessionID == "" {
		return fmt.Errorf("no active session")
	}
	// Use Appium's mobile: pressKey for Android.
	_, err := a.appiumPost(
		ctx,
		a.sessPath("/appium/device/press_keycode"),
		map[string]interface{}{
			"keycode": keycodeToInt(keycode),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"press key %s: %w", keycode, err,
		)
	}
	return nil
}

// WaitForApp waits until the application is fully launched and
// responsive, up to the given timeout.
func (a *AppiumAdapter) WaitForApp(
	ctx context.Context, timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	interval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		running, _ := a.IsAppRunning(ctx)
		if running {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf(
		"app not running after %v", timeout,
	)
}

// RunInstrumentedTests runs instrumented tests on the device
// via the Appium session. For Android, this uses the
// mobile:shell command to execute am instrument.
func (a *AppiumAdapter) RunInstrumentedTests(
	ctx context.Context, testClass string,
) (*TestResult, error) {
	if a.sessionID == "" {
		if err := a.initialize(ctx); err != nil {
			return nil, err
		}
	}

	runner := a.config.PackageName + ".test/androidx.test.runner.AndroidJUnitRunner"
	args := []string{
		"am", "instrument", "-w",
	}
	if testClass != "" {
		args = append(args, "-e", "class", testClass)
	}
	args = append(args, runner)

	resp, err := a.appiumPost(
		ctx,
		a.sessPath("/appium/execute_driver"),
		map[string]interface{}{
			"script": fmt.Sprintf(
				"return await driver.execute('mobile: shell', "+
					"{command: '%s', args: %s});",
				args[0], toJSArray(args[1:]),
			),
			"type": "webdriverio",
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"run instrumented tests: %w", err,
		)
	}

	output, _ := resp["value"].(string)
	return &TestResult{
		Output: output,
		Suites: []TestSuite{{
			Name:  "instrumented",
			Tests: 1,
		}},
		TotalTests: 1,
	}, nil
}

// Close terminates the Appium session and releases resources.
func (a *AppiumAdapter) Close(
	ctx context.Context,
) error {
	if a.sessionID == "" {
		return nil
	}
	_, err := a.appiumDelete(ctx, a.sessPath(""))
	a.sessionID = ""
	if err != nil {
		return fmt.Errorf("close appium session: %w", err)
	}
	return nil
}

// Available returns true if the Appium server is reachable.
func (a *AppiumAdapter) Available(
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

// sessPath returns the Appium path prefixed with the current
// session ID.
func (a *AppiumAdapter) sessPath(suffix string) string {
	return "/session/" + a.sessionID + suffix
}

// appiumPost sends a POST request to the Appium server.
func (a *AppiumAdapter) appiumPost(
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

	return a.doAppiumRequest(req)
}

// appiumGet sends a GET request to the Appium server.
func (a *AppiumAdapter) appiumGet(
	ctx context.Context, path string,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		a.serverURL+path, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return a.doAppiumRequest(req)
}

// appiumDelete sends a DELETE request to the Appium server.
func (a *AppiumAdapter) appiumDelete(
	ctx context.Context, path string,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete,
		a.serverURL+path, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return a.doAppiumRequest(req)
}

// doAppiumRequest executes an HTTP request and parses the JSON
// response from the Appium server.
func (a *AppiumAdapter) doAppiumRequest(
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
	if err := json.Unmarshal(
		respBody, &result,
	); err != nil {
		return nil, fmt.Errorf(
			"parse response (status %d): %w\nbody: %s",
			resp.StatusCode, err, string(respBody),
		)
	}

	if resp.StatusCode >= 400 {
		errMsg := "unknown error"
		if val, ok := result["value"].(map[string]interface{}); ok {
			if msg, ok := val["message"].(string); ok {
				errMsg = msg
			}
		}
		return nil, fmt.Errorf(
			"appium error (status %d): %s",
			resp.StatusCode, errMsg,
		)
	}

	return result, nil
}

// extractSessionID extracts the session ID from a W3C WebDriver
// response, handling both flat and nested formats.
func extractSessionID(
	resp map[string]interface{},
) string {
	if id, ok := resp["sessionId"].(string); ok {
		return id
	}
	if val, ok := resp["value"].(map[string]interface{}); ok {
		if id, ok := val["sessionId"].(string); ok {
			return id
		}
	}
	return ""
}

// extractAppiumElementID extracts the element ID from an
// Appium/WebDriver element response.
func extractAppiumElementID(
	resp map[string]interface{},
) string {
	val, ok := resp["value"].(map[string]interface{})
	if !ok {
		return ""
	}
	for _, key := range []string{
		"element-6066-11e4-a52e-4f735466cecf",
		"ELEMENT",
	} {
		if id, ok := val[key].(string); ok {
			return id
		}
	}
	return ""
}

// keycodeToInt maps Android keycode names to their integer
// values. Returns 0 for unknown keycodes.
func keycodeToInt(keycode string) int {
	keycodes := map[string]int{
		"KEYCODE_HOME":           3,
		"KEYCODE_BACK":           4,
		"KEYCODE_MENU":           82,
		"KEYCODE_SEARCH":         84,
		"KEYCODE_ENTER":          66,
		"KEYCODE_DEL":            67,
		"KEYCODE_VOLUME_UP":      24,
		"KEYCODE_VOLUME_DOWN":    25,
		"KEYCODE_POWER":          26,
		"KEYCODE_APP_SWITCH":     187,
		"KEYCODE_NOTIFICATION":   83,
		"KEYCODE_DPAD_UP":        19,
		"KEYCODE_DPAD_DOWN":      20,
		"KEYCODE_DPAD_LEFT":      21,
		"KEYCODE_DPAD_RIGHT":     22,
		"KEYCODE_DPAD_CENTER":    23,
		"KEYCODE_TAB":            61,
		"KEYCODE_SPACE":          62,
		"KEYCODE_ESCAPE":         111,
		"KEYCODE_MEDIA_PLAY":     126,
		"KEYCODE_MEDIA_PAUSE":    127,
		"KEYCODE_MEDIA_NEXT":     87,
		"KEYCODE_MEDIA_PREVIOUS": 88,
	}
	if code, ok := keycodes[keycode]; ok {
		return code
	}
	return 0
}

// toJSArray converts a slice of strings to a JavaScript array
// literal for use in Appium execute driver scripts.
func toJSArray(args []string) string {
	if len(args) == 0 {
		return "[]"
	}
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = fmt.Sprintf("'%s'", escapeJSSingle(arg))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
