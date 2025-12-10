package internals

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func Build(r *RequestDefinition) (req *http.Request, err error) {
	req, err = http.NewRequest(r.Method, r.URL, r.Body)

	if len(r.Headers) != 0 {
		for k, v := range r.Headers {
			req.Header.Set(k, v)
		}
	}

	if len(r.Query) != 0 {
		q := req.URL.Query()
		for k, v := range r.Query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	return req, err
}

func ExecuteRequest(client *http.Client, request *http.Request) (Response, error) {
	start := time.Now()

	resp, err := client.Do(request)
	if err != nil {
		return Response{Error: err, Duration: time.Since(start).Seconds()}, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{
			StatusCode: resp.StatusCode,
			Error:      err,
			Duration:   time.Since(start).Seconds(),
		}, err
	}

	var result interface{}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") || json.Valid(bodyBytes) {
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			result = string(bodyBytes)
		}
	} else {
		result = string(bodyBytes)
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = v[0]
	}

	cookies := make(map[string]string)
	for _, cookie := range resp.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("Request: %s %s\n", request.Method, request.URL.String())
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Status Code: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	fmt.Printf("Duration: %.3f seconds\n", time.Since(start).Seconds())

	if resp.Header.Get("X-Request-Id") != "" {
		fmt.Printf("Request ID: %s\n", resp.Header.Get("X-Request-Id"))
	}

	fmt.Println("\nHeaders:")
	for k, v := range headers {
		fmt.Printf("  %s: %s\n", k, v)
	}

	if len(cookies) > 0 {
		fmt.Println("\nCookies:")
		for k, v := range cookies {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	fmt.Println("\nResponse Body:")
	if jsonData, ok := result.(map[string]interface{}); ok {
		prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
		if err == nil {
			fmt.Println(string(prettyJSON))
		} else {
			fmt.Printf("%v\n", result)
		}
	} else if jsonArray, ok := result.([]interface{}); ok {
		prettyJSON, err := json.MarshalIndent(jsonArray, "", "  ")
		if err == nil {
			fmt.Println(string(prettyJSON))
		} else {
			fmt.Printf("%v\n", result)
		}
	} else {
		fmt.Printf("%v\n", result)
	}
	fmt.Println(strings.Repeat("=", 60))

	return Response{
		StatusCode: resp.StatusCode,
		Body:       result,
		Headers:    headers,
		Cookies:    cookies,
		Duration:   time.Since(start).Seconds(),
		RequestId:  resp.Header.Get("X-Request-Id"),
	}, nil
}

func ParseRequestfromYaml(requestStr []byte) (*RequestDefinition, error) {
	var requestDef RequestDefinition

	if err := yaml.Unmarshal(requestStr, &requestDef); err != nil {
		return nil, err
	}

	if requestDef.URL == "" || requestDef.Method == "" {
		return nil, fmt.Errorf("invalid request: missing required fields")
	}

	return &requestDef, nil
}
