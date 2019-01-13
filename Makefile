#Makefile for prometheus operator
DOCKER_VERSION="latest"
PROMETHEUS_OPERATOR="cmd/prometheus-operator/prometheus-operator.go"
PROMETHEUS_OPERATOR_OUTDIR="cmd/prometheus-operator/prometheus-operator"
all: build docker

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${PROMETHEUS_OPERATOR_OUTDIR} ${PROMETHEUS_OPERATOR}
docker:
	docker build . -t myonlyzzy/prometheus-operator:${DOCKER_VERSION}
	docker push myonlyzzy/prometheus-operator:${DOCKER_VERSION}

