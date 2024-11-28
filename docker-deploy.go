package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var lastModified string;
var etag string;

func main () {
	checkNewConfig()

	ticker := time.NewTicker(30 * time.Second)
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

	req, err := http.NewRequest("GET", "http://localhost:3000/docker-compose.yml", nil)

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