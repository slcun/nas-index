.PHONY: all build clean run test install deps

BINARY_NAME=nas-manager
VERSION=1.0.0

all: build

build: deps
	@echo "正在编译 $(BINARY_NAME)..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "编译完成！"

deps:
	@echo "正在下载依赖..."
	go mod download
	go mod tidy

run: build
	@echo "正在启动 $(BINARY_NAME)..."
	./$(BINARY_NAME)

clean:
	@echo "正在清理..."
	rm -f $(BINARY_NAME)
	@echo "清理完成！"

test:
	@echo "正在运行测试..."
	go test -v ./...

install: build
	@echo "正在安装 $(BINARY_NAME)..."
	@echo "安装功能待实现"

# 跨平台编译
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-darwin-amd64 ./cmd/$(BINARY_NAME)

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-windows-amd64.exe ./cmd/$(BINARY_NAME)

build-all: build-linux build-darwin build-windows
	@echo "所有平台编译完成！"
