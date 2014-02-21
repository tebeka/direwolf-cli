package main

import (
	"os/exec"
	"strings"
	"testing"
)

var testCloud = "brainard.herokudev.com"
var testCloudKey = "ce1294f7-f500-4df7-9490-9c2f68d6ddc6"
var testApp = "./dw"
var testHost = "direwolf-brainard.herokuapp.com"
var okSuite = "examples"
var errSuite = "examples-failing"

func init() {
	if len(*apiKey) == 0 {
		panic("DW_API_KEY not set")
	}

	cmd := exec.Command("go", "build")
	if cmd.Run() != nil {
		panic("can't build")
	}

	*direwolfHost = testHost
}

func createCmd(suite, region string, extra ...string) *exec.Cmd {
	args := []string{
		"-direwolfHost", testHost,
		"-apiKey", *apiKey,
		"-domain", testCloud,
		"-region", region,
		"-suite", suite,
	}
	args = append(args, extra...)
	return exec.Command(testApp, args...)
}

func TestOK(t *testing.T) {
	cmd := createCmd(okSuite, "us")
	if cmd.Run() != nil {
		t.Fatalf("error running 'examples'")
	}
}

func TestFailing(t *testing.T) {
	cmd := createCmd(errSuite, "us")
	if cmd.Run() == nil {
		t.Fatalf("managed to run 'examples-failing' without failure")
	}
}

func TestList(t *testing.T) {
	cmd := createCmd(okSuite, "us", "-listClouds")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("error running 'examples'")
	}

	if !strings.Contains(string(out), testCloud) {
		t.Fatalf("%s not found in -listClouds", testCloud)
	}
}

func TestHelp(t *testing.T) {
	cmd := exec.Command(testApp, "-h")
	// -h return 2 to the OS
	out, _ := cmd.CombinedOutput()

	keys := []string{
		"-apiKey",
		"-direwolfHost",
		"-domain",
		"-listClouds",
		"-region",
		"-suite",
	}

	for _, key := range keys {
		if !strings.Contains(string(out), key) {
			t.Fatalf("%s not found in help output", key)
		}
	}
}

func Test_getClouds(t *testing.T) {
	clouds, err := getClouds()
	if err != nil {
		t.Fatalf("can't get clouds - %s", err)
	}

	if len(clouds) == 0 {
		t.Fatalf("no clouds found")
	}

	found := false
	for _, cloud := range clouds {
		// FIXME: Use findCloudId
		if (cloud.Domain == testCloud) && (cloud.Region == "us") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("can't find %s cloud", testCloud)
	}
}

// FIXME: Table driven
func test_findCloudId(t *testing.T) {
	clouds, err := getClouds()
	if err != nil {
		t.Fatalf("can't get clouds - %s", err)
	}

	id := findCloudId(testCloud, "us", clouds)
	if len(id) == 0 {
		t.Fatalf("can't find cloud %s- %s", testCloud, err)
	}

	id = findCloudId(testCloud, "ussr", clouds)
	if len(id) != 0 {
		t.Fatalf("found cloud in bad region")
	}

	id = findCloudId(testCloud+"no-there", "us", clouds)
	if len(id) != 0 {
		t.Fatalf("found non existing cloud")
	}
}

func run(suite string, t *testing.T) (*Status, error) {
	status, err := startRun(testCloudKey, suite)
	if err != nil {
		return nil, err
	}

	return waitForRun(status, false)
}

func checkSummary(suite string, expected StatusSummary, t *testing.T) {
	status, err := run(suite, t)
	if err != nil {
		t.Fatalf("can't run - %s", err)
	}

	if status.Summary != expected {
		t.Fatalf("bad summary: %+v != %+v", status.Summary, expected)
	}
}

func TestSummary(t *testing.T) {
	var cases = []struct {
		suite    string
		expected StatusSummary
	}{
		{okSuite, StatusSummary{Passed: 5, Failed: 0, Skipped: 1, Running: 0, Pending: 0}},
		{errSuite, StatusSummary{Passed: 2, Failed: 1, Skipped: 1, Running: 0, Pending: 0}},
	}

	for _, data := range cases {
		t.Logf("Testing summary of %s", data.suite)
		checkSummary(data.suite, data.expected, t)
	}
}
