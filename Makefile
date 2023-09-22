IMAGE ?= quay.io/jotak/kafka-clickhouse-example:latest

OCI_BIN_PATH = $(shell which docker 2>/dev/null || which podman)
OCI_BIN ?= $(shell basename ${OCI_BIN_PATH})

.PHONY: vendors
vendors:
	go mod tidy && go mod vendor

.PHONY: build
build:
	go build -o bin/kafka-clickhouse-example cmd/main.go

.PHONY: image
image:
	$(OCI_BIN) build -t ${IMAGE} .
	$(OCI_BIN) push ${IMAGE}

.PHONY: deploy
deploy:
	kubectl apply -n netobserv -f contrib/topic.yaml
	kubectl apply -n netobserv -f contrib/deployment.yaml

.PHONY: deploy-stdout
deploy-stdout:
	kubectl apply -n netobserv -f contrib/topic.yaml
	kubectl apply -n netobserv -f contrib/deployment-stdout.yaml

.PHONY: undeploy
undeploy:
	kubectl delete -n netobserv -f contrib/deployment.yaml
	kubectl delete -n netobserv -f contrib/topic.yaml
