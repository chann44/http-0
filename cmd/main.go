package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chann44/http-0/internals"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("failed to start client", err)
	}

	cwd = filepath.Join(cwd, "requests")

	requests, err := internals.GetAllRequestInProject(cwd)
	if err != nil {
		fmt.Printf("Failed to process requests: %v\n", err)
		return
	}

	var parsedRequests []internals.RequestDefinition

	for _, reqPath := range requests {
		reqBytes, err := internals.ReadRequestFile(reqPath)
		if err != nil {
			fmt.Printf("Failed to read request file %s: %v\n", reqPath, err)
			continue
		}
		parsedReq, err := internals.ParseRequestfromYaml(reqBytes)
		if err != nil {
			fmt.Printf("Failed to parse request from %s: %v\n", reqPath, err)
			continue
		}
		parsedRequests = append(parsedRequests, *parsedReq)
	}

	var requeestsToProcess []*http.Request

	for _, rp := range parsedRequests {
		req, err := internals.Build(&rp)
		if err != nil {
			fmt.Printf("Failed to build request %s: %v\n", rp.Name, err)
			continue
		}
		requeestsToProcess = append(requeestsToProcess, req)
	}

	client := &http.Client{}

	for _, req := range requeestsToProcess {
		_, err := internals.ExecuteRequest(client, req)
		if err != nil {
			fmt.Printf("Failed to execute request: %v\n", err)
			continue
		}
	}
}
