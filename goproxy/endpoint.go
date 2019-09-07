package goproxy

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
)

// MakeProxyServiceEndpoints wrapper for creating endpoints
func MakeProxyServiceEndpoints(psvc Service) Endpoints {
	var endpoint Endpoints

	endpoint.ReceiveAndForward = makeReceiveAndForwardEndpoint(psvc)
	endpoint.HealthCheck = makeHealthCheckEndpoint(psvc)
	endpoint.Version = makeVersionEndpoint(psvc)

	return endpoint
}

func makeReceiveAndForwardEndpoint(psvc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ReceiveAndForwardRequest)
		output, err := psvc.ReceiveAndForward(ctx, req)
		if err != nil {
			return output, err
		}
		return output, nil
	}
}

func makeHealthCheckEndpoint(psvc Service) endpoint.Endpoint {
	return func(ctx context.Context, _ interface{}) (interface{}, error) {
		output, err := psvc.HealthCheck(ctx)
		return output, err
	}
}

func makeVersionEndpoint(psvc Service) endpoint.Endpoint {
	return func(ctx context.Context, _ interface{}) (interface{}, error) {
		output, err := psvc.Version(ctx)
		return output, err
	}
}

//MakeEndpointMiddlewares orchastrate all required middlewares
func MakeEndpointMiddlewares(endpoints Endpoints, logger log.Logger) Endpoints {

	//ReceiveAndForward Middlewares
	endpoints.ReceiveAndForward = EndpointRequestValidationMiddleware()(endpoints.ReceiveAndForward)
	endpoints.ReceiveAndForward = EndpointLoggingMiddleware(logger)(endpoints.ReceiveAndForward)

	// HealthCheck Middlewares
	endpoints.HealthCheck = EndpointLoggingMiddleware(logger)(endpoints.HealthCheck)

	// Version Middlewares
	endpoints.Version = EndpointLoggingMiddleware(logger)(endpoints.Version)

	return endpoints
}
