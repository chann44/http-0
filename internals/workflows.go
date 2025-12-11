package internals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseWorkFlows(workflowBytes []byte) (*Workflow, error) {
	var workflowDef Workflow
	if err := yaml.Unmarshal(workflowBytes, &workflowDef); err != nil {
		return nil, err
	}
	return &workflowDef, nil
}

func ExtractNestedData(data interface{}, path string) (interface{}, error) {
	if path == "" {
		return data, nil
	}

	parts := strings.Split(path, "|")
	current := data

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("key '%s' not found at path segment %d", part, i)
			}
			current = val

		case []interface{}:
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("expected array index at segment %d, got '%s'", i, part)
			}
			if idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("array index %d out of bounds (length: %d)", idx, len(v))
			}
			current = v[idx]

		default:
			return nil, fmt.Errorf("cannot traverse path at segment %d: current value is not a map or array", i)
		}
	}

	return current, nil
}

func SubstituteVariables(input string, ctx *WorkflowContext) (string, error) {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	var lastErr error
	result := re.ReplaceAllStringFunc(input, func(match string) string {
		content := strings.TrimSpace(match[2 : len(match)-2])

		if strings.Contains(content, "|") {
			parts := strings.SplitN(content, "|", 2)
			stepName := strings.TrimSpace(parts[0])
			path := strings.TrimSpace(parts[1])

			response, ok := ctx.Steps[stepName]
			if !ok {
				lastErr = fmt.Errorf("step '%s' not found in context", stepName)
				return match
			}

			if response.Error != nil {
				lastErr = fmt.Errorf("step '%s' had an error", stepName)
				return match
			}

			value, err := ExtractNestedData(response.Body, path)
			if err != nil {
				lastErr = fmt.Errorf("failed to extract data from step '%s': %v", stepName, err)
				return match
			}

			return fmt.Sprintf("%v", value)
		} else {
			if val, ok := ctx.Variables[content]; ok {
				return fmt.Sprintf("%v", val)
			}
			if val, ok := ctx.Global[content]; ok {
				return fmt.Sprintf("%v", val)
			}
			lastErr = fmt.Errorf("variable '%s' not found", content)
			return match
		}
	})

	return result, lastErr
}

func SubstituteInMap(m map[string]string, ctx *WorkflowContext) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range m {
		substituted, err := SubstituteVariables(v, ctx)
		if err != nil {
			return nil, err
		}
		result[k] = substituted
	}
	return result, nil
}

func SubstituteInBody(body interface{}, ctx *WorkflowContext) (interface{}, error) {
	if body == nil {
		return nil, nil
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %v", err)
	}

	bodyStr := string(bodyBytes)
	substituted, err := SubstituteVariables(bodyStr, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute variables in body: %v", err)
	}

	var result interface{}
	if err := json.Unmarshal([]byte(substituted), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal substituted body: %v", err)
	}

	return result, nil
}

func ExecuteWorkflows(workflow *Workflow, requestsMap map[string]*RequestDefinition, client *http.Client) error {
	ctx := &WorkflowContext{
		Steps:     make(map[string]*Response),
		Variables: workflow.Variables,
		Global:    make(map[string]interface{}),
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("WORKFLOW: %s\n", workflow.Name)
	fmt.Printf("%s\n", strings.Repeat("=", 60))

	for stepIdx, step := range workflow.Steps {
		fmt.Printf("\n[Step %d/%d] %s\n", stepIdx+1, len(workflow.Steps), step.Name)
		fmt.Println(strings.Repeat("-", 40))

		requestDef, ok := requestsMap[step.Request]
		if !ok {
			return fmt.Errorf("request '%s' not found for step '%s'", step.Request, step.Name)
		}

		stepRequest := &RequestDefinition{
			Name:    requestDef.Name,
			Method:  requestDef.Method,
			URL:     requestDef.URL,
			Headers: make(map[string]string),
			Query:   make(map[string]string),
		}

		for k, v := range requestDef.Headers {
			stepRequest.Headers[k] = v
		}
		for k, v := range requestDef.Query {
			stepRequest.Query[k] = v
		}

		if step.Headers != nil {
			for k, v := range step.Headers {
				stepRequest.Headers[k] = v
			}
		}
		if step.Query != nil {
			for k, v := range step.Query {
				stepRequest.Query[k] = v
			}
		}

		url, err := SubstituteVariables(stepRequest.URL, ctx)
		if err != nil {
			return fmt.Errorf("failed to substitute variables in URL for step '%s': %v", step.Name, err)
		}
		stepRequest.URL = url

		stepRequest.Headers, err = SubstituteInMap(stepRequest.Headers, ctx)
		if err != nil {
			return fmt.Errorf("failed to substitute variables in headers for step '%s': %v", step.Name, err)
		}

		stepRequest.Query, err = SubstituteInMap(stepRequest.Query, ctx)
		if err != nil {
			return fmt.Errorf("failed to substitute variables in query for step '%s': %v", step.Name, err)
		}

		var bodyToUse interface{}
		if step.Body != nil {
			bodyToUse = step.Body
		} else if requestDef.Body != nil {
			bodyBytes, err := io.ReadAll(requestDef.Body)
			if err == nil && len(bodyBytes) > 0 {
				json.Unmarshal(bodyBytes, &bodyToUse)
			}
		}

		if bodyToUse != nil {
			substitutedBody, err := SubstituteInBody(bodyToUse, ctx)
			if err != nil {
				return fmt.Errorf("failed to substitute variables in body for step '%s': %v", step.Name, err)
			}

			bodyJSON, err := json.Marshal(substitutedBody)
			if err != nil {
				return fmt.Errorf("failed to marshal body for step '%s': %v", step.Name, err)
			}
			stepRequest.Body = bytes.NewReader(bodyJSON)

			if _, ok := stepRequest.Headers["Content-Type"]; !ok {
				stepRequest.Headers["Content-Type"] = "application/json"
			}
		}

		httpRequest, err := Build(stepRequest)
		if err != nil {
			return fmt.Errorf("failed to build request for step '%s': %v", step.Name, err)
		}

		response, err := ExecuteRequest(client, httpRequest)
		if err != nil {
			fmt.Printf("ERROR in step '%s': %v\n", step.Name, err)
		}

		ctx.Steps[step.Name] = &response

		if step.Extract != nil {
			fmt.Println("\nExtracting variables:")
			for varName, path := range step.Extract {
				value, err := ExtractNestedData(response.Body, path)
				if err != nil {
					fmt.Printf("  Warning: failed to extract '%s': %v\n", varName, err)
					continue
				}
				ctx.Variables[varName] = value
				fmt.Printf("  %s = %v\n", varName, value)
			}
		}

		if response.Error != nil && step.OnError != nil {
			for _, cond := range step.OnError {
				if cond.Stop {
					return fmt.Errorf("workflow stopped due to error in step '%s': %v", step.Name, response.Error)
				}
				if cond.Goto != "" {
					// TODO: Implement goto logic if needed
					fmt.Printf("Warning: goto not yet implemented\n")
				}
			}
		}

		if response.Error == nil && step.OnSuccess != nil {
			for _, cond := range step.OnSuccess {
				if cond.Stop {
					fmt.Printf("\nWorkflow completed successfully after step '%s'\n", step.Name)
					return nil
				}
				if cond.Goto != "" {
					// TODO: Implement goto logic if needed
					fmt.Printf("Warning: goto not yet implemented\n")
				}
			}
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("WORKFLOW COMPLETED: %s\n", workflow.Name)
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))

	return nil
}
