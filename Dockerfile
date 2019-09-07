# Builder 
FROM    golang:alpine as BUILDER
ENV     CGO_ENABLED=0 
RUN        apk update &&\
            apk add --no-cache ca-certificates make git &&\
            rm -rf /var/cache/apk/*
RUN     mkdir -p /go/src/github.com/deepk777/go-proxy/goproxy
WORKDIR /go/src/github.com/deepk777/go-proxy/goproxy
COPY    . .
RUN     make build

# Final Image
FROM       alpine:latest
COPY       --from=BUILDER /go/src/github.com/deepk777/go-proxy/goproxy /bin/
RUN         ls -l /bin/
ENTRYPOINT [ "/bin/go-proxy" ]
CMD        [ "-ca-certs-dir", "/root/cert.pem", "-server-cert-path", "/root/server.crt", "-server-key-path", "/root/server.key"]
