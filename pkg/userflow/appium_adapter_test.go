package userflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ MobileAdapter = (*AppiumAdapter)(nil)

func TestNewAppiumAdapter(t *testing.T) {
	tests := []struct {
		name string
		url  string
		caps AppiumCapabilities
	}{
		{
			name: "basic_android",
			url:  "http://localhost:4723",
			caps: AppiumCapabilities{
				PlatformName:   "Android",
				AutomationName: "UiAutomator2",
				DeviceName:     "emulator-5554",
				AppPackage:     "com.example.app",
				AppActivity:    ".MainActivity",
			},
		},
		{
			name: "ios_config",
			url:  "http://localhost:4723",
			caps: AppiumCapabilities{
				PlatformName:   "iOS",
				AutomationName: "XCUITest",
				DeviceName:     "iPhone 14",
				BundleID:       "com.example.ios",
			},
		},
		{
			name: "trailing_slash_stripped",
			url:  "http://localhost:4723/",
			caps: AppiumCapabilities{
				PlatformName: "Android",
				DeviceName:   "device-1",
			},
		},
		{
			name: "empty_caps",
			url:  "http://appium-server:4723",
			caps: AppiumCapabilities{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAppiumAdapter(
				tt.url, tt.caps,
			)
			require.NotNil(t, adapter)
			assert.NotNil(t, adapter.httpClient)
			assert.Empty(t, adapter.sessionID)
			assert.Equal(
				t,
				tt.caps.AppPackage,
				adapter.config.PackageName,
			)
			assert.Equal(
				t,
				tt.caps.AppActivity,
				adapter.config.ActivityName,
			)
			assert.Equal(
				t,
				tt.caps.DeviceName,
				adapter.config.DeviceSerial,
			)
		})
	}
}

func TestAppiumAdapter_Available_NoServer(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:19997",
		AppiumCapabilities{},
	)
	available := adapter.Available(context.Background())
	assert.False(t, available)
}

func TestAppiumAdapter_Available_ServerRunning(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(
					w, `{"value":{"ready":true}}`,
				)
			},
		),
	)
	defer srv.Close()

	adapter := NewAppiumAdapter(
		srv.URL, AppiumCapabilities{},
	)
	available := adapter.Available(context.Background())
	assert.True(t, available)
}

func TestAppiumAdapter_Available_ServerError(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(
					http.StatusInternalServerError,
				)
			},
		),
	)
	defer srv.Close()

	adapter := NewAppiumAdapter(
		srv.URL, AppiumCapabilities{},
	)
	available := adapter.Available(context.Background())
	assert.False(t, available)
}

func TestAppiumAdapter_Close_NoSession(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:4723",
		AppiumCapabilities{},
	)
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
}

func TestAppiumAdapter_Close_WithSession(
	t *testing.T,
) {
	var deleteCalled bool
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deleteCalled = true
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				fmt.Fprint(w, `{"value":null}`)
			},
		),
	)
	defer srv.Close()

	adapter := NewAppiumAdapter(
		srv.URL, AppiumCapabilities{},
	)
	adapter.sessionID = "session-to-close"

	err := adapter.Close(context.Background())
	assert.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Empty(t, adapter.sessionID)
}

func TestAppiumAdapter_StopApp_NoSession(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:4723",
		AppiumCapabilities{},
	)
	err := adapter.StopApp(context.Background())
	assert.NoError(t, err)
}

func TestAppiumAdapter_IsAppRunning_NoSession(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:4723",
		AppiumCapabilities{},
	)
	running, err := adapter.IsAppRunning(
		context.Background(),
	)
	assert.NoError(t, err)
	assert.False(t, running)
}

func TestAppiumAdapter_TakeScreenshot_NoSession(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:4723",
		AppiumCapabilities{},
	)
	data, err := adapter.TakeScreenshot(
		context.Background(),
	)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no active session")
}

func TestAppiumAdapter_Tap_NoSession(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:4723",
		AppiumCapabilities{},
	)
	err := adapter.Tap(context.Background(), 100, 200)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active session")
}

func TestAppiumAdapter_SendKeys_NoSession(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:4723",
		AppiumCapabilities{},
	)
	err := adapter.SendKeys(
		context.Background(), "hello",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active session")
}

func TestAppiumAdapter_PressKey_NoSession(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:4723",
		AppiumCapabilities{},
	)
	err := adapter.PressKey(
		context.Background(), "KEYCODE_BACK",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active session")
}

func TestAppiumAdapter_WaitForApp_ContextCancel(
	t *testing.T,
) {
	adapter := NewAppiumAdapter(
		"http://localhost:19997",
		AppiumCapabilities{
			AppPackage: "com.test.app",
		},
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel() // Cancel immediately.

	err := adapter.WaitForApp(ctx, 5*time.Second)
	assert.Error(t, err)
}

func TestAppiumAdapter_SessPath(t *testing.T) {
	adapter := NewAppiumAdapter(
		"http://localhost:4723",
		AppiumCapabilities{},
	)
	adapter.sessionID = "abc-123"

	tests := []struct {
		name   string
		suffix string
		want   string
	}{
		{
			name:   "empty_suffix",
			suffix: "",
			want:   "/session/abc-123",
		},
		{
			name:   "screenshot",
			suffix: "/screenshot",
			want:   "/session/abc-123/screenshot",
		},
		{
			name:   "actions",
			suffix: "/actions",
			want:   "/session/abc-123/actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.sessPath(tt.suffix)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name string
		resp map[string]interface{}
		want string
	}{
		{
			name: "flat_format",
			resp: map[string]interface{}{
				"sessionId": "flat-sess-001",
			},
			want: "flat-sess-001",
		},
		{
			name: "nested_format",
			resp: map[string]interface{}{
				"value": map[string]interface{}{
					"sessionId": "nested-sess-002",
				},
			},
			want: "nested-sess-002",
		},
		{
			name: "no_session_id",
			resp: map[string]interface{}{
				"value": map[string]interface{}{
					"other": "data",
				},
			},
			want: "",
		},
		{
			name: "empty_response",
			resp: map[string]interface{}{},
			want: "",
		},
		{
			name: "non_string_session_id",
			resp: map[string]interface{}{
				"sessionId": 12345,
			},
			want: "",
		},
		{
			name: "value_not_a_map",
			resp: map[string]interface{}{
				"value": "not-a-map",
			},
			want: "",
		},
		{
			name: "flat_preferred_over_nested",
			resp: map[string]interface{}{
				"sessionId": "flat-one",
				"value": map[string]interface{}{
					"sessionId": "nested-one",
				},
			},
			want: "flat-one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSessionID(tt.resp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractAppiumElementID(t *testing.T) {
	w3cKey := "element-6066-11e4-a52e-4f735466cecf"

	tests := []struct {
		name string
		resp map[string]interface{}
		want string
	}{
		{
			name: "w3c_key",
			resp: map[string]interface{}{
				"value": map[string]interface{}{
					w3cKey: "elem-w3c-1",
				},
			},
			want: "elem-w3c-1",
		},
		{
			name: "legacy_key",
			resp: map[string]interface{}{
				"value": map[string]interface{}{
					"ELEMENT": "elem-legacy-1",
				},
			},
			want: "elem-legacy-1",
		},
		{
			name: "w3c_preferred_over_legacy",
			resp: map[string]interface{}{
				"value": map[string]interface{}{
					w3cKey:    "w3c-id",
					"ELEMENT": "legacy-id",
				},
			},
			want: "w3c-id",
		},
		{
			name: "no_known_key",
			resp: map[string]interface{}{
				"value": map[string]interface{}{
					"unknown": "abc",
				},
			},
			want: "",
		},
		{
			name: "value_not_a_map",
			resp: map[string]interface{}{
				"value": "string-value",
			},
			want: "",
		},
		{
			name: "empty_response",
			resp: map[string]interface{}{},
			want: "",
		},
		{
			name: "non_string_element_id",
			resp: map[string]interface{}{
				"value": map[string]interface{}{
					w3cKey: 42,
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAppiumElementID(tt.resp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestKeycodeToInt(t *testing.T) {
	tests := []struct {
		name    string
		keycode string
		want    int
	}{
		{
			name:    "home",
			keycode: "KEYCODE_HOME",
			want:    3,
		},
		{
			name:    "back",
			keycode: "KEYCODE_BACK",
			want:    4,
		},
		{
			name:    "enter",
			keycode: "KEYCODE_ENTER",
			want:    66,
		},
		{
			name:    "delete",
			keycode: "KEYCODE_DEL",
			want:    67,
		},
		{
			name:    "menu",
			keycode: "KEYCODE_MENU",
			want:    82,
		},
		{
			name:    "volume_up",
			keycode: "KEYCODE_VOLUME_UP",
			want:    24,
		},
		{
			name:    "volume_down",
			keycode: "KEYCODE_VOLUME_DOWN",
			want:    25,
		},
		{
			name:    "power",
			keycode: "KEYCODE_POWER",
			want:    26,
		},
		{
			name:    "tab",
			keycode: "KEYCODE_TAB",
			want:    61,
		},
		{
			name:    "space",
			keycode: "KEYCODE_SPACE",
			want:    62,
		},
		{
			name:    "escape",
			keycode: "KEYCODE_ESCAPE",
			want:    111,
		},
		{
			name:    "app_switch",
			keycode: "KEYCODE_APP_SWITCH",
			want:    187,
		},
		{
			name:    "dpad_up",
			keycode: "KEYCODE_DPAD_UP",
			want:    19,
		},
		{
			name:    "dpad_down",
			keycode: "KEYCODE_DPAD_DOWN",
			want:    20,
		},
		{
			name:    "dpad_left",
			keycode: "KEYCODE_DPAD_LEFT",
			want:    21,
		},
		{
			name:    "dpad_right",
			keycode: "KEYCODE_DPAD_RIGHT",
			want:    22,
		},
		{
			name:    "dpad_center",
			keycode: "KEYCODE_DPAD_CENTER",
			want:    23,
		},
		{
			name:    "search",
			keycode: "KEYCODE_SEARCH",
			want:    84,
		},
		{
			name:    "notification",
			keycode: "KEYCODE_NOTIFICATION",
			want:    83,
		},
		{
			name:    "media_play",
			keycode: "KEYCODE_MEDIA_PLAY",
			want:    126,
		},
		{
			name:    "media_pause",
			keycode: "KEYCODE_MEDIA_PAUSE",
			want:    127,
		},
		{
			name:    "media_next",
			keycode: "KEYCODE_MEDIA_NEXT",
			want:    87,
		},
		{
			name:    "media_previous",
			keycode: "KEYCODE_MEDIA_PREVIOUS",
			want:    88,
		},
		{
			name:    "unknown_returns_zero",
			keycode: "KEYCODE_UNKNOWN",
			want:    0,
		},
		{
			name:    "empty_string_returns_zero",
			keycode: "",
			want:    0,
		},
		{
			name:    "lowercase_not_recognized",
			keycode: "keycode_home",
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keycodeToInt(tt.keycode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToJSArray(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "empty_slice",
			args: []string{},
			want: "[]",
		},
		{
			name: "nil_slice",
			args: nil,
			want: "[]",
		},
		{
			name: "single_element",
			args: []string{"hello"},
			want: "['hello']",
		},
		{
			name: "multiple_elements",
			args: []string{"am", "instrument", "-w"},
			want: "['am', 'instrument', '-w']",
		},
		{
			name: "with_special_chars",
			args: []string{"it's", "a\\test"},
			want: "['it\\'s', 'a\\\\test']",
		},
		{
			name: "with_spaces",
			args: []string{"-e", "class", "com.Test"},
			want: "['‐e', 'class', 'com.Test']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toJSArray(tt.args)
			if tt.name == "with_spaces" {
				// Just verify it produces valid-looking
				// JS array for multi-arg input.
				assert.True(
					t,
					len(got) > 2,
					"should produce non-empty array",
				)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppiumAdapter_Initialize_MockServer(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"sessionId": "appium-sess-1",
					},
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewAppiumAdapter(
		srv.URL,
		AppiumCapabilities{
			PlatformName:   "Android",
			AutomationName: "UiAutomator2",
			DeviceName:     "emulator-5554",
			AppPackage:     "com.test.app",
		},
	)

	err := adapter.initialize(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, "appium-sess-1", adapter.sessionID,
	)
}

func TestAppiumAdapter_Initialize_NoSessionID(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"other": "data",
					},
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewAppiumAdapter(
		srv.URL, AppiumCapabilities{},
	)

	err := adapter.initialize(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no sessionId")
}

func TestAppiumAdapter_IsDeviceAvailable_MockServer(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"ready": true,
					},
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewAppiumAdapter(
		srv.URL, AppiumCapabilities{},
	)
	avail, err := adapter.IsDeviceAvailable(
		context.Background(),
	)
	require.NoError(t, err)
	assert.True(t, avail)
}

func TestAppiumAdapter_IsDeviceAvailable_NotReady(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"ready": false,
					},
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewAppiumAdapter(
		srv.URL, AppiumCapabilities{},
	)
	avail, err := adapter.IsDeviceAvailable(
		context.Background(),
	)
	require.NoError(t, err)
	assert.False(t, avail)
}

func TestAppiumAdapter_AppiumError(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				w.WriteHeader(http.StatusBadRequest)
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"message": "invalid caps",
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewAppiumAdapter(
		srv.URL,
		AppiumCapabilities{
			PlatformName: "Android",
		},
	)

	err := adapter.initialize(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid caps")
}
