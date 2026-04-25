// Package userflow provides a generic user flow testing framework
// for multi-platform testing across API, browser, mobile, desktop,
// and build pipelines.
package userflow

import (
	"encoding/xml"
	"fmt"
	"time"
)

// TestResult aggregates results from one or more test suites.
type TestResult struct {
	Suites       []TestSuite   `json:"suites"`
	TotalTests   int           `json:"total_tests"`
	TotalFailed  int           `json:"total_failed"`
	TotalErrors  int           `json:"total_errors"`
	TotalSkipped int           `json:"total_skipped"`
	Duration     time.Duration `json:"duration"`
	Output       string        `json:"output"`
}

// TestSuite represents a named group of test cases.
type TestSuite struct {
	Name      string        `json:"name"`
	Tests     int           `json:"tests"`
	Failures  int           `json:"failures"`
	Errors    int           `json:"errors"`
	Skipped   int           `json:"skipped"`
	Duration  time.Duration `json:"duration"`
	TestCases []TestCase    `json:"test_cases"`
}

// TestCase represents a single test within a suite.
type TestCase struct {
	Name      string       `json:"name"`
	ClassName string       `json:"class_name"`
	Duration  string       `json:"duration"`
	Status    string       `json:"status"`
	Failure   *TestFailure `json:"failure,omitempty"`
}

// TestFailure captures details about a test failure.
type TestFailure struct {
	Message    string `json:"message"`
	Type       string `json:"type"`
	StackTrace string `json:"stack_trace"`
}

// BuildResult captures the outcome of a build operation.
type BuildResult struct {
	Target    string        `json:"target"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	Output    string        `json:"output"`
	Artifacts []string      `json:"artifacts"`
}

// LintResult captures the outcome of a lint operation.
type LintResult struct {
	Tool     string        `json:"tool"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Warnings int           `json:"warnings"`
	Errors   int           `json:"errors"`
	Output   string        `json:"output"`
}

// BuildTarget specifies a build to execute.
type BuildTarget struct {
	Name string   `json:"name"`
	Task string   `json:"task"`
	Args []string `json:"args"`
}

// TestTarget specifies a test suite to execute.
type TestTarget struct {
	Name   string `json:"name"`
	Task   string `json:"task"`
	Filter string `json:"filter"`
}

// LintTarget specifies a linting tool to execute.
type LintTarget struct {
	Name string   `json:"name"`
	Task string   `json:"task"`
	Args []string `json:"args"`
}

// Credentials holds authentication information for API access.
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
}

// BrowserConfig holds configuration for browser-based testing.
type BrowserConfig struct {
	BrowserType string   `json:"browser_type"`
	Headless    bool     `json:"headless"`
	WindowSize  [2]int   `json:"window_size"`
	ExtraArgs   []string `json:"extra_args"`
}

// DesktopAppConfig holds configuration for launching desktop
// applications.
type DesktopAppConfig struct {
	BinaryPath string            `json:"binary_path"`
	Args       []string          `json:"args"`
	WorkDir    string            `json:"work_dir"`
	Env        map[string]string `json:"env"`
}

// ProcessConfig holds configuration for launching a process.
type ProcessConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	WorkDir string            `json:"work_dir"`
	Env     map[string]string `json:"env"`
}

// MobileConfig holds configuration for mobile device testing.
type MobileConfig struct {
	PackageName  string `json:"package_name"`
	ActivityName string `json:"activity_name"`
	DeviceSerial string `json:"device_serial"`
}

// JUnit XML types for parsing test results from various tools.

// JUnitTestSuites wraps multiple JUnit test suites.
type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a single JUnit test suite.
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case in JUnit XML.
type JUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitError   `xml:"error,omitempty"`
}

// JUnitFailure represents a test failure in JUnit XML.
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// JUnitError represents a test error in JUnit XML.
type JUnitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// ParseJUnitXML parses JUnit XML data into test suites. It
// first tries to parse as a <testsuites> wrapper, then falls
// back to parsing as a single <testsuite>.
func ParseJUnitXML(data []byte) ([]JUnitTestSuite, error) {
	// Try parsing as <testsuites> first.
	var suites JUnitTestSuites
	if err := xml.Unmarshal(data, &suites); err == nil {
		if len(suites.Suites) > 0 {
			return suites.Suites, nil
		}
	}

	// Fall back to single <testsuite>.
	var suite JUnitTestSuite
	if err := xml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parse junit xml: %w", err)
	}
	return []JUnitTestSuite{suite}, nil
}

// JUnitToTestResult converts parsed JUnit suites into a
// TestResult with aggregated counts.
func JUnitToTestResult(
	suites []JUnitTestSuite,
	duration time.Duration,
	output string,
) *TestResult {
	result := &TestResult{
		Suites:   make([]TestSuite, 0, len(suites)),
		Duration: duration,
		Output:   output,
	}

	for _, js := range suites {
		suiteDuration := time.Duration(
			js.Time * float64(time.Second),
		)
		ts := TestSuite{
			Name:      js.Name,
			Tests:     js.Tests,
			Failures:  js.Failures,
			Errors:    js.Errors,
			Skipped:   js.Skipped,
			Duration:  suiteDuration,
			TestCases: make([]TestCase, 0, len(js.TestCases)),
		}
		for _, jc := range js.TestCases {
			tc := TestCase{
				Name:      jc.Name,
				ClassName: jc.ClassName,
				Duration: fmt.Sprintf(
					"%.3fs", jc.Time,
				),
				Status: "passed",
			}
			if jc.Failure != nil {
				tc.Status = "failed"
				tc.Failure = &TestFailure{
					Message:    jc.Failure.Message,
					Type:       jc.Failure.Type,
					StackTrace: jc.Failure.Body,
				}
			}
			if jc.Error != nil {
				tc.Status = "error"
				tc.Failure = &TestFailure{
					Message:    jc.Error.Message,
					Type:       jc.Error.Type,
					StackTrace: jc.Error.Body,
				}
			}
			ts.TestCases = append(ts.TestCases, tc)
		}
		result.Suites = append(result.Suites, ts)
		result.TotalTests += js.Tests
		result.TotalFailed += js.Failures
		result.TotalErrors += js.Errors
		result.TotalSkipped += js.Skipped
	}

	return result
}
