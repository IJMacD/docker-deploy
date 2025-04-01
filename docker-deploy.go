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

const defaultProjectName = "zakkaya-deploy"
const defaultUpdateFrequencySeconds = 30

var projectName string
var updateFrequency time.Duration
var basicAuth string
var apiEndpoint string
var lastModified string
var etag string

func main () {
	var f int

	flag.StringVar(&projectName, "p", defaultProjectName, "Project name")
	flag.IntVar(&f, "i", defaultUpdateFrequencySeconds, "Update interval in seconds")
	flag.StringVar(&basicAuth, "http-basic", "", "HTTP Basic auth username:password")

	flag.Parse()

	updateFrequency = time.Duration(f) * time.Second

	args := flag.Args()

	if len(args) == 0 {
		fmt.Printf(`Error: apiEndpoint not specified.`)
		printUsage()
		return
	}

	apiEndpoint = args[0]

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

func checkNewConfig () {
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
		} else {
			fmt.Println("Basic Auth: Expected <username>:<password>")
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
	lastModified = res.Header.Get("Last-Modified")
	etag = res.Header.Get("Etag")
}

func runCompose(fileName string) error {
	cmd := exec.Command("docker", "compose", "-p", projectName, "-f", fileName, "up", "-d", "--remove-orphans")
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func printUsage () {
	fmt.Print(`
Usage:
	docker-deploy [OPTIONS] https://.../api/v1/machines/$(hostname -s)/docker-compose.yml
	docker-deploy [OPTIONS] https://.../api/v1/fleets/default/docker-compose.yml
`);
}