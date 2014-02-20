package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"flag"
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

type Cloud struct {
	Id     string `json:"id"`
	Domain string `json:"domain"`
	Label  string `json:"label"`
	Region string `json:"region"`
	State  string `json:"state"`
}

// Die prints error message and aborts the program
func die(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	os.Exit(1)
}

// getClouds get list of clouds
func getClouds() ([]Cloud, error) {
	url := fmt.Sprintf("%s/clouds", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(*apiKey, "")
	resp, err := http.DefaultClient.Do(req)
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

func clouldId(domain, region string, clouds []Cloud) string {
	for _, cloud := range clouds {
		if (cloud.Domain == domain) && (cloud.Region == region) {
			return cloud.Id
		}
	}

	return ""
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

	if (len(*domain) == 0) || (len(*region) == 0) {
		die("missing domain or region")
	}

	id := clouldId(*domain, *region, clouds)
	if len(id) == 0 {
		die("unknown cloud %s (%s)", *domain, *region)
	}
}
