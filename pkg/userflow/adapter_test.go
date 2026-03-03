package userflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// stubBrowser implements BrowserAdapter for compile-time checks.
type stubBrowser struct{}

func (s *stubBrowser) Initialize(
	_ context.Context, _ BrowserConfig,
) error {
	return nil
}
func (s *stubBrowser) Navigate(
	_ context.Context, _ string,
) error {
	return nil
}
func (s *stubBrowser) Click(
	_ context.Context, _ string,
) error {
	return nil
}
func (s *stubBrowser) Fill(
	_ context.Context, _, _ string,
) error {
	return nil
}
func (s *stubBrowser) SelectOption(
	_ context.Context, _, _ string,
) error {
	return nil
}
func (s *stubBrowser) IsVisible(
	_ context.Context, _ string,
) (bool, error) {
	return false, nil
}
func (s *stubBrowser) WaitForSelector(
	_ context.Context, _ string, _ time.Duration,
) error {
	return nil
}
func (s *stubBrowser) GetText(
	_ context.Context, _ string,
) (string, error) {
	return "", nil
}
func (s *stubBrowser) GetAttribute(
	_ context.Context, _, _ string,
) (string, error) {
	return "", nil
}
func (s *stubBrowser) Screenshot(
	_ context.Context,
) ([]byte, error) {
	return nil, nil
}
func (s *stubBrowser) EvaluateJS(
	_ context.Context, _ string,
) (string, error) {
	return "", nil
}
func (s *stubBrowser) NetworkIntercept(
	_ context.Context,
	_ string,
	_ func(req *InterceptedRequest),
) error {
	return nil
}
func (s *stubBrowser) Close(_ context.Context) error {
	return nil
}
func (s *stubBrowser) Available(_ context.Context) bool {
	return false
}

// stubMobile implements MobileAdapter for compile-time checks.
type stubMobile struct{}

func (s *stubMobile) IsDeviceAvailable(
	_ context.Context,
) (bool, error) {
	return false, nil
}
func (s *stubMobile) InstallApp(
	_ context.Context, _ string,
) error {
	return nil
}
func (s *stubMobile) LaunchApp(_ context.Context) error {
	return nil
}
func (s *stubMobile) StopApp(_ context.Context) error {
	return nil
}
func (s *stubMobile) IsAppRunning(
	_ context.Context,
) (bool, error) {
	return false, nil
}
func (s *stubMobile) TakeScreenshot(
	_ context.Context,
) ([]byte, error) {
	return nil, nil
}
func (s *stubMobile) Tap(
	_ context.Context, _, _ int,
) error {
	return nil
}
func (s *stubMobile) SendKeys(
	_ context.Context, _ string,
) error {
	return nil
}
func (s *stubMobile) PressKey(
	_ context.Context, _ string,
) error {
	return nil
}
func (s *stubMobile) WaitForApp(
	_ context.Context, _ time.Duration,
) error {
	return nil
}
func (s *stubMobile) RunInstrumentedTests(
	_ context.Context, _ string,
) (*TestResult, error) {
	return nil, nil
}
func (s *stubMobile) Close(_ context.Context) error {
	return nil
}
func (s *stubMobile) Available(_ context.Context) bool {
	return false
}

// stubDesktop implements DesktopAdapter for compile-time checks.
type stubDesktop struct{}

func (s *stubDesktop) LaunchApp(
	_ context.Context, _ DesktopAppConfig,
) error {
	return nil
}
func (s *stubDesktop) IsAppRunning(
	_ context.Context,
) (bool, error) {
	return false, nil
}
func (s *stubDesktop) Navigate(
	_ context.Context, _ string,
) error {
	return nil
}
func (s *stubDesktop) Click(
	_ context.Context, _ string,
) error {
	return nil
}
func (s *stubDesktop) Fill(
	_ context.Context, _, _ string,
) error {
	return nil
}
func (s *stubDesktop) IsVisible(
	_ context.Context, _ string,
) (bool, error) {
	return false, nil
}
func (s *stubDesktop) WaitForSelector(
	_ context.Context, _ string, _ time.Duration,
) error {
	return nil
}
func (s *stubDesktop) Screenshot(
	_ context.Context,
) ([]byte, error) {
	return nil, nil
}
func (s *stubDesktop) InvokeCommand(
	_ context.Context, _ string, _ ...string,
) (string, error) {
	return "", nil
}
func (s *stubDesktop) WaitForWindow(
	_ context.Context, _ time.Duration,
) error {
	return nil
}
func (s *stubDesktop) Close(_ context.Context) error {
	return nil
}
func (s *stubDesktop) Available(_ context.Context) bool {
	return false
}

// stubAPI implements APIAdapter for compile-time checks.
type stubAPI struct{}

func (s *stubAPI) Login(
	_ context.Context, _ Credentials,
) (string, error) {
	return "", nil
}
func (s *stubAPI) LoginWithRetry(
	_ context.Context, _ Credentials, _ int,
) (string, error) {
	return "", nil
}
func (s *stubAPI) Get(
	_ context.Context, _ string,
) (int, map[string]interface{}, error) {
	return 0, nil, nil
}
func (s *stubAPI) GetRaw(
	_ context.Context, _ string,
) (int, []byte, error) {
	return 0, nil, nil
}
func (s *stubAPI) GetArray(
	_ context.Context, _ string,
) (int, []interface{}, error) {
	return 0, nil, nil
}
func (s *stubAPI) PostJSON(
	_ context.Context, _, _ string,
) (int, []byte, error) {
	return 0, nil, nil
}
func (s *stubAPI) PutJSON(
	_ context.Context, _, _ string,
) (int, []byte, error) {
	return 0, nil, nil
}
func (s *stubAPI) Delete(
	_ context.Context, _ string,
) (int, []byte, error) {
	return 0, nil, nil
}
func (s *stubAPI) DeleteWithBody(
	_ context.Context, _, _ string,
) (int, []byte, error) {
	return 0, nil, nil
}
func (s *stubAPI) WebSocketConnect(
	_ context.Context, _ string,
) (WebSocketConn, error) {
	return nil, nil
}
func (s *stubAPI) SetToken(_ string)          {}
func (s *stubAPI) Available(_ context.Context) bool {
	return false
}

// stubBuild implements BuildAdapter for compile-time checks.
type stubBuild struct{}

func (s *stubBuild) Build(
	_ context.Context, _ BuildTarget,
) (*BuildResult, error) {
	return nil, nil
}
func (s *stubBuild) RunTests(
	_ context.Context, _ TestTarget,
) (*TestResult, error) {
	return nil, nil
}
func (s *stubBuild) Lint(
	_ context.Context, _ LintTarget,
) (*LintResult, error) {
	return nil, nil
}
func (s *stubBuild) Available(_ context.Context) bool {
	return false
}

// stubProcess implements ProcessAdapter for compile-time checks.
type stubProcess struct{}

func (s *stubProcess) Launch(
	_ context.Context, _ ProcessConfig,
) error {
	return nil
}
func (s *stubProcess) IsRunning() bool { return false }
func (s *stubProcess) WaitForReady(
	_ context.Context, _ time.Duration,
) error {
	return nil
}
func (s *stubProcess) Stop() error { return nil }

// stubWebSocket implements WebSocketConn for compile-time
// checks.
type stubWebSocket struct{}

func (s *stubWebSocket) WriteMessage(_ []byte) error {
	return nil
}
func (s *stubWebSocket) ReadMessage() ([]byte, error) {
	return nil, nil
}
func (s *stubWebSocket) Close() error { return nil }

// Compile-time interface satisfaction checks.
var (
	_ BrowserAdapter = (*stubBrowser)(nil)
	_ MobileAdapter  = (*stubMobile)(nil)
	_ DesktopAdapter = (*stubDesktop)(nil)
	_ APIAdapter     = (*stubAPI)(nil)
	_ BuildAdapter   = (*stubBuild)(nil)
	_ ProcessAdapter = (*stubProcess)(nil)
	_ WebSocketConn  = (*stubWebSocket)(nil)
)

func TestBrowserAdapter_Satisfies(t *testing.T) {
	var a BrowserAdapter = &stubBrowser{}
	assert.NotNil(t, a)
	assert.False(t, a.Available(context.Background()))
}

func TestMobileAdapter_Satisfies(t *testing.T) {
	var a MobileAdapter = &stubMobile{}
	assert.NotNil(t, a)
	assert.False(t, a.Available(context.Background()))
}

func TestDesktopAdapter_Satisfies(t *testing.T) {
	var a DesktopAdapter = &stubDesktop{}
	assert.NotNil(t, a)
	assert.False(t, a.Available(context.Background()))
}

func TestAPIAdapter_Satisfies(t *testing.T) {
	var a APIAdapter = &stubAPI{}
	assert.NotNil(t, a)
	assert.False(t, a.Available(context.Background()))
}

func TestBuildAdapter_Satisfies(t *testing.T) {
	var a BuildAdapter = &stubBuild{}
	assert.NotNil(t, a)
	assert.False(t, a.Available(context.Background()))
}

func TestProcessAdapter_Satisfies(t *testing.T) {
	var a ProcessAdapter = &stubProcess{}
	assert.NotNil(t, a)
	assert.False(t, a.IsRunning())
}

func TestWebSocketConn_Satisfies(t *testing.T) {
	var c WebSocketConn = &stubWebSocket{}
	assert.NotNil(t, c)
}
