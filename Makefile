BUILD_PATH ?= _build
PACKAGES_PATH ?= _packages
VERSION := $(shell git describe --tags --always)

PACKAGE_NAME := wormhole-$(VERSION)-$(shell go env GOOS)-$(shell go env GOARCH)

.PHONY: build
build:
	@CGO_ENABLED=0 go build -o agent main.go
	@mkdir -p $(BUILD_PATH)/$(PACKAGE_NAME)/etc
	@mkdir -p $(BUILD_PATH)/$(PACKAGE_NAME)/log
	@mv agent $(BUILD_PATH)/$(PACKAGE_NAME)
	@cp etc/client.yaml $(BUILD_PATH)/$(PACKAGE_NAME)/etc
	@echo "Build successfully"

.PHONY: pkg
pkg: build
	@mkdir -p $(PACKAGES_PATH)
	@cd $(BUILD_PATH) && zip -rq $(PACKAGE_NAME).zip $(PACKAGE_NAME)
	@cd $(BUILD_PATH) && tar -czf $(PACKAGE_NAME).tar.gz $(PACKAGE_NAME)
	@mv $(BUILD_PATH)/$(PACKAGE_NAME).zip $(BUILD_PATH)/$(PACKAGE_NAME).tar.gz $(PACKAGES_PATH)
	@echo "Package build successfully"
