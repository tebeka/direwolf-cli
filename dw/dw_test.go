package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

var testKey = ""
var testCloud = "brainard.herokudev.com"
var testApp = "./dw"

func init() {
	testKey = os.Getenv("DW_API_KEY")
	if len(testKey) == 0 {
		panic("DW_API_KEY not set")
	}

	cmd := exec.Command("go", "build")
	if cmd.Run() != nil {
		panic("can't build")
	}
}

func createCmd(suite, region string, extra ...string) *exec.Cmd {
	args := []string{
		"-direwolfHost", "direwolf-brainard.herokuapp.com",
		"-apiKey", testKey,
		"-domain", testCloud,
		"-region", region,
		"-suite", suite,
	}
	args = append(args, extra...)
	return exec.Command(testApp, args...)
}

func TestOK(t *testing.T) {
	cmd := createCmd("examples", "us")
	if cmd.Run() != nil {
		t.Fatalf("error running 'examples'")
	}
}

func TestFailing(t *testing.T) {
	cmd := createCmd("examples-failing", "us")
	if cmd.Run() == nil {
		t.Fatalf("managed to run 'examples-failing' without failure")
	}
}

func TestList(t *testing.T) {
	cmd := createCmd("examples", "us", "-listClouds")
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
