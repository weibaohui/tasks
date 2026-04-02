.PHONY: help build clean dev dev-server dev-web dev-api dev-frontend stop test fmt lint setup install
SHELL := /bin/bash

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  make setup        - 安装依赖 (包括 air)"
	@echo "  make build        - 构建后端和前端（含嵌入）"
	@echo "  make install      - 安装 taskmanager CLI 到 /usr/local/bin"
	@echo "  make clean        - 清理构建产物"
	@echo "  make dev          - 启动完整开发环境（server + web + frontend）"
	@echo "  make dev-server   - 仅启动核心服务 (air 热重载)"
	@echo "  make dev-web      - 仅启动 Web 服务 (air 热重载)"
	@echo "  make dev-api      - 仅启动 Web + 前端（不启动核心业务）"
	@echo "  make dev-frontend - 仅启动前端开发服务器"
	@echo "  make stop         - 停止所有 taskmanager 相关进程"
	@echo "  make test         - 运行测试"
	@echo "  make fmt          - 格式化代码"
	@echo "  make lint         - 运行代码检查"

# Setup - install dependencies
setup:
	@echo "安装 Go 依赖..."
	cd backend && go mod tidy
	@echo "安装前端依赖..."
	cd frontend && pnpm install
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/air-verse/air@latest; }

# 构建
build: build-frontend prepare-embed build-backend cleanup-embed
	@echo "构建完成！"

build-backend:
	@echo "构建核心服务 (server)..."
	cd backend && go build -o bin/taskmanager-server ./cmd/server
	@echo "构建 Web 服务 (web)..."
	cd backend && go build -o bin/taskmanager-web ./cmd/web
	@echo "构建 CLI..."
	cd backend && go build -o bin/taskmanager ./cmd/cli

build-frontend:
	@echo "构建前端..."
	cd frontend && pnpm run build

prepare-embed:
	@echo "准备嵌入目录..."
	@mkdir -p backend/internal/embed/ui/dist
	@rm -rf backend/internal/embed/ui/dist/*
	@cp -r frontend/dist/* backend/internal/embed/ui/dist/
	@echo "前端文件已复制到 backend/internal/embed/ui/dist/"

cleanup-embed:
	@echo "清理嵌入目录..."
	@rm -rf backend/internal/embed/ui/dist/*
	@touch backend/internal/embed/ui/dist/.keep
	@echo "嵌入目录已清理"

# 安装 CLI、Server 和 Web 到 /usr/local/bin
install: build
	@echo "安装 taskmanager 到 /usr/local/bin..."
	@cp backend/bin/taskmanager-server /usr/local/bin/taskmanager-server
	@cp backend/bin/taskmanager-web /usr/local/bin/taskmanager-web
	@cp backend/bin/taskmanager /usr/local/bin/taskmanager
	@echo "安装完成！"
	@echo ""
	@echo "用法:"
	@echo "  核心服务 (消息处理、任务调度):"
	@echo "    taskmanager server start    启动核心服务"
	@echo "    taskmanager server stop     停止核心服务"
	@echo ""
	@echo "  Web 服务 (HTTP API + 前端):"
	@echo "    taskmanager web start       启动 Web 服务（包含前端）"
	@echo "    taskmanager web stop        停止 Web 服务"
	@echo ""
	@echo "  其他 CLI 命令:"
	@echo "    taskmanager <command>       查看所有命令"

# 清理
clean:
	rm -rf backend/bin/
	rm -f backend/*.log
	rm -rf backend/internal/embed/ui/dist/*
	@touch backend/internal/embed/ui/dist/.keep
	cd frontend && rm -rf dist/ node_modules/.vite

# 开发模式 - 启动合并后的完整服务（包含核心业务 + HTTP API + 前端）
dev:
	@echo "========================================="
	@echo "  启动 TaskManager 开发环境"
	@echo "========================================="
	@echo "  服务地址: http://localhost:13618"
	@echo "  前端界面: http://localhost:13618"
	@echo "  日志文件: backend/logs/air.log"
	@echo "  按 Ctrl+C 停止所有服务"
	@echo "========================================="
	@mkdir -p backend/logs
	@(trap 'kill 0' INT; \
		set -a; source backend/.env; set +a; \
		echo "启动服务..."; \
		cd backend && air --build.cmd "go build -o bin/taskmanager-server ./cmd/server" --build.bin "./bin/taskmanager-server" 2>&1 | tee logs/air.log & \
		wait)

# 启动核心服务开发模式 (air 热重载) - 已合并为单一服务
dev-server:
	@command -v air >/dev/null 2>&1 || { echo "air 未安装，正在安装..."; go install github.com/air-verse/air@latest; }
	@echo "启动服务 (server) - air 热重载..."
	@mkdir -p backend/logs
	set -a; source backend/.env; set +a; cd backend && air --build.cmd "go build -o bin/taskmanager-server ./cmd/server" --build.bin "./bin/taskmanager-server" 2>&1 | tee logs/air.log

# 启动 Web 服务开发模式 (air 热重载) - 已废弃，请使用 dev-server
dev-web:
	@echo "dev-web 已废弃，服务已合并到 server，使用 make dev-server 启动"

# 启动前端开发服务器 (Vite)
dev-frontend:
	cd frontend && pnpm run dev

# 只启动前后端（Web API + 前端），不启动核心业务
dev-api:
	@echo "========================================="
	@echo "  启动 Web + 前端开发环境"
	@echo "========================================="
	@echo "  Web 服务: http://localhost:8888"
	@echo "  前端界面: http://localhost:3000"
	@echo "  日志文件: backend/logs/air.log"
	@echo "  按 Ctrl+C 停止所有服务"
	@echo "========================================="
	@mkdir -p backend/logs
	@(trap 'kill 0' INT; \
		set -a; source backend/.env; set +a; \
		echo "启动 Web 服务 (web)..."; \
		cd backend && SERVER_PORT=8888 air --build.cmd "go build -o bin/taskmanager-web ./cmd/web" --build.bin "SERVER_PORT=8888 ./bin/taskmanager-web" 2>&1 | tee logs/air.log & \
		echo "启动前端开发服务器..."; \
		cd frontend && pnpm run dev 2>&1 & \
		wait)

# 停止所有 TaskManager 相关进程
stop:
	@echo "正在停止 TaskManager 进程..."
	@-lsof -ti :13618 | xargs kill -9 2>/dev/null || true
	@-lsof -ti :3000 | xargs kill -9 2>/dev/null || true
	@sleep 1
	@echo "已停止 TaskManager 进程"

# 运行测试
test:
	@echo "运行后端测试..."
	cd backend && go test ./...
	@echo "运行前端测试..."
	cd frontend && pnpm run test 2>/dev/null || true

# 格式化代码
fmt:
	@echo "格式化 Go 代码..."
	cd backend && go fmt ./... && gofmt -w .
	@echo "格式化前端代码..."
	cd frontend && pnpm run fmt 2>/dev/null || true

# 代码检查
lint:
	@echo "运行 Go 代码检查..."
	cd backend && golangci-lint run 2>/dev/null || go vet ./...
	@echo "运行前端代码检查..."
	cd frontend && pnpm run lint 2>/dev/null || true
