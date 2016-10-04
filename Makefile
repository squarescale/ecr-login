.PHONY: build

build:	## Build project (after fetching depencencies if needed)
	docker run -it --rm -v `pwd`:/go -w /go -e GOPATH=/go -e GOBIN=/go/bin -e CGO_ENABLED=0 -e GOOS=linux golang /bin/bash -c "go get; go build -a -installsuffix cgo ecr-login.go"

build-local:	## Build project using local go compiler (after fetching depencencies if needed)
	rm -rf go/src/ecr-login
	mkdir -p go/src/ecr-login go/bin go/pkg
	cp *.go go/src/ecr-login
	GOPATH=$$PWD/go go get -d ./go/src/ecr-login
	GOPATH=$$PWD/go CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo-linux ./go/src/ecr-login

container:	## Build container
	docker build -t sjourdan/ecr-login .

clean:	## Clean project
	rm -rf ./bin ./pkg ./src ; docker run -it --rm -v `pwd`:/go -w /go -e GOPATH=/go golang go clean

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
