TAG ?= latest
IMAGE_NAME ?= monitoring-agent
REGISTRY ?= localhost:5000/homelab
DOCKER_IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(TAG)

.PHONY: build push test test-cover compile

build:
	docker build -t $(DOCKER_IMAGE) .

push:
	docker push $(DOCKER_IMAGE)

test:
	go test -coverprofile=c.out -coverpkg=./... ./...
	go tool cover -html=c.out -o test-coverage.html

test-cover:
	go test -coverprofile=c.out -coverpkg=./... ./...
	go tool cover -func=c.out

compile:
