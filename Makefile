# Application name
APP_NAME := nsq_exporter

# Define build directories for each platform
BUILD_DIR := build
DARWIN_ARM64_BINARY := $(BUILD_DIR)/$(APP_NAME)-darwin-arm64
LINUX_AMD64_BINARY := $(BUILD_DIR)/$(APP_NAME)-linux-amd64

# Default Go build flags
GO_BUILD_FLAGS := -ldflags "-s -w"

# Build for darwin/arm64
$(DARWIN_ARM64_BINARY):
	@echo "Building $(DARWIN_ARM64_BINARY)..."
	GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(DARWIN_ARM64_BINARY) .

# Build for linux/amd64
$(LINUX_AMD64_BINARY):
	@echo "Building $(LINUX_AMD64_BINARY)..."
	GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(LINUX_AMD64_BINARY) .

.PHONY: docker-build-linux
docker-build-linux: $(LINUX_AMD64_BINARY)
	docker build --platform=linux/amd64 -t sysfiller/nsq_exporter:latest .

# Clean up binaries
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	@echo "Cleaned up build files."
