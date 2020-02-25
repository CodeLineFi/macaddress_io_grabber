# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOGET=$(GOCMD) get
BINARY_NAME=mac_grabber

MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_DIR := $(dir $(MAKEFILE_PATH))

BUILD_DIR=$(PROJECT_DIR)/build

.PHONY: build test fmt

all: fmt test build

build: fmt
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v

build-test:
	$(GOBUILD) -race -o $(BUILD_DIR)/$(BINARY_NAME) -v


fmt:
	$(GOFMT) ./...

test:
	$(GOTEST) -cover macaddress_io_grabber/...

clean:
	$(GOCLEAN)
	rm -f $(BUILD_DIR)/*
