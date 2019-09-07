package goproxy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type loggingMiddlerware struct {
	logger log.Logger
	next   Service
}

func (lmw *loggingMiddlerware) ReceiveAndForward(ctx context.Context, request ReceiveAndForwardRequest) (output ReceiveAndForwardResponse, err error) {

	defer func(begin time.Time) {
		ilv := make([]interface{}, 0, 100)
		logLevel := "Info"
		var msg interface{}

		if output.Message != nil {
			_, errMarshal := json.Marshal(&output.Message)
			if errMarshal != nil {
				msg = ""
			} else {
				msg = output.Message
			}
		}

		ilv = createLogStyleInterface(ilv,
			"method", "ReceiveAndForward",
			"response", msg,
			"request-url", request.Body.TargetURL,
			"x-request-id", request.RequestID,
		)

		if err != nil {
			logLevel = "Error"
			ilv = createLogStyleInterface(ilv, "error_description", err.Error())
			err = ErrInternalServerError

			if output.Status != 0 && output.Status != http.StatusOK {
				err = errors.New(strconv.Itoa(output.Status))
			}
		}

		ilv = createLogStyleInterface(ilv, "took", time.Since(begin).String())
		logMyTask(lmw.logger, logLevel, ilv)

	}(time.Now())

	output, err = lmw.next.ReceiveAndForward(ctx, request)

	return output, err
}

func (lmw *loggingMiddlerware) HealthCheck(ctx context.Context) (output HealthCheckResponse, err error) {

	defer func(begin time.Time) {
		ilv := make([]interface{}, 0, 100)
		logLevel := "Info"

		ilv = createLogStyleInterface(ilv,
			"method", "HealthCheck",
			"response", output.Status,
		)

		if err != nil {
			logLevel = "Error"
			ilv = createLogStyleInterface(ilv, "error_description", err.Error())
		}
		ilv = createLogStyleInterface(ilv, "took", time.Since(begin).String())
		logMyTask(lmw.logger, logLevel, ilv)

	}(time.Now())

	output, err = lmw.next.HealthCheck(ctx)
	return output, err
}

func (lmw *loggingMiddlerware) Version(ctx context.Context) (output VersionResponse, err error) {
	defer func(begin time.Time) {
		ilv := make([]interface{}, 0, 100)
		logLevel := "Info"

		ilv = createLogStyleInterface(ilv,
			"method", "Version",
			"response", output.GoproxyVersion,
		)

		if err != nil {
			logLevel = "Error"
			ilv = createLogStyleInterface(ilv, "error_description", err.Error())
		}
		ilv = createLogStyleInterface(ilv, "took", time.Since(begin).String())
		logMyTask(lmw.logger, logLevel, ilv)

	}(time.Now())

	output, err = lmw.next.Version(ctx)
	return output, err
}

//func logMyTask(logger log.Logger, logLevel string, logValues map[string]string) {
func logMyTask(logger log.Logger, logLevel string, ilv []interface{}) {

	if logLevel == "Error" {
		level.Error(logger).Log(ilv...)
	} else {
		logger.Log(ilv...)
	}
}

func createLogStyleInterface(ilv []interface{}, logValues ...interface{}) []interface{} {

	for _, v := range logValues {
		ilv = append(ilv, v)
	}
	return ilv
}
