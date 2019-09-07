package goproxy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"

	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// MakeHTTPHandler returns an http handler for the endpoints
func MakeHTTPHandler(endpoints Endpoints) (http.Handler, http.Handler) {
	r := mux.NewRouter()
	r1 := mux.NewRouter()

	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(encodeError),
	}
	receiveAndForwardHandler := httptransport.NewServer(
		endpoints.ReceiveAndForward,
		decodeReceiveAndForwardRequest,
		encodeReceiveAndForwardResponse,
		options...,
	)
	r.Methods("POST").Path("/task").Handler(receiveAndForwardHandler)

	healthCheckHandler := httptransport.NewServer(
		endpoints.HealthCheck,
		decodeHealthCheckRequest,
		encodeGenericResponse,
		options...,
	)
	r1.Methods("GET").Path("/health").Handler(healthCheckHandler)

	versionHandler := httptransport.NewServer(
		endpoints.Version,
		decodeVersionRequest,
		encodeGenericResponse,
		options...,
	)
	r1.Methods("GET").Path("/version").Handler(versionHandler)

	return r, r1
}

func copyHeaders(req ReceiveAndForwardRequest, r *http.Request) ReceiveAndForwardRequest {

	req.Headers.Authorization = r.Header.Get("Authorization")
	req.Headers.RequestID = r.Header.Get("x-request-id")
	req.Headers.ContentType = r.Header.Get("Content-Type")

	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
			clientIP = prior + "," + clientIP
		}
		req.Headers.XForwardedFor = clientIP
	}

	return req
}

func decodeReceiveAndForwardRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req ReceiveAndForwardRequest
	if e := json.NewDecoder(r.Body).Decode(&req.Body); e != nil {
		switch {
		case e == io.EOF:
			return nil, ErrEmptyRequestBody
		default:
			return nil, ErrMalformedRequest
		}
	}
	defer r.Body.Close()

	// copy headers from incoming request for logging and forwarding purposes
	// also request-id is good candidates for context package.
	req = copyHeaders(req, r)

	// check if reqeust has beta flag.
	isBeta := r.URL.Query().Get("beta")
	if len(isBeta) > 0 {
		beta, err := strconv.ParseBool(isBeta)
		if err != nil {
			return "", err
		}
		req.QueryString.IsBeta = beta
	}

	return req, nil
}

func encodeReceiveAndForwardResponse(_ context.Context, w http.ResponseWriter, resp interface{}) error {
	if e, ok := resp.(errorer); ok && e.error() != nil {
		w.WriteHeader(codeFrom(e.error()))
		json.NewEncoder(w).Encode(resp.(ReceiveAndForwardResponse))
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	httpStatusText := ErrInternalServerError
	jsonResponse := make(map[string]interface{})

	if response, ok := resp.(ReceiveAndForwardResponse); ok {

		if response.Reason != "" {
			jsonResponse["reason"] = response.Reason
		}

		if response.Status > 0 {
			httpStatus := strconv.Itoa(response.Status)
			httpStatusText = errors.New(httpStatus)
		} else if err := response.ErrorDescription; err != nil {
			httpStatusText = response.ErrorDescription
		} else {
			httpStatusText = ErrInternalServerError
		}

		if response.Message != nil {
			jsonResponse["message"] = ""
			_, err := json.Marshal(&response.Message)
			if err == nil {
				jsonResponse["message"] = response.Message
			}
		}

		if response.Error != 0 {
			jsonResponse["error"] = ErrUnknown
			_, err := json.Marshal(&response.Error)
			if err == nil {
				jsonResponse["error"] = response.Error
			}
		}
	}

	httpStatusCode := codeFrom(httpStatusText)
	w.WriteHeader(httpStatusCode)

	jsonResponse["status"] = httpStatusCode
	if val, ok := jsonResponse["message"]; !ok || ok && val == "" {
		jsonResponse["message"] = http.StatusText(httpStatusCode)
	}

	return json.NewEncoder(w).Encode(jsonResponse)
}

func decodeHealthCheckRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return HealthCheckRequest{}, nil
}
func decodeVersionRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return VersionRequest{}, nil
}

type errorer interface {
	error() error
}

func encodeGenericResponse(ctx context.Context, w http.ResponseWriter, resp interface{}) error {
	if e, ok := resp.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(resp)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	if err == nil {
		panic("encodeError with nil error")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	httpStatusCode := codeFrom(err)
	w.WriteHeader(httpStatusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  httpStatusCode,
		"message": http.StatusText(httpStatusCode),
		"reason":  err.Error(),
	})
}

func codeFrom(err error) int {
	switch err {
	case ErrJSONUnMarshall, ErrMissingTargetURL, ErrEmptyRequestBody, ErrMalformedRequest,
		ErrBadUpstreamURL:
		return http.StatusBadRequest
	case ErrInvalidContentType:
		return http.StatusUnsupportedMediaType
	case ErrInternalServerError, ErrFailedCreatingNewRequest,
		ErrReadingResponseBody, ErrTypeAssertion:
		return http.StatusInternalServerError
	case ErrRequestTimeout, ErrUpstreamHealthCheckFailed:
		return http.StatusServiceUnavailable
	default:
		if statusText := ValidHTTPStatusCode(err.Error()); statusText != "" {
			code, _ := strconv.Atoi(err.Error())
			return code
		}
		return http.StatusInternalServerError
	}
}

// ValidHTTPStatusCode takes in a string and return true if the string can
// be represented as valid http status code
func ValidHTTPStatusCode(code string) string {

	codeNumber, err := strconv.Atoi(code)
	if err != nil {
		return ""
	}
	statusText := http.StatusText(codeNumber)
	if statusText == "" {
		return ""
	}
	return statusText
}
