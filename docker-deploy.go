package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const defaultProjectName = "zakkaya-deploy"
const defaultUpdateFrequency = 30

var projectName string
var updateFrequency time.Duration
var updateFrequencySeconds int
var apiEndpoint string
var lastModified string
var etag string

func main () {
	flag.StringVar(&projectName, "p", defaultProjectName, "Project name")
	flag.IntVar(&updateFrequencySeconds, "i", defaultUpdateFrequency, "Update interval in seconds")

	flag.Parse()

	updateFrequency = time.Duration(updateFrequencySeconds) * time.Second

	args := flag.Args()

	if len(args) == 0 {
		fmt.Printf("apiEndpoint not specified.\n\nUsage: docker-deploy [OPTIONS] http://.../docker-compose.yml\n")
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

	res, err := client.Do(req)

	if err != nil {
		fmt.Println("HTTP request failed")
		return
	}

	fmt.Printf("HTTP/1.1 %d\n", res.StatusCode)

	if res.StatusCode == 200 {
		lastModified = res.Header.Get("Last-Modified")
		etag = res.Header.Get("Etag")

		f, err := os.CreateTemp("", "compose")
		if err != nil {
			fmt.Println("Couldn't create temp file")
			return
		}
		defer os.Remove(f.Name())

		io.Copy(f, res.Body)

		runCompose(f.Name())
	}
}

func runCompose(fileName string) {
	cmd := exec.Command("docker-compose", "-p", "zakkaya-deploy", "-f", fileName, "up", "-d")
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}