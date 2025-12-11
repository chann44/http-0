package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/chann44/http-0/internals"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("failed to start client: %v\n", err)
		return
	}

	requestsDir := filepath.Join(cwd, "requests")
	requestFiles, err := internals.GetFiles(requestsDir)
	if err != nil {
		fmt.Printf("Failed to get request files: %v\n", err)
		return
	}

	requestsMap := make(map[string]*internals.RequestDefinition)
	for _, reqPath := range requestFiles {
		reqBytes, err := internals.ReadFileToBytes(reqPath)
		if err != nil {
			fmt.Printf("Failed to read request file %s: %v\n", reqPath, err)
			continue
		}
		parsedReq, err := internals.ParseRequestfromYaml(reqBytes)
		if err != nil {
			fmt.Printf("Failed to parse request from %s: %v\n", reqPath, err)
			continue
		}
		requestsMap[parsedReq.Name] = parsedReq
	}

	workflowsMap := make(map[string]*internals.Workflow)
	workflowsDir := filepath.Join(cwd, "workflows")
	if internals.DirExists(workflowsDir) {
		workflowFiles, err := internals.GetFiles(workflowsDir)
		if err != nil {
			fmt.Printf("Failed to get workflow files: %v\n", err)
			return
		}

		for _, wfPath := range workflowFiles {
			wfBytes, err := internals.ReadFileToBytes(wfPath)
			if err != nil {
				fmt.Printf("Failed to read workflow file %s: %v\n", wfPath, err)
				continue
			}
			workflow, err := internals.ParseWorkFlows(wfBytes)
			if err != nil {
				fmt.Printf("Failed to parse workflow from %s: %v\n", wfPath, err)
				continue
			}
			workflowsMap[workflow.Name] = workflow
		}
	}

	flag.Usage = func() {
		showHelp(requestsMap, workflowsMap)
	}
	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		showHelp(requestsMap, workflowsMap)
		return
	}

	client := &http.Client{}

	command := args[0]

	switch command {
	case "request", "req", "r":
		if len(args) < 2 {
			fmt.Println("Error: request name is required")
			fmt.Println("\nUsage: http-0 request <request-name>")
			return
		}
		requestName := args[1]
		executeRequest(requestName, requestsMap, client)

	case "workflow", "wf", "w":
		if len(args) < 2 {
			fmt.Println("Error: workflow name is required")
			fmt.Println("\nUsage: http-0 workflow <workflow-name>")
			return
		}
		workflowName := args[1]
		executeWorkflow(workflowName, workflowsMap, requestsMap, client)

	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		fmt.Println("\nRun 'http-0' without arguments to see available commands.")
	}
}

func showHelp(requestsMap map[string]*internals.RequestDefinition, workflowsMap map[string]*internals.Workflow) {
	fmt.Println("http-0 - HTTP request and workflow executor")
	fmt.Println("\nUsage:")
	fmt.Println("  http-0 request <request-name>   Execute a single request")
	fmt.Println("  http-0 workflow <workflow-name>  Execute a workflow")
	fmt.Println("\nShort aliases:")
	fmt.Println("  request: req, r")
	fmt.Println("  workflow: wf, w")

	// Show available requests
	if len(requestsMap) > 0 {
		fmt.Println("\nAvailable Requests:")
		requestNames := make([]string, 0, len(requestsMap))
		for name := range requestsMap {
			requestNames = append(requestNames, name)
		}
		sort.Strings(requestNames)
		for _, name := range requestNames {
			req := requestsMap[name]
			fmt.Printf("  - %s (%s %s)\n", name, req.Method, req.URL)
		}
	} else {
		fmt.Println("\nNo requests found in ./requests directory")
	}

	if len(workflowsMap) > 0 {
		fmt.Println("\nAvailable Workflows:")
		workflowNames := make([]string, 0, len(workflowsMap))
		for name := range workflowsMap {
			workflowNames = append(workflowNames, name)
		}
		sort.Strings(workflowNames)
		for _, name := range workflowNames {
			wf := workflowsMap[name]
			fmt.Printf("  - %s (%d steps)\n", name, len(wf.Steps))
		}
	} else {
		fmt.Println("\nNo workflows found in ./workflows directory")
	}

	fmt.Println("\nExamples:")
	if len(requestsMap) > 0 {
		for name := range requestsMap {
			fmt.Printf("  http-0 request %s\n", name)
			break
		}
	}
	if len(workflowsMap) > 0 {
		for name := range workflowsMap {
			fmt.Printf("  http-0 workflow %s\n", name)
			break
		}
	}
}

func executeRequest(requestName string, requestsMap map[string]*internals.RequestDefinition, client *http.Client) {
	requestDef, ok := requestsMap[requestName]
	if !ok {
		fmt.Printf("Error: request '%s' not found\n", requestName)
		fmt.Println("\nAvailable requests:")
		requestNames := make([]string, 0, len(requestsMap))
		for name := range requestsMap {
			requestNames = append(requestNames, name)
		}
		sort.Strings(requestNames)
		for _, name := range requestNames {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	fmt.Printf("Executing request: %s\n", requestName)
	req, err := internals.Build(requestDef)
	if err != nil {
		fmt.Printf("Failed to build request: %v\n", err)
		return
	}

	_, err = internals.ExecuteRequest(client, req)
	if err != nil {
		fmt.Printf("Failed to execute request: %v\n", err)
		return
	}
}

func executeWorkflow(workflowName string, workflowsMap map[string]*internals.Workflow, requestsMap map[string]*internals.RequestDefinition, client *http.Client) {
	workflow, ok := workflowsMap[workflowName]
	if !ok {
		fmt.Printf("Error: workflow '%s' not found\n", workflowName)
		fmt.Println("\nAvailable workflows:")
		workflowNames := make([]string, 0, len(workflowsMap))
		for name := range workflowsMap {
			workflowNames = append(workflowNames, name)
		}
		sort.Strings(workflowNames)
		for _, name := range workflowNames {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	if err := internals.ExecuteWorkflows(workflow, requestsMap, client); err != nil {
		fmt.Printf("Failed to execute workflow: %v\n", err)
		return
	}
}
