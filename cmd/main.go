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

	fmt.Printf(cwd)
	requests, err := internals.GetAllRequestInProject(cwd)
	fmt.Printf("%v", requests)

	if err != nil {
		fmt.Printf("failed to process requests", err)
	}

	var parsedRequests []internals.RequestDefinition

	for _, reqPath := range requests {
		reqBytes, err := internals.ReadRequestFile(reqPath)
		if err != nil {
			fmt.Printf("failed to parse request from yaml", err)
			continue
		}
		parsedReq, err := internals.ParseRequestfromYaml(reqBytes)
		if err != nil {
			fmt.Printf("failed to parse request bytes int requet defination", err)
			continue
		}
		parsedRequests = append(parsedRequests, *parsedReq)
	}

	var requeestsToProcess []*http.Request

	for _, rp := range parsedRequests {
		req, err := internals.Build(&rp)
		if err != nil {
			fmt.Printf("Failed to prcess request %v", rp.Name)
			continue
		}
		requeestsToProcess = append(requeestsToProcess, req)
	}

}
