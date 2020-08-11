GO_LDFLAGS = -ldflags "-s -w"
GO_VERSION = 1.14
GO_TESTPKGS:=$(shell go list ./... | grep -v cmd | grep -v conf | grep -v node)
GO_COVERPKGS:=$(shell echo $(GO_TESTPKGS) | paste -s -d ',')

go_deps:
	go mod download

clean:
	rm -rf bin

build: go_deps
	go build -o bin/avp $(GO_LDFLAGS) examples/save-to-webm/server/main.go

test: go_deps
	go test \
		-coverpkg=${GO_COVERPKGS} -coverprofile=cover.out -covermode=atomic \
		-v -race ${GO_TESTPKGS}
