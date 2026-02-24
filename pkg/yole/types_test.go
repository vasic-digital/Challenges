package yole

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename),
		"testdata", name,
	)
}

func TestJUnitXML_Parsing(t *testing.T) {
	data, err := os.ReadFile(
		testdataPath("junit-results.xml"),
	)
	require.NoError(t, err)

	var suites JUnitTestSuites
	err = xml.Unmarshal(data, &suites)
	require.NoError(t, err)

	assert.Len(t, suites.TestSuites, 2)

	md := suites.TestSuites[0]
	assert.Contains(t, md.Name, "MarkdownParserTests")
	assert.Equal(t, 25, md.Tests)
	assert.Equal(t, 0, md.Failures)
	assert.Equal(t, 0, md.Errors)
	assert.Len(t, md.TestCases, 3)
	assert.Nil(t, md.TestCases[0].Failure)

	todo := suites.TestSuites[1]
	assert.Contains(t, todo.Name, "TodoTxtParserTests")
	assert.Equal(t, 18, todo.Tests)
	assert.Equal(t, 1, todo.Failures)
	assert.Len(t, todo.TestCases, 2)
	assert.NotNil(t, todo.TestCases[1].Failure)
	assert.Contains(t,
		todo.TestCases[1].Failure.Message, "@home",
	)
}

func TestBuildTarget(t *testing.T) {
	bt := BuildTarget{
		Name: "Android Debug",
		Task: ":androidApp:assembleDebug",
	}
	assert.Equal(t, "Android Debug", bt.Name)
	assert.Equal(t, ":androidApp:assembleDebug", bt.Task)
}

func TestTestTarget(t *testing.T) {
	tt := TestTarget{
		Name:   "Shared Tests",
		Task:   ":shared:test",
		Filter: "digital.vasic.yole.format.*",
	}
	assert.Equal(t, "Shared Tests", tt.Name)
	assert.Equal(t, ":shared:test", tt.Task)
	assert.Equal(t,
		"digital.vasic.yole.format.*", tt.Filter,
	)
}
