package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultHost = "integration-test.herokai.com"
)

// Command line options
var direwolfHost = flag.String("direwolfHost", defaultHost, "direwolf host")
var apiKey = flag.String("apiKey", "", "api key to use (or DW_API_KEY)")
var listClouds = flag.Bool("listClouds", false, "list clouds and exit")
var domain = flag.String("domain", "", "cloud domain")
var region = flag.String("region", "", "cloud region")
var suite = flag.String("suite", "", "suite to run")

// Cloud object returned by /clouds api
type Cloud struct {
	Id     string `json:"id"`
	Domain string `json:"domain"`
	Label  string `json:"label"`
	Region string `json:"region"`
	State  string `json:"state"`
}

// die prints error message and aborts the program
func die(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	os.Exit(1)
}

// apiCall calls the HTTP api (setting auth), returns the response
func apiCall(method, path string, payload []byte) (*http.Response, error) {
	// FIXME: https won't work with localhost
	url := fmt.Sprintf("https://%s/api/%s", *direwolfHost, path)
	var rdr io.Reader

	if payload == nil {
		rdr = nil
	} else {
		rdr = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(*apiKey, "")
	req.Header.Add("Content-Type", "application/json")

	return http.DefaultClient.Do(req)
}

// getClouds get list of clouds
func getClouds() ([]Cloud, error) {
	resp, err := apiCall("GET", "clouds", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// FIXME: Move to apiCall
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad return code: %d", resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	var reply []Cloud
	if err = dec.Decode(&reply); err != nil {
		return nil, err
	}

	return reply, nil
}

// findCloudId finds the matching cloud id for domain and region
func findCloudId(domain, region string, clouds []Cloud) string {
	for _, cloud := range clouds {
		if (cloud.Domain == domain) && (cloud.Region == region) {
			return cloud.Id
		}
	}

	return ""
}

// Runs JSON payload
// FIXME: There's probably a better way to do this
type RunsCloud struct {
	Id string `json:"id"`
}
type RunsSuite struct {
	Label string `json:"label"`
}

type RunsPayload struct {
	RunsCloud `json:"cloud"`
	RunsSuite `json:"suite"`
}

// encodeRunsPayload creates JSON object for POST /runs
func encodeRunsPayload(cloud, suite string) ([]byte, error) {
	payload := RunsPayload{
		RunsCloud{cloud},
		RunsSuite{suite},
	}
	return json.Marshal(payload)
}

// StatusSummary is the "summary" inner struct in Status
type StatusSummary struct {
	Passed  int
	Failed  int
	Skipped int
	Running int
	Pending int
}

// Status is the reply from POST to /runs or GET /runs/<id>
type Status struct {
	Id      string        `json:"id"`
	State   string        `json:"state"`
	Summary StatusSummary `json:"summary"`
	Start   *time.Time    `json:"started_at"`
	End     *time.Time    `json:"ended_at"`
}

// String is the string formatting for Status
func (status *Status) String() string {
	return fmt.Sprintf("state: %s, summary: %+v\r", status.State, status.Summary)
}

// decodeStatus decodes status response
func decodeStatus(resp *http.Response) (*Status, error) {
	dec := json.NewDecoder(resp.Body)
	var reply Status
	if err := dec.Decode(&reply); err != nil {
		return nil, fmt.Errorf("can't decode runs reply - %s", err)
	}
	return &reply, nil
}

// startRun dispatches a run of <suite> on <cloud>, returns run id
func startRun(cloud, suite string) (*Status, error) {
	payload, err := encodeRunsPayload(cloud, suite)
	if err != nil {
		return nil, fmt.Errorf("can't encode runs payload - %s", err)
	}
	resp, err := apiCall("POST", "/runs", payload)
	if err != nil {
		return nil, fmt.Errorf("can't dispatch new run - %s", err)
	}
	defer resp.Body.Close()
	return decodeStatus(resp)
}

// runStatus get the run status of run <id>
func runStatus(id string) (*Status, error) {
	url := fmt.Sprintf("/runs/%s", id)
	resp, err := apiCall("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("can't get %s status - %s", id, err)
	}
	defer resp.Body.Close()
	return decodeStatus(resp)
}

func waitForRun(status *Status, print bool) (*Status, error) {
	for {
		status, err := runStatus(status.Id)
		if err != nil {
			return nil, err
		}

		if print {
			fmt.Printf("%s\r", status)
		}

		if status.End != nil {
			return status, nil
		}

		time.Sleep(time.Second)
	}
}

func main() {
	flag.Parse()

	if len(*apiKey) == 0 {
		*apiKey = os.Getenv("DW_API_KEY")
	}
	if len(*apiKey) == 0 {
		die("no api key")
	}

	clouds, err := getClouds()
	if err != nil {
		die("cannot get list of clouds %s", err)
	}

	if *listClouds {
		for _, cloud := range clouds {
			fmt.Printf("%s (%s):\t\t%s\n", cloud.Domain, cloud.Region, cloud.Id)
		}
		os.Exit(0)
	}

	if (len(*domain) == 0) || (len(*region) == 0) || (len(*suite) == 0) {
		die("missing domain or region")
	}

	cloudId := findCloudId(*domain, *region, clouds)
	if len(cloudId) == 0 {
		die("unknown cloud %s (%s)", *domain, *region)
	}

	status, err := startRun(cloudId, *suite)
	if err != nil {
		die("can't run - %s", err)
	}
	fmt.Printf("run id: %s\n", status.Id)

	status, err = waitForRun(status, true)
	if err != nil {
		die("error waiting for %s - %s", status.Id, err)
	}

	fmt.Printf("%s\n", status) // Print final status
	duration := status.End.Sub(*status.Start).Seconds()
	fmt.Printf("run %s ended at %s (started at %s - took %.1fsec)\n",
		status.Id, *status.End, *status.Start, duration)
	if status.Summary.Failed > 0 {
		os.Exit(1)
	}
}
