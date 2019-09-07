package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	proxy "github.com/deepk777/go-proxy/goproxy"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/peterbourgon/ff"
)

const (
	// UDP used for sending logs on udp port
	UDP = "udp"

	// DefaultSocket address to emit logs
	DefaultSocket = "127.0.0.1:514"
)

func main() {

	fs := flag.NewFlagSet("go-proxy", flag.ExitOnError)
	var (
		tlsPort        = fs.String("tls-port", "443", "HTTPS listen address")
		monitoringPort = fs.String("monitoring-port", "5000", "HTTPS listen address")
		logLevel       = fs.String("log-level", "info", "Enable verbose log level.")
		logOut         = fs.String("log-output", "stdout", "Log output location. \n Valid options file, socket, stdout")
		logConnAddr    = fs.String("log-conn-addr", DefaultSocket, "Socket (address:port) of where to send logs")
		caCertsDir     = fs.String("ca-certs-dir", "", "Path of directory having list of allowed Certificate Authorities")
		serverCert     = fs.String("server-cert-path", "", "Path for Server crt")
		serverKey      = fs.String("server-key-path", "", "Path for Server key")
		upstreamPort   = fs.String("upstream-port", "12000", "Denotes the port on which upstream service is running")
		logDirectory   = fs.String("logdir", "/var/log/goproxy", "Log output directory")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarNoPrefix(),
	)

	ctx := context.Background()
	errChan := make(chan error)

	var httpsAddress string
	httpsAddress = *tlsPort
	defaultEndpointPort := ":" + httpsAddress

	httpsAddress = *monitoringPort
	monitoringEndpointPort := ":" + httpsAddress

	httpsAddress = *upstreamPort
	upstreamEndpointPort := ":" + httpsAddress

	// Initialize logger
	logger, err := logSetup(*logOut, *logLevel, *logConnAddr, *logDirectory)
	if err != nil {
		logAndExit(logger, err)
	}

	caFiles, err := filePathWalkDir(*caCertsDir)
	if err != nil {
		logAndExit(logger, err)
	}
	upstreamClient, err := proxy.MakeTLSClient(caFiles)
	if err != nil {
		logAndExit(logger, err)
	}

	service, err := proxy.NewService(ctx, upstreamEndpointPort, upstreamClient)
	if err != nil {
		logAndExit(logger, err)
	}
	level.Debug(logger).Log("msg", "service initialized")
	service = proxy.ServiceLoggingMiddleware(logger)(service)

	//Endpoints
	endpoints := proxy.MakeProxyServiceEndpoints(service)
	endpoints = proxy.MakeEndpointMiddlewares(endpoints, logger)
	level.Debug(logger).Log("msg", "endpoint middlewares installed")

	//HTTP Transport
	mutualTLSHandler, nonMutualTLSHandler := proxy.MakeHTTPHandler(endpoints)

	go func() {
		level.Info(logger).Log("serverStatus", "listening", "port", defaultEndpointPort)

		// MutualTLS setup
		tlsConfig, err := configureMutualTLS(caFiles, *serverCert, *serverKey)
		if err != nil {
			logAndExit(logger, err)
		}
		l, err := tls.Listen("tcp4", defaultEndpointPort, tlsConfig)
		if err != nil {
			logAndExit(logger, err)
		}
		errChan <- http.Serve(l, mutualTLSHandler)

	}()

	go func() {
		level.Info(logger).Log("serverStatus", "listening", "port", monitoringEndpointPort)

		tlsConfig, err := configureServerTLS(*serverCert, *serverKey)
		if err != nil {
			logAndExit(logger, err)
		}
		l, err := tls.Listen("tcp4", monitoringEndpointPort, tlsConfig)
		if err != nil {
			logAndExit(logger, err)
		}
		errChan <- http.Serve(l, nonMutualTLSHandler)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	level.Error(logger).Log("Error", <-errChan)
}

func logSetup(logOut, logLevel, logConnAddr, logDirectory string) (log.Logger, error) {

	var logger log.Logger
	{
		logger = log.NewJSONLogger(os.Stdout)
		if logOut == "socket" {
			conn, err := net.Dial(UDP, logConnAddr)
			if err != nil {
				return nil, err
			}
			logger = log.NewJSONLogger(conn)
		} else if logOut == "file" {
			logFilePath := path.Join(logDirectory, "goproxy.log")
			fp, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, err
			}
			logger = log.NewJSONLogger(fp)
		}

		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "service", "go-proxy")

		switch strings.ToLower(logLevel) {
		case "debug":
			logger = level.NewFilter(logger, level.AllowDebug())
		default:
			logger = level.NewFilter(logger, level.AllowInfo())
		}
	}
	return logger, nil
}

func filePathWalkDir(caDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(caDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func configureClientTLS(caFiles []string, tlsConfig *tls.Config) (*tls.Config, error) {
	//https://en.wikipedia.org/wiki/Transport_Layer_Security#Client-authenticated_TLS_handshake

	clientCertPool := x509.NewCertPool()
	for _, caFile := range caFiles {
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		clientCertPool.AppendCertsFromPEM(caCert)
	}

	// Reject any TLS certificate that cannot be validated
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	//Ensure we are only using the CA we trust
	tlsConfig.ClientCAs = clientCertPool

	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

func configureServerTLS(serverCert, serverKey string) (*tls.Config, error) {
	cer, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return &tls.Config{}, err
	}

	tlsConfig := tls.Config{Certificates: []tls.Certificate{cer}}
	return &tlsConfig, nil
}

func configureMutualTLS(caFiles []string, serverCert, serverKey string) (*tls.Config, error) {

	tlsConfig, err := configureServerTLS(serverCert, serverKey)
	if err != nil {
		return tlsConfig, err
	}

	tlsConfig, err = configureClientTLS(caFiles, tlsConfig)
	if err != nil {
		return tlsConfig, err
	}

	return tlsConfig, nil
}

func logAndExit(logger log.Logger, err error) {
	level.Error(logger).Log("msg", err)
	os.Exit(1)
}
