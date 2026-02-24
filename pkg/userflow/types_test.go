package userflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildResult_Fields(t *testing.T) {
	tests := []struct {
		name    string
		result  BuildResult
		success bool
		target  string
	}{
		{
			name: "successful build",
			result: BuildResult{
				Target:   "catalog-api",
				Success:  true,
				Duration: 30 * time.Second,
				Output:   "build ok",
				Artifacts: []string{
					"bin/catalog-api",
				},
			},
			success: true,
			target:  "catalog-api",
		},
		{
			name: "failed build",
			result: BuildResult{
				Target:    "catalog-web",
				Success:   false,
				Duration:  5 * time.Second,
				Output:    "compile error",
				Artifacts: nil,
			},
			success: false,
			target:  "catalog-web",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.target, tt.result.Target)
			assert.Equal(t, tt.success, tt.result.Success)
		})
	}
}

func TestTestResult_Fields(t *testing.T) {
	tests := []struct {
		name        string
		result      TestResult
		totalTests  int
		totalFailed int
	}{
		{
			name: "all passing",
			result: TestResult{
				TotalTests:  10,
				TotalFailed: 0,
				TotalErrors: 0,
				Duration:    5 * time.Second,
			},
			totalTests:  10,
			totalFailed: 0,
		},
		{
			name: "some failures",
			result: TestResult{
				TotalTests:  10,
				TotalFailed: 3,
				TotalErrors: 1,
				Duration:    8 * time.Second,
			},
			totalTests:  10,
			totalFailed: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(
				t, tt.totalTests, tt.result.TotalTests,
			)
			assert.Equal(
				t, tt.totalFailed, tt.result.TotalFailed,
			)
		})
	}
}

func TestTestCase_Fields(t *testing.T) {
	tests := []struct {
		name     string
		tc       TestCase
		status   string
		hasError bool
	}{
		{
			name: "passing test",
			tc: TestCase{
				Name:      "TestLogin",
				ClassName: "auth_test",
				Duration:  "0.500s",
				Status:    "passed",
				Failure:   nil,
			},
			status:   "passed",
			hasError: false,
		},
		{
			name: "failing test",
			tc: TestCase{
				Name:      "TestLogin",
				ClassName: "auth_test",
				Duration:  "0.200s",
				Status:    "failed",
				Failure: &TestFailure{
					Message:    "expected 200 got 401",
					Type:       "AssertionError",
					StackTrace: "at line 42",
				},
			},
			status:   "failed",
			hasError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.status, tt.tc.Status)
			if tt.hasError {
				require.NotNil(t, tt.tc.Failure)
				assert.NotEmpty(t, tt.tc.Failure.Message)
			} else {
				assert.Nil(t, tt.tc.Failure)
			}
		})
	}
}

func TestBuildTarget_Fields(t *testing.T) {
	tests := []struct {
		name   string
		target BuildTarget
	}{
		{
			name: "go build",
			target: BuildTarget{
				Name: "catalog-api",
				Task: "go build",
				Args: []string{"-o", "bin/catalog-api"},
			},
		},
		{
			name: "npm build",
			target: BuildTarget{
				Name: "catalog-web",
				Task: "npm run build",
				Args: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.target.Name)
			assert.NotEmpty(t, tt.target.Task)
		})
	}
}

func TestLintResult_Fields(t *testing.T) {
	tests := []struct {
		name    string
		result  LintResult
		success bool
	}{
		{
			name: "clean lint",
			result: LintResult{
				Tool:     "golangci-lint",
				Success:  true,
				Duration: 10 * time.Second,
				Warnings: 0,
				Errors:   0,
				Output:   "",
			},
			success: true,
		},
		{
			name: "lint with warnings",
			result: LintResult{
				Tool:     "eslint",
				Success:  true,
				Duration: 3 * time.Second,
				Warnings: 5,
				Errors:   0,
				Output:   "5 warnings",
			},
			success: true,
		},
		{
			name: "lint with errors",
			result: LintResult{
				Tool:     "eslint",
				Success:  false,
				Duration: 3 * time.Second,
				Warnings: 2,
				Errors:   3,
				Output:   "3 errors, 2 warnings",
			},
			success: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.success, tt.result.Success)
			assert.NotEmpty(t, tt.result.Tool)
		})
	}
}

func TestParseJUnitXML_MultiSuite(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="suite1" tests="2" failures="1" errors="0" skipped="0" time="1.5">
    <testcase name="TestA" classname="pkg.Foo" time="0.5"/>
    <testcase name="TestB" classname="pkg.Foo" time="1.0">
      <failure message="expected true" type="AssertionError">stack trace here</failure>
    </testcase>
  </testsuite>
  <testsuite name="suite2" tests="1" failures="0" errors="0" skipped="1" time="0.1">
    <testcase name="TestC" classname="pkg.Bar" time="0.1"/>
  </testsuite>
</testsuites>`)

	suites, err := ParseJUnitXML(xmlData)
	require.NoError(t, err)
	require.Len(t, suites, 2)

	assert.Equal(t, "suite1", suites[0].Name)
	assert.Equal(t, 2, suites[0].Tests)
	assert.Equal(t, 1, suites[0].Failures)
	require.Len(t, suites[0].TestCases, 2)
	assert.Equal(t, "TestA", suites[0].TestCases[0].Name)
	assert.Nil(t, suites[0].TestCases[0].Failure)
	require.NotNil(t, suites[0].TestCases[1].Failure)
	assert.Equal(
		t, "expected true",
		suites[0].TestCases[1].Failure.Message,
	)

	assert.Equal(t, "suite2", suites[1].Name)
	assert.Equal(t, 1, suites[1].Tests)
}

func TestParseJUnitXML_SingleSuite(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="single" tests="3" failures="0" errors="1" skipped="0" time="2.0">
  <testcase name="TestX" classname="pkg.X" time="0.5"/>
  <testcase name="TestY" classname="pkg.X" time="0.8"/>
  <testcase name="TestZ" classname="pkg.X" time="0.7">
    <error message="nil pointer" type="RuntimeError">panic at line 10</error>
  </testcase>
</testsuite>`)

	suites, err := ParseJUnitXML(xmlData)
	require.NoError(t, err)
	require.Len(t, suites, 1)

	assert.Equal(t, "single", suites[0].Name)
	assert.Equal(t, 3, suites[0].Tests)
	assert.Equal(t, 1, suites[0].Errors)
	require.Len(t, suites[0].TestCases, 3)
	require.NotNil(t, suites[0].TestCases[2].Error)
	assert.Equal(
		t, "nil pointer",
		suites[0].TestCases[2].Error.Message,
	)
}

func TestParseJUnitXML_InvalidXML(t *testing.T) {
	_, err := ParseJUnitXML([]byte("not xml"))
	assert.Error(t, err)
}

func TestJUnitToTestResult(t *testing.T) {
	suites := []JUnitTestSuite{
		{
			Name:     "suite1",
			Tests:    3,
			Failures: 1,
			Errors:   0,
			Skipped:  0,
			Time:     2.5,
			TestCases: []JUnitTestCase{
				{
					Name:      "TestA",
					ClassName: "pkg.A",
					Time:      0.5,
				},
				{
					Name:      "TestB",
					ClassName: "pkg.A",
					Time:      1.0,
					Failure: &JUnitFailure{
						Message: "fail",
						Type:    "AssertionError",
						Body:    "trace",
					},
				},
				{
					Name:      "TestC",
					ClassName: "pkg.A",
					Time:      1.0,
				},
			},
		},
		{
			Name:     "suite2",
			Tests:    2,
			Failures: 0,
			Errors:   1,
			Skipped:  1,
			Time:     1.0,
			TestCases: []JUnitTestCase{
				{
					Name:      "TestD",
					ClassName: "pkg.B",
					Time:      0.5,
					Error: &JUnitError{
						Message: "err",
						Type:    "RuntimeError",
						Body:    "panic",
					},
				},
				{
					Name:      "TestE",
					ClassName: "pkg.B",
					Time:      0.5,
				},
			},
		},
	}

	result := JUnitToTestResult(
		suites, 4*time.Second, "output",
	)

	assert.Equal(t, 5, result.TotalTests)
	assert.Equal(t, 1, result.TotalFailed)
	assert.Equal(t, 1, result.TotalErrors)
	assert.Equal(t, 1, result.TotalSkipped)
	assert.Equal(t, 4*time.Second, result.Duration)
	assert.Equal(t, "output", result.Output)
	require.Len(t, result.Suites, 2)

	// Check status mapping
	assert.Equal(t, "passed", result.Suites[0].TestCases[0].Status)
	assert.Equal(t, "failed", result.Suites[0].TestCases[1].Status)
	assert.Equal(t, "error", result.Suites[1].TestCases[0].Status)
}
