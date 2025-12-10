package internals

import (
	"encoding/json"
	"fmt"
	"net/http"
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

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Response{
			StatusCode: resp.StatusCode,
			Error:      err,
			Duration:   time.Since(start).Seconds(),
		}, err
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = v[0]
	}

	cookies := make(map[string]string)
	for _, cookie := range resp.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}

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
