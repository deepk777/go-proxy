package goproxy

import (
	"encoding/json"

	"github.com/go-kit/kit/endpoint"
)

//Body .
type Body struct {
	TargetURL string           `json:"target"`
	Task      *json.RawMessage `json:"task"`
}

//Headers .
type Headers struct {
	Authorization string `json:"Authorization,omitempty"`
	RequestID     string `json:"x-request-id,omitempty"`
	ContentType   string `json:"Content-Type,omitempty"`
	XForwardedFor string `json:"X-Forwarded-For"`
}

//QueryString .
type QueryString struct {
	IsBeta bool
}

// Endpoints for every service method
type Endpoints struct {
	ReceiveAndForward endpoint.Endpoint
	HealthCheck       endpoint.Endpoint
	Version           endpoint.Endpoint
}

//ReceiveAndForwardRequest is request structure for /task
type ReceiveAndForwardRequest struct {
	Headers
	QueryString
	Body
}

//ReceiveAndForwardResponse is response structure for /task
type ReceiveAndForwardResponse struct {
	Status           int              `json:"status,omitempty"`
	Message          *json.RawMessage `json:"message,omitempty"`
	Reason           string           `json:"reason,omitempty"`
	Error            int              `json:"error,omitempty"`
	ErrorDescription error
}

//HealthCheckRequest is request structure for /healthcheck
type HealthCheckRequest struct{}

//HealthCheckResponse is response for /healthcheck endpoint
type HealthCheckResponse struct {
	Status string `json:"status"`
}

//VersionRequest is request structure for /Version endpoint
type VersionRequest struct{}

//VersionResponse is response for /Version endpoint
type VersionResponse struct {
	GoproxyVersion string `json:"goproxy,omitempty"`
}
