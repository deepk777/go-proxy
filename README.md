# Go Proxy
[![Go Report Card](https://goreportcard.com/badge/github.com/deepk777/go-proxy?style=flat-square)](https://goreportcard.com/badge/github.com/deepk777/go-proxy)

**Table of Contents**

- [Overview](#overview)
- [Usage](#usage)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installing](#installing)
- [Sample Request](#sample-request)
- [Built With](#built-with)
- [Versioning](#versioning)
- [Acknowledgments](#acknowledgments)

## Overview 

This project aimed to create a lightweight web service written in golang listening over a port 443. Proxy uses [Mutual TLS](http://en.wikipedia.org/wiki/Transport_Layer_Security#Client-authenticated_TLS_handshake) to authenticate client. There is scope for adding authentication middleware. On success, request is forwarded with required headers to target host.
It follows the famous onion architecture.

## Usage
This proxy can be used for sending any request payload to target host and returning back any response payload. 
All you have to do is add request payload into `task` key and response payload from target host into `message` key.

For more information check 
#### Request Body Payload Struct 

```
type Body struct {
	TargetURL string           `json:"target"`
	Task      *json.RawMessage `json:"task"`
}
```

#### Response Body Payload Struct
```
type ReceiveAndForwardResponse struct {
	Status           int              `json:"status,omitempty"`
	Message          *json.RawMessage `json:"message,omitempty"`
	Reason           string           `json:"reason,omitempty"`
	Error            int              `json:"error,omitempty"`
	ErrorDescription error
}
```

## Endpoints
### Mutual TLS
- /task

### Only TLS
- /health
- /version

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

### Prerequisites

What things you need to install the software and how to install them

1) Install golang compiler<br>MAC OSx Users
```
brew install go
```
2) SSL/TLS Certificate
Steps to generate a self signed certificate.
```
openssl req -x509 -newkey rsa:2048 -keyout nssaproxy.key.pem -out goproxy.crt.pem -days 365 -nodes
```
3) Configure HTTP Client with client side certificate to make a successful request.
      
      Check [Sample Request](#sample-request)


### Installing

```
git clone https://github.com/deepk777/go-proxy
go build -o goproxy cmd/main.go
./goproxy
```

Examples
```
$ go build -o go-proxy cmd/main.go
$ ./go-proxy --help

Usage of go-proxy:
  -ca-certs-dir string
        Path of directory having list of allowed Certificate Authorities
  -log-conn-addr string
        Socket (address:port) of where to send logs (default "127.0.0.1:514")
  -log-level string
        Enable verbose log level. (default "info")
  -log-output string
        Log output location. 
         Valid options file, socket, stdout (default "stdout")
  -logdir string
        Log output directory (default "/var/log/goproxy")
  -monitoring-port string
        HTTPS listen address (default "5000")
  -server-cert-path string
        Path for Server crt
  -server-key-path string
        Path for Server key
  -tls-port string
        HTTPS listen address (default "443")
  -upstream-port string
        Denotes the port on which upstream service is running (default "12000")
```


## Sample Request

### GET /health
```
curl https://localhost:5000/health

{"status":"OK"}
```
### GET /version
```
curl https://localhost:5000/version

{"goproxy":"1.0.0"}
```

### POST /task
```
curl --key "client.key" --cert "client.crt" -X POST \
  https://localhost/task \
  -H 'Authorization: Bearer <TOKEN-GOES-HERE>' \
  -H 'Content-Type: application/json' \
  -H 'x-request-id: DA6016D4-BECB-4C95-A26D-53261D93092F' \
  -d '{
    "target" : "target-hostname",
    "task": {
    }
}'
```

## Built With

* [Golang](https://golang.org) - The Google Go Language
* [GoModules](https://github.com/golang/go/wiki/Modules) - Built-In Golang Modules Dependency Manager


## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/deepk777/go-proxy/tags). 


## Acknowledgments

* This project is built with [gokit](https://gokit.io) design.
