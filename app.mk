GOPATH:=$(shell go env GOPATH)
APP_VERSION=$(shell git describe --tags --always)
APP_RELATIVE_PATH=$(shell a=`basename $$PWD` && cd .. && b=`basename $$PWD` && echo $$b/$$a)

.PHONY: dep api conf ent wire openapi build clean run test

# download dependencies of module
dep:
	@go mod download

# generate protobuf api go code
api:
	@cd ../../../ && \
	buf generate

# generate config define code
conf:
	@buf generate --path internal/conf --template internal/conf/buf.conf.gen.yaml

# generate ent code
ent:
ifneq ("$(wildcard ./internal/data/ent)","")
	@go run -mod=mod entgo.io/ent/cmd/ent generate \
				--feature privacy \
				--feature entql \
				--feature sql/modifier \
				--feature sql/upsert \
				./internal/data/ent/schema
endif

# generate wire code
wire:
	@go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server

# generate OpenAPI v3 doc
openapi:
	@cd ../../../ && \
	buf generate --path api/pallas/service/v1 --template api/pallas/service/v1/buf.openapi.gen.yaml

# build golang application
build:
ifeq ("$(wildcard ./bin/)","")
	mkdir bin
endif
	@go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

# clean build files
clean:
	@go clean

# run application
run:
	@go run ./cmd/server -conf ./configs

# run tests
test:
	@go test -v ./... -cover

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help