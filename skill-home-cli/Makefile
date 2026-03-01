# skill-home CLI Makefile

.PHONY: all build install clean test lint fmt vet

# 版本信息
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# 构建参数
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"
BINARY := skill-home

# 默认目标
all: build

# 构建
build:
	@echo "Building $(BINARY)..."
	cd cmd/skill-home && go build $(LDFLAGS) -o ../../bin/$(BINARY)
	@echo "Build complete: bin/$(BINARY)"

# 安装到系统
install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	@cp bin/$(BINARY) /usr/local/bin/
	@echo "Installation complete"

# 清理构建产物
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean
	@echo "Clean complete"

# 运行测试
test:
	@echo "Running tests..."
	@go test -v ./...

# 代码格式化
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# 代码检查
vet:
	@echo "Running go vet..."
	@go vet ./...

# 静态检查 (需要安装 golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run

# 下载依赖
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# 跨平台构建
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	# macOS AMD64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 ./cmd/skill-home
	# macOS ARM64
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 ./cmd/skill-home
	# Linux AMD64
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/skill-home
	# Linux ARM64
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 ./cmd/skill-home
	# Windows AMD64
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe ./cmd/skill-home
	@echo "Build complete: dist/"

# 创建发布包
release: build-all
	@echo "Creating release archives..."
	@cd dist && \
	for file in $(BINARY)-*; do \
		if [[ "$$file" == *.exe ]]; then \
			zip "$$file.zip" "$$file" && rm "$$file"; \
		else \
			tar -czf "$$file.tar.gz" "$$file" && rm "$$file"; \
		fi; \
	done
	@echo "Release archives created: dist/"

# 开发模式运行
dev:
	@go run ./cmd/skill-home

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  install    - Install to /usr/local/bin"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format code"
	@echo "  vet        - Run go vet"
	@echo "  lint       - Run linter"
	@echo "  deps       - Download and tidy dependencies"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  release    - Create release archives"
	@echo "  dev        - Run in development mode"
	@echo "  help       - Show this help"
