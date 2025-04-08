package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultProjectName = "docker-deploy"
const defaultUpdateFrequencySeconds = 30

var projectName string
var updateFrequency time.Duration
var basicAuth string
var apiEndpoint string
var noCache bool

var lastModified string
var etag string

func main() {
	var f int

	h, _ := os.Hostname()

	flag.StringVar(&projectName, "p", defaultProjectName, "Project name")
	flag.IntVar(&f, "i", defaultUpdateFrequencySeconds, "Update interval in seconds")
	flag.BoolVar(&noCache, "no-cache", false, "Pass this flag to disable last-modified checks")

	flag.Usage = func() {
		w := flag.CommandLine.Output()

		fmt.Fprintf(w, `Usage:
	%s [OPTIONS] <apiEndpoint>

Example Endpoints:
	https://.../api/v1/machines/$(hostname -s)/docker-compose.yml
	https://.../api/v1/machines/:hostname/docker-compose.yml
	https://.../api/v1/fleets/default/docker-comnpose.yml

Placeholders:
:hostname	Replaced with system hostname

Environment:
HTTP_BASIC	Basic Auth in the form of <username>:<password>

Options:
`, os.Args[0])

		flag.PrintDefaults()
	}

	flag.Parse()

	updateFrequency = time.Duration(f) * time.Second

	args := flag.Args()

	if len(args) == 0 {
		fmt.Println(`Error: apiEndpoint not specified.`)
		flag.Usage()
		return
	}

	// Check for auth env var
	basicAuth = os.Getenv("HTTP_BASIC")
	if basicAuth != "" {
		ss := strings.SplitN(basicAuth, ":", 2)
		if len(ss) != 2 {
			fmt.Println("Basic Auth: Expected <username>:<password>")
		}
	}

	apiEndpoint = args[0]

	// Placeholder replacements
	apiEndpoint = strings.ReplaceAll(apiEndpoint, ":hostname", h)

	checkNewConfig()

	ticker := time.NewTicker(updateFrequency)
	go func() {
		for range ticker.C {
			checkNewConfig()
		}
	}()

	// Never return
	<-make(chan struct{})
}

func checkNewConfig() {
	client := &http.Client{}

	req, err := http.NewRequest("GET", apiEndpoint, nil)

	if err != nil {
		fmt.Println("Error creating HTTP request")
		return
	}

	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	if basicAuth != "" {
		ss := strings.SplitN(basicAuth, ":", 2)
		if len(ss) == 2 {
			req.SetBasicAuth(ss[0], ss[1])
		}
	}

	res, err := client.Do(req)

	if err != nil {
		fmt.Println("HTTP request failed")
		return
	}

	fmt.Printf("HTTP/1.1 %d\n", res.StatusCode)

	if res.StatusCode != 200 {
		return
	}

	f, err := os.CreateTemp("", "compose")
	if err != nil {
		fmt.Println("Couldn't create temp file")
		return
	}
	defer os.Remove(f.Name())

	io.Copy(f, res.Body)

	if runCompose(f.Name()) != nil {
		fmt.Println("Problem running docker compose")

		// Clear out last modified and etag so that we can try to recover from an error if we're told to reapply last successful config
		lastModified = ""
		etag = ""

		return
	}

	// If we were successful then save headers
	if !noCache {
		lastModified = res.Header.Get("Last-Modified")
		etag = res.Header.Get("Etag")
	}
}

func runCompose(fileName string) error {
	cmd := exec.Command("docker", "compose", "-p", projectName, "-f", fileName, "up", "-d", "--remove-orphans")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
