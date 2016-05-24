.PHONY: build

build:	## Build project (after fetching depencencies if needed)
	docker run -it --rm -v `pwd`:/go -w /go -e GOPATH=/go -e GOBIN=/go/bin -e CGO_ENABLED=0 -e GOOS=linux golang /bin/bash -c "go get; go build -a -installsuffix cgo ecr-login.go"

container:	## Build container
	docker build -t sjourdan/ecr-login .

clean:	## Clean project
	rm -rf ./bin ./pkg ./src ; docker run -it --rm -v `pwd`:/go -w /go -e GOPATH=/go golang go clean

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
