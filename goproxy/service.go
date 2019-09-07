package goproxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

//ProxyVersion Information
var ProxyVersion string

// Constants used by service
const (
	UpstreamSelfServiceEndpoint = "/task"
	UpstreamHealthEndpoint      = "/health"
	DefaultUpstreamScheme       = "https"
	DefaultTimeout              = time.Second * 300
)

// Service defines a nss proxy interface
type Service interface {
	ReceiveAndForward(ctx context.Context, request ReceiveAndForwardRequest) (ReceiveAndForwardResponse, error)
	HealthCheck(ctx context.Context) (HealthCheckResponse, error)
	Version(ctx context.Context) (VersionResponse, error)
}

type service struct {
	upstreamPort   string
	upstreamCAFile string
	upstreamClient *http.Client
}

// Errorify used to represent http status and error
type Errorify struct {
	Status  int
	Err     error
	Message string
}

// NewService creates new service
func NewService(_ context.Context, upstreamPort string, upstreamClient *http.Client) (Service, error) {
	return &service{
		upstreamPort:   upstreamPort,
		upstreamClient: upstreamClient,
	}, nil
}

//MakeTLSClient to create a tls client
func MakeTLSClient(caFiles []string) (*http.Client, error) {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Append all CAs
	for _, caFile := range caFiles {
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, ErrCertLoadFailed
		}
		rootCAs.AppendCertsFromPEM(caCert)
	}

	tr := http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: true,
		},
		MaxIdleConns:    50,
		MaxConnsPerHost: 5,
		IdleConnTimeout: 300 * time.Second,
	}

	client := &http.Client{
		Timeout:   DefaultTimeout,
		Transport: &tr,
	}

	return client, nil
}

func (svc service) ReceiveAndForward(ctx context.Context, request ReceiveAndForwardRequest) (ReceiveAndForwardResponse, error) {

	var rf ReceiveAndForwardResponse
	inBytes, err := json.Marshal(request.Body.Task)
	if err != nil {
		return setReceiveAndForwardResponse(http.StatusBadRequest, ErrJSONUnMarshall.Error()),
			ErrJSONUnMarshall
	}

	var upstreamServer string
	var errStruct Errorify
	upstreamPort := svc.upstreamPort
	upstreamServer = DefaultUpstreamScheme + "://" + request.Body.TargetURL

	if err := testUpstreamHealth(upstreamServer, upstreamPort, svc.upstreamClient); err != (Errorify{}) {
		return setReceiveAndForwardResponse(err.Status, ErrUpstreamHealthCheckFailed.Error()),
			errors.Wrap(err.Err, ErrUpstreamHealthCheckFailed.Error())
	}

	// Creating Request object
	upstreamURL := upstreamServer + upstreamPort + UpstreamSelfServiceEndpoint
	req, err := http.NewRequest("POST", upstreamURL, bytes.NewBuffer(inBytes))
	if err != nil {
		return setReceiveAndForwardResponse(http.StatusInternalServerError, ErrFailedCreatingNewRequest.Error()),
			errors.Wrap(err, ErrFailedCreatingNewRequest.Error())
	}

	// Setting Request Headers
	req = setHeaders(request, req)

	// Setting Reqeust Queryparameters
	q := req.URL.Query()
	if request.QueryString.IsBeta {
		q.Add("beta", "true")
		req.URL.RawQuery = q.Encode()
	}

	resp, err := svc.upstreamClient.Do(req)
	if err != nil {
		errStruct = classifyRequestError(err)
		errMessage := errStruct.Err.Error()
		if errStruct.Message != "" {
			errMessage = errStruct.Message
		}
		return setReceiveAndForwardResponse(errStruct.Status, errMessage),
			errStruct.Err
	}
	defer resp.Body.Close()

	rf.Status = resp.StatusCode
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return setReceiveAndForwardResponse(http.StatusInternalServerError, ErrReadingResponseBody.Error()),
			errors.Wrap(err, ErrReadingResponseBody.Error())
	}

	if err := json.Unmarshal(responseBody, &rf); err != nil {
		return setReceiveAndForwardResponse(http.StatusInternalServerError, ErrReadingResponseBody.Error()),
			errors.Wrap(err, ErrReadingResponseBody.Error())
	}

	return rf, nil
}

func setHeaders(request ReceiveAndForwardRequest, req *http.Request) *http.Request {

	req.Header.Set("Authorization", request.Authorization)
	req.Header.Set("Content-Type", request.ContentType)
	req.Header.Set("X-Forwarded-For", request.XForwardedFor)
	req.Header.Set("x-request-id", request.RequestID)

	return req
}

func setReceiveAndForwardResponse(httpStatusCode int, reason string) ReceiveAndForwardResponse {

	var rf ReceiveAndForwardResponse
	rf.Status = httpStatusCode
	raw := json.RawMessage(http.StatusText(rf.Status))
	rf.Message = &raw
	rf.Reason = reason

	return rf
}

func testUpstreamHealth(upstreamServer, upstreamPort string, upstreamClient *http.Client) Errorify {

	var errStruct Errorify
	upstreamURL := upstreamServer + upstreamPort + UpstreamHealthEndpoint
	request, err := http.NewRequest("GET", upstreamURL, nil)
	if err != nil {
		errStruct = classifyRequestError(err)
		return errStruct
	}

	resp, err := upstreamClient.Do(request)
	if err != nil {
		errStruct = classifyRequestError(err)
		return errStruct
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return Errorify{}
	}

	// For anything not 200 ok
	errStruct.Status = http.StatusServiceUnavailable
	errStruct.Err = err
	return errStruct
}

func classifyRequestError(err error) Errorify {

	var errStruct Errorify
	errStruct.Status = http.StatusInternalServerError
	errStruct.Err = err

	switch err := err.(type) {
	case net.Error:
		if err.Timeout() {
			errStruct.Status = http.StatusGatewayTimeout
			errStruct.Message = "Request timeout"
		} else if strings.HasSuffix(err.Error(), "no such host") {
			errStruct.Status = http.StatusBadRequest
			errStruct.Message = ErrBadUpstreamURL.Error()
		} else if strings.HasSuffix(err.Error(), "connection refused") {
			errStruct.Status = http.StatusBadGateway
			errStruct.Message = "Connection Refused"
		}
	}
	return errStruct
}

func (service) HealthCheck(_ context.Context) (HealthCheckResponse, error) {
	return HealthCheckResponse{
		Status: "OK",
	}, nil
}

func (service) Version(_ context.Context) (VersionResponse, error) {
	return VersionResponse{
		GoproxyVersion: ProxyVersion,
	}, nil
}
