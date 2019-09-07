package goproxy

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
)

// Middleware acts as wrapper
type Middleware func(service Service) Service

//ServiceLoggingMiddleware is used for logging on service layer.
func ServiceLoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return &loggingMiddlerware{
			next:   next,
			logger: logger,
		}
	}
}

//EndpointLoggingMiddleware is used for logging on endpoint layer.
func EndpointLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (output interface{}, err error) {

			defer func(begin time.Time) {
				// Taking max hard limit for total KV pair in a single log line as 100
				ilv := make([]interface{}, 0, 100)
				logLevel := "Info"

				if req, ok := request.(ReceiveAndForwardRequest); ok {
					ilv = createLogStyleInterface(ilv,
						"x-request-id", req.RequestID,
						"endpoint", "/task",
						"client-addr", req.XForwardedFor,
					)
				} else if _, ok := request.(VersionRequest); ok {
					ilv = createLogStyleInterface(ilv, "endpoint", "/version")
				} else if _, ok := request.(HealthCheckRequest); ok {
					ilv = createLogStyleInterface(ilv, "endpoint", "/health")
				} else {
					ilv = createLogStyleInterface(ilv, "error_description", ErrTypeAssertion.Error())
				}

				if r := recover(); r != nil {
					ilv = createLogStyleInterface(ilv, "traceback", string(debug.Stack()))
					e := fmt.Sprintf("%v", r)
					err = errors.New(e)
				}

				if err != nil {
					logLevel = "Error"
					if statusText := ValidHTTPStatusCode(err.Error()); statusText != "" {
						ilv = createLogStyleInterface(ilv, "error_description", statusText)
					} else {
						ilv = createLogStyleInterface(ilv, "error_description", err.Error())
					}
					err = nil
				}

				ilv = createLogStyleInterface(ilv, "took", time.Since(begin).String())
				logMyTask(logger, logLevel, ilv)

			}(time.Now())
			output, err = next(ctx, request)
			return output, err
		}
	}
}

// EndpointRequestValidationMiddleware is used for Request Validation on endpoint layer.
func EndpointRequestValidationMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			req := request.(ReceiveAndForwardRequest)
			var rf ReceiveAndForwardResponse

			//Header Validations
			if req.Headers.ContentType != "application/json" {
				rf.ErrorDescription = ErrInvalidContentType
				rf.Reason = ErrInvalidContentType.Error()
				return rf, ErrInvalidContentType
			}

			// Auth Validation
			// if req.Headers.Authorization == "" {
			// }

			//Body Validation
			if req.Body.TargetURL == "" {
				rf.ErrorDescription = ErrMissingTargetURL
				rf.Reason = ErrMissingTargetURL.Error()
				return rf, ErrMissingTargetURL
			}
			return next(ctx, request)
		}
	}
}
