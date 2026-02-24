package userflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIFlow_Fields(t *testing.T) {
	flow := APIFlow{
		Name:        "login-flow",
		Description: "Test login and token refresh",
		BaseURL:     "http://localhost:8080",
		Credentials: Credentials{
			Username: "admin",
			Password: "pass",
			URL:      "http://localhost:8080",
		},
		Steps: []APIStep{
			{
				Name:   "login",
				Method: "POST",
				Path:   "/api/v1/auth/login",
				Body:   `{"username":"admin","password":"pass"}`,
				Assertions: []StepAssertion{
					{
						Type:    "status_code",
						Target:  "status",
						Value:   200,
						Message: "login should return 200",
					},
				},
			},
			{
				Name:   "get profile",
				Method: "GET",
				Path:   "/api/v1/users/me",
			},
		},
	}

	assert.Equal(t, "login-flow", flow.Name)
	assert.Equal(
		t, "http://localhost:8080", flow.BaseURL,
	)
	assert.Equal(t, "admin", flow.Credentials.Username)
	require.Len(t, flow.Steps, 2)
	assert.Equal(t, "login", flow.Steps[0].Name)
	assert.Equal(t, "POST", flow.Steps[0].Method)
	require.Len(t, flow.Steps[0].Assertions, 1)
	assert.Equal(
		t, "status_code",
		flow.Steps[0].Assertions[0].Type,
	)
	assert.Equal(
		t, "get profile", flow.Steps[1].Name,
	)
}

func TestAPIStep_Headers(t *testing.T) {
	step := APIStep{
		Name:   "custom-header",
		Method: "GET",
		Path:   "/api/v1/test",
		Headers: map[string]string{
			"X-Custom": "value",
		},
	}
	assert.Equal(t, "value", step.Headers["X-Custom"])
}

func TestBrowserFlow_Fields(t *testing.T) {
	flow := BrowserFlow{
		Name:        "signup-flow",
		Description: "Test user signup UI",
		StartURL:    "http://localhost:3000/signup",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
			WindowSize:  [2]int{1920, 1080},
		},
		Steps: []BrowserStep{
			{
				Name:     "fill email",
				Action:   "fill",
				Selector: "#email",
				Value:    "test@example.com",
			},
			{
				Name:     "click submit",
				Action:   "click",
				Selector: "#submit-btn",
				Assertions: []StepAssertion{
					{
						Type:    "flow_completes",
						Target:  "signup",
						Message: "signup should complete",
					},
				},
			},
			{
				Name:   "take screenshot",
				Action: "screenshot",
			},
		},
	}

	assert.Equal(t, "signup-flow", flow.Name)
	assert.Equal(
		t, "chromium", flow.Config.BrowserType,
	)
	assert.True(t, flow.Config.Headless)
	assert.Equal(t, 1920, flow.Config.WindowSize[0])
	require.Len(t, flow.Steps, 3)
	assert.Equal(t, "fill", flow.Steps[0].Action)
	assert.Equal(t, "#email", flow.Steps[0].Selector)
	assert.Equal(
		t, "test@example.com", flow.Steps[0].Value,
	)
	assert.Equal(t, "click", flow.Steps[1].Action)
	require.Len(t, flow.Steps[1].Assertions, 1)
	assert.Equal(
		t, "screenshot", flow.Steps[2].Action,
	)
}

func TestBrowserStep_EvaluateJS(t *testing.T) {
	step := BrowserStep{
		Name:   "check title",
		Action: "evaluate_js",
		Script: "document.title",
	}
	assert.Equal(t, "evaluate_js", step.Action)
	assert.NotEmpty(t, step.Script)
}

func TestMobileFlow_Fields(t *testing.T) {
	flow := MobileFlow{
		Name:        "android-login",
		Description: "Test Android login flow",
		Config: MobileConfig{
			PackageName:  "com.example.app",
			ActivityName: ".MainActivity",
			DeviceSerial: "emulator-5554",
		},
		AppPath: "/tmp/app.apk",
		Steps: []MobileStep{
			{
				Name:   "launch app",
				Action: "launch",
			},
			{
				Name:   "tap login button",
				Action: "tap",
				X:      540,
				Y:      960,
			},
			{
				Name:   "enter username",
				Action: "send_keys",
				Value:  "admin",
			},
			{
				Name:   "press back",
				Action: "press_key",
				Value:  "KEYCODE_BACK",
			},
			{
				Name:   "capture screen",
				Action: "screenshot",
				Assertions: []StepAssertion{
					{
						Type:    "screenshot_exists",
						Target:  "screen",
						Message: "screenshot should exist",
					},
				},
			},
		},
	}

	assert.Equal(t, "android-login", flow.Name)
	assert.Equal(
		t, "com.example.app", flow.Config.PackageName,
	)
	assert.Equal(
		t, "emulator-5554", flow.Config.DeviceSerial,
	)
	assert.Equal(t, "/tmp/app.apk", flow.AppPath)
	require.Len(t, flow.Steps, 5)
	assert.Equal(t, "launch", flow.Steps[0].Action)
	assert.Equal(t, "tap", flow.Steps[1].Action)
	assert.Equal(t, 540, flow.Steps[1].X)
	assert.Equal(t, 960, flow.Steps[1].Y)
	assert.Equal(t, "send_keys", flow.Steps[2].Action)
	assert.Equal(t, "admin", flow.Steps[2].Value)
	assert.Equal(t, "press_key", flow.Steps[3].Action)
	assert.Equal(
		t, "KEYCODE_BACK", flow.Steps[3].Value,
	)
	require.Len(t, flow.Steps[4].Assertions, 1)
}

func TestIPCCommand_Fields(t *testing.T) {
	cmd := IPCCommand{
		Name:           "get-config",
		Command:        "get_app_config",
		Args:           []string{"--format", "json"},
		ExpectedResult: `{"theme":"dark"}`,
		Assertions: []StepAssertion{
			{
				Type:    "response_contains",
				Target:  "response",
				Value:   "dark",
				Message: "should contain dark theme",
			},
		},
	}

	assert.Equal(t, "get-config", cmd.Name)
	assert.Equal(t, "get_app_config", cmd.Command)
	require.Len(t, cmd.Args, 2)
	assert.Equal(t, "--format", cmd.Args[0])
	assert.Equal(
		t, `{"theme":"dark"}`, cmd.ExpectedResult,
	)
	require.Len(t, cmd.Assertions, 1)
	assert.Equal(
		t, "response_contains",
		cmd.Assertions[0].Type,
	)
}

func TestStepAssertion_Fields(t *testing.T) {
	tests := []struct {
		name      string
		assertion StepAssertion
		aType     string
	}{
		{
			name: "status code",
			assertion: StepAssertion{
				Type:    "status_code",
				Target:  "status",
				Value:   200,
				Message: "should be 200",
			},
			aType: "status_code",
		},
		{
			name: "response contains",
			assertion: StepAssertion{
				Type:    "response_contains",
				Target:  "body",
				Value:   "ok",
				Message: "should contain ok",
			},
			aType: "response_contains",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.aType, tt.assertion.Type)
			assert.NotEmpty(t, tt.assertion.Target)
			assert.NotEmpty(t, tt.assertion.Message)
		})
	}
}
