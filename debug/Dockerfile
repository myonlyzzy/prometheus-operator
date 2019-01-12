#Dockerfile for build a image debug in k8s
FROM golang:1.10-alpine3.7 AS build-env
MAINTAINER myonlyzzy@gmai.com
ENV CGO_ENABLED 0
RUN apk add --no-cache git

#compile prometheus-operator
RUN mkdir -p /go/src/github.com/myonlyzzy/
RUN git clone https://github.com/myonlyzzy/prometheus-operator /go/src/github.com/myonlyzzy/prometheus-operator

RUN go build -gcflags "all=-N -l" -o /prometheus-operator  /go/src/github.com/myonlyzzy/prometheus-operator/cmd/prometheus-operator/prometheus-operator.go

#compile dlv
RUN go get github.com/derekparker/delve/cmd/dlv

#Final stage
FROM alpine:3.7
EXPOSE 40000
RUN apk add --no-cache libc6-compat
WORKDIR /

COPY --from=build-env /prometheus-operator /
COPY --from=build-env /go/bin/dlv /

#RUN delve
CMD ["/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "exec", "/prometheus-operator"]



