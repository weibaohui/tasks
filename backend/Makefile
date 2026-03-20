.PHONY: help build clean dev dev-backend dev-web stop test fmt lint setup

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  make setup      - 安装依赖 (包括 air)"
	@echo "  make build      - 构建后端"
	@echo "  make clean      - 清理构建产物"
	@echo "  make dev        - 同时启动后端和前端开发服务器"
	@echo "  make dev-backend - 仅启动后端开发服务器 (air 热重载)"
	@echo "  make dev-web    - 仅启动前端开发服务器"
	@echo "  make stop       - 停止所有 taskmanager 相关进程"
	@echo "  make test       - 运行测试"
	@echo "  make fmt        - 格式化代码"
	@echo "  make lint       - 运行代码检查"

# Setup - install dependencies
setup:
	@echo "安装 Go 依赖..."
	go mod tidy
	@echo "安装前端依赖..."
	cd ../frontend && pnpm install
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/air-verse/air@latest; }

# 构建
build:
	@echo "构建后端..."
	go build -o bin/taskmanager ./cmd/server

# 清理
clean:
	rm -rf bin/
	rm -f *.log
	cd ../frontend && rm -rf dist/ node_modules/.vite

# 开发模式 - 同时启动后端和前端
dev:
	@echo "========================================="
	@echo "  启动 TaskManager 开发环境"
	@echo "========================================="
	@echo "  后端 API: http://localhost:8888"
	@echo "  前端界面: http://localhost:3000"
	@echo "  按 Ctrl+C 停止所有服务"
	@echo "========================================="
	@(trap 'kill 0' INT; \
		air 2>&1 & \
		cd ../frontend && pnpm run dev 2>&1 & \
		wait)

# 启动后端开发服务器 (air 热重载)
dev-backend:
	@command -v air >/dev/null 2>&1 || { echo "air 未安装，正在安装..."; go install github.com/air-verse/air@latest; }
	@echo "启动后端开发服务器 (air 热重载)..."
	air

# 启动前端开发服务器
dev-web:
	cd ../frontend && pnpm run dev

# 停止所有 taskmanager 相关进程
stop:
	@echo "正在停止所有 taskmanager 相关进程..."
	@echo "  - 停止 air..."
	@ps -ef | grep "air" | grep -v grep | awk '{print $$2}' | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 taskmanager 二进制..."
	@ps -ef | grep "taskmanager" | grep -v grep | awk '{print $$2}' | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 go run taskmanager..."
	@ps -ef | grep "go run" | grep "taskmanager" | grep -v grep | awk '{print $$2}' | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 make dev 相关 shell..."
	@ps -ef | grep -E "make dev|trap.*kill 0" | grep -v grep | awk '{print $$2}' | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 vite/node..."
	@pgrep -f "vite" | xargs -I {} kill -9 {} 2>/dev/null || true
	@echo "  - 停止 esbuild..."
	@pgrep -f "esbuild.*taskmanager" | xargs -I {} kill -9 {} 2>/dev/null || true
	@pgrep -f "esbuild.*frontend" | xargs -I {} kill -9 {} 2>/dev/null || true
	@sleep 1
	@remaining=$$(ps -ef | grep -E "(taskmanager|go run.*taskmanager|air)" | grep -v grep | wc -l); \
	if [ $$remaining -eq 0 ]; then \
		echo "已停止所有 taskmanager 进程"; \
	else \
		echo "警告: 仍有 $$remaining 个进程在运行"; \
		ps -ef | grep -E "(taskmanager|go run.*taskmanager|air)" | grep -v grep; \
	fi

# 运行测试
test:
	@echo "运行后端测试..."
	go test ./...
	@echo "运行前端测试..."
	cd ../frontend && pnpm run test 2>/dev/null || true

# 格式化代码
fmt:
	@echo "格式化 Go 代码..."
	go fmt ./...
	gofmt -w .
	@echo "格式化前端代码..."
	cd ../frontend && pnpm run fmt 2>/dev/null || true

# 代码检查
lint:
	@echo "运行 Go 代码检查..."
	golangci-lint run 2>/dev/null || go vet ./...
	@echo "运行前端代码检查..."
	cd ../frontend && pnpm run lint 2>/dev/null || true
