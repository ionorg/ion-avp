GO_LDFLAGS = -ldflags "-s -w"
GO_VERSION = 1.14
GO_TESTPKGS:=$(shell go list ./... | grep -v cmd | grep -v conf | grep -v node)
GO_COVERPKGS:=$(shell echo $(GO_TESTPKGS) | paste -s -d ',')

all: nodes

go_deps:
	go mod download

clean:
	rm -rf bin

nodes: go_deps
	go build -o bin/avp $(GO_LDFLAGS) cmd/server/grpc/main.go

test: nodes
	go test \
		-coverpkg=${GO_COVERPKGS} -coverprofile=cover.out -covermode=atomic \
		-v -race ${GO_TESTPKGS}
