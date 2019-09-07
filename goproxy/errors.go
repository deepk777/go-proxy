package goproxy

import "errors"

// Transport Errors
var (
	//ErrInternalServerError will be returned in case of http 5xx errors
	ErrInternalServerError = errors.New("internal server error")

	//ErrInvalidContentType will be returned in case of content type is not application/json
	ErrInvalidContentType = errors.New("invalid content type")

	//ErrMissingTargetURL will be returned in case of target campaign host is not empty
	ErrMissingTargetURL = errors.New("missing target URL in request body")

	//ErrJSONUnMarshall will be returned in case of failed json parsing.
	ErrJSONUnMarshall = errors.New("failed to parse json")

	//ErrEmptyRequestBody will be returned in case of request body is empty
	ErrEmptyRequestBody = errors.New("empty request body")

	//ErrReadingResponseBody will be returned in case of response body is empty
	ErrReadingResponseBody = errors.New("empty response body")

	//ErrMalformedRequest will be returned in case of request sent is invalid
	ErrMalformedRequest = errors.New("request not formed correctly")

	// ErrRequestTimeout will be returned in case of request timed out
	ErrRequestTimeout = errors.New("request timeout")

	// ErrFailedCreatingNewRequest
	ErrFailedCreatingNewRequest = errors.New("failed creating new request")
)

// Endpoint Errors
var (
	// ErrTypeAssertion will be returned in case of unknown endpoint
	ErrTypeAssertion = errors.New("failed to type assert")
)

// Service Errors
var (
	// ErrUpstreamHealthCheckFailed will be returned in case of upstream server is un healthy
	ErrUpstreamHealthCheckFailed = errors.New("upstream health check failed")

	// ErrBadUpstreamURL
	ErrBadUpstreamURL = errors.New("upstream host not found")
)

// Certs Error
var (
	// ErrCertLoadFailed will be returned in case of upstream server is un healthy
	ErrCertLoadFailed = errors.New("failed to load certificate")
)

// Unknown Error
var (
	ErrUnknown = 3999
)
