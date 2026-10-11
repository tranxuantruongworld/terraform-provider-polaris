NAME := polaris
BINARY := terraform-provider-${NAME}
VERSION := 1.0.0
OS_ARCH := $(shell go env GOOS)_$(shell go env GOARCH)

.PHONY: default build test generate install clean

default: build

build:
	go build -o bin/${BINARY} .

test:
	go test -v ./...

generate:
	/home/xtruong/go/bin/tfplugindocs generate

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/tranxuantruongworld/${NAME}/${VERSION}/${OS_ARCH}
	cp bin/${BINARY} ~/.terraform.d/plugins/registry.terraform.io/tranxuantruongworld/${NAME}/${VERSION}/${OS_ARCH}/${BINARY}_v${VERSION}

clean:
	rm -rf bin/

