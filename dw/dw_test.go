package main

import (
	"os"
	"os/exec"
	"testing"
)

var testKey = ""

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
		"-domain", "brainard.herokudev.com",
		"-region", region,
		"-suite", suite,
	}
	args = append(args, extra...)
	return exec.Command("./dw", args...)
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
