package internals

import "io"

type Response struct {
	StatusCode int
	Body       interface{}
	Headers    map[string]string
	Cookies    map[string]string
	Error      error
	Duration   float64
	RequestId  string
}

type Workflow struct {
	Name        string
	Environment string
	Variables   map[string]interface{}
	Steps       []WorkflowStep
}

type WorkflowStep struct {
	Name      string
	Request   string
	Variables map[string]interface{}
	Body      interface{}
	Headers   map[string]string
	Query     map[string]string
	Extract   map[string]string
	OnSuccess []Condition
	OnError   []Condition
}

type Condition struct {
	Condition string
	Goto      string
	Stop      bool
}

type RequestDefinition struct {
	Name    string
	Method  string
	URL     string
	Headers map[string]string
	Body    io.Reader
	Query   map[string]string
}

type RequestFile struct {
	Requests []RequestDefinition
}

type Config struct {
	Environments map[string]map[string]interface{}
	Default      string
	Timeout      int
	Retries      int
}

type WorkflowContext struct {
	Steps     map[string]*Response
	Variables map[string]interface{}
	Global    map[string]interface{}
}
