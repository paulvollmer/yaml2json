VERSION=0.1.1

all: lint build test

lint:
	@go fmt
	@golint

build:
	@go build

test:
	./yaml2json .travis.yml

release:
	git tag -a v${VERSION} -m "Version ${VERSION}"
	git push origin v${VERSION}
	goreleaser

.PHONY: all lint build test release
