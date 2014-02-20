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
	//	baseURL = "https://integration-test.herokai.com/api"
	baseURL = "https://direwolf-brainard.herokuapp.com/api"
)

// Command line options
var apiKey = flag.String("apiKey", "", "api key to use")
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
	url := fmt.Sprintf("%s/%s", baseURL, path)
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

// findClouldId finds the matching cloud id for domain and region
func findClouldId(domain, region string, clouds []Cloud) string {
	for _, cloud := range clouds {
		if (cloud.Domain == domain) && (cloud.Region == region) {
			return cloud.Id
		}
	}

	return ""
}

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

func decodeStatus(resp *http.Response) (*Status, error) {
	dec := json.NewDecoder(resp.Body)
	var reply Status
	if err := dec.Decode(&reply); err != nil {
		return nil, fmt.Errorf("can't decode runs reply - %s", err)
	}

	return &reply, nil
}

// executeRun dispatches a run of <suite> on <cloud>, returns run id
func executeRun(cloud, suite string) (*Status, error) {
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

func main() {
	flag.Parse()

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

	cloudId := findClouldId(*domain, *region, clouds)
	if len(cloudId) == 0 {
		die("unknown cloud %s (%s)", *domain, *region)
	}

	status, err := executeRun(cloudId, *suite)
	if err != nil {
		die("can't run - %s", err)
	}
	fmt.Printf("run id: %s\n", status.Id)

	for {
		status, err = runStatus(status.Id)
		fmt.Printf("state: %s, summary: %+v\r", status.State, status.Summary)
		if status.End != nil {
			break
		}
		time.Sleep(time.Second)
	}

	duration := status.End.Sub(*status.Start).Seconds()
	fmt.Printf("run %s ended at %s (started at %s - took %.1fsec)\n", status.Id, *status.End, *status.Start, duration)
	if status.Summary.Failed > 0 {
		os.Exit(1)
	}
}
