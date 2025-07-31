.PHONY: init dep migrations mock lint lint-dupl test bench build build-linux build-aarch64 clean all serve cov

init:
	#pip install pre-commit
	pre-commit install
	# go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.3.0
	# for make doc
	go get -u -d github.com/swaggo/swag/cmd/swag@latest
	# for make mock
	go install go.uber.org/mock/mockgen@latest
	# for goimports
	go install -v github.com/incu6us/goimports-reviser/v3@latest

dep:
	go mod tidy

doc:
	swag init

godoc:
	godoc -http=127.0.0.1:6060 -goroot="."

mock:
	go generate ./...

fmt:
	golines ./ -m 120 -w --base-formatter gofmt --no-reformat-tags
	gofumpt  -l -w .
	goimports-reviser -rm-unused -excludes '.git,vendor' -format ./...

lint:
	golangci-lint run

lint-dupl:
	golangci-lint run --no-config --disable-all --enable=dupl

test:
ifeq ($(shell go env GOOS)-$(shell go env GOARCH),darwin-arm64)
		GOARCH=amd64 go test -gcflags=all=-l $(shell go list ./... | grep -v mock | grep -v docs) -covermode=count -coverprofile .coverage.cov
else
		go test -gcflags=all=-l $(shell go list ./... | grep -v mock | grep -v docs) -covermode=count -coverprofile .coverage.cov
endif

cov:
	go tool cover -html=.coverage.cov

bench:
	go test -run=nonthingplease -benchmem -bench=. $(shell go list ./... | grep -v /vendor/)

build:
	# go build .
	go build -tags=jsoniter -ldflags "-X bkauth/pkg/version.Version=${VERSION} -X bkauth/pkg/version.Commit=${COMMIT} -X bkauth/pkg/version.BuildTime=`date +%Y-%m-%d_%I:%M:%S` -X 'bkauth/pkg/version.GoVersion=`go version`'" .

build-linux:
	# GOOS=linux GOARCH=amd64 go build .
	GOOS=linux GOARCH=amd64 go build  -tags=jsoniter -ldflags "-X bkauth/pkg/version.Version=${VERSION} -X bkauth/pkg/version.Commit=${COMMIT} -X bkauth/pkg/version.BuildTime=`date +%Y-%m-%d_%I:%M:%S` -X 'bkauth/pkg/version.GoVersion=`go version`'" .

build-aarch64:
	GOOS=linux GOARCH=arm64 go build  -tags=jsoniter -ldflags "-X bkauth/pkg/version.Version=${VERSION} -X bkauth/pkg/version.Commit=${COMMIT} -X bkauth/pkg/version.BuildTime=`date +%Y-%m-%d_%I:%M:%S` -X 'bkauth/pkg/version.GoVersion=`go version`'" .

all: lint test build

serve: build
	./bkauth -c config.yaml

docker-build:
	docker build --build-arg VERSION=`git describe --tags --abbrev=0` --build-arg COMMIT=`git rev-parse HEAD` -t bkauth:development .
