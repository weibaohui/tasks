.PHONY: help build clean dev dev-server dev-frontend stop test fmt lint setup install
SHELL := /bin/bash

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  make setup        - 安装依赖 (包括 air)"
	@echo "  make build        - 构建后端（含前端嵌入）"
	@echo "  make install      - 安装 taskmanager 到 ~/bin"
	@echo "  make clean        - 清理构建产物"
	@echo "  make dev          - 启动开发环境 (后端+前端热重载)"
	@echo "  make dev-server   - 启动开发环境 (air 热重载)"
	@echo "  make stop         - 停止服务"
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
build: build-frontend prepare-embed build-backend
	@echo "构建完成！"

build-backend:
	@echo "构建核心服务..."
	cd backend && go build -o bin/taskmanager-server ./cmd/server
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

# 安装
install: build
	@echo "安装 taskmanager 到 ~/bin..."
	@mkdir -p $(HOME)/bin
	@cp backend/bin/taskmanager-server $(HOME)/bin/taskmanager-server
	@cp backend/bin/taskmanager $(HOME)/bin/taskmanager
	@echo "安装完成！"
	@echo ""
	@echo "请确保 ~/bin 在 PATH 中:"
	@echo "  export PATH=\"$$HOME/bin:$$PATH\""
	@echo ""
	@echo "用法:"
	@echo "  taskmanager server start    启动服务"
	@echo "  taskmanager server stop     停止服务"
	@echo "  taskmanager server status   查看状态"
	@echo "  taskmanager <command>       查看其他命令"

# 清理
clean:
	rm -rf backend/bin/
	rm -f backend/*.log
	rm -rf backend/internal/embed/ui/dist/*
	@touch backend/internal/embed/ui/dist/.keep
	cd frontend && rm -rf dist/ node_modules/.vite

# 开发模式 - 同时启动后端（air 热重载）和前端（Vite 热重载）
dev:
	@echo "========================================="
	@echo "  启动 TaskManager 开发环境"
	@echo "========================================="
	@echo "  后端 API:  http://localhost:13618"
	@echo "  前端界面: http://localhost:3000"
	@echo "  按 Ctrl+C 停止所有服务"
	@echo "========================================="
	@mkdir -p backend/logs
	@mkdir -p ~/.taskmanager
	@(trap 'kill 0' INT; \
		CONFIG_PATH=$${TASKMANAGER_CONFIG:-$$HOME/.taskmanager/config.yaml}; \
		if [ ! -f "$$CONFIG_PATH" ]; then \
			cd backend && go run ./cmd/cli config init >/dev/null; \
			cd ..; \
		fi; \
		LOG_FILE=$$(awk '\
			/^logging:/ {in_logging=1; next} \
			/^[^[:space:]]/ {in_logging=0} \
			in_logging && $$1=="server_log_path:" {print $$2; exit} \
		' "$$CONFIG_PATH"); \
		if [ -z "$$LOG_FILE" ]; then \
			echo "配置文件缺少 logging.server_log_path: $$CONFIG_PATH"; \
			exit 1; \
		fi; \
		LOG_FILE=$${LOG_FILE%\"}; \
		LOG_FILE=$${LOG_FILE#\"}; \
		LOG_FILE=$${LOG_FILE%\'}; \
		LOG_FILE=$${LOG_FILE#\'}; \
		if [[ "$$LOG_FILE" == "~/"* ]]; then \
			LOG_FILE="$$HOME/$${LOG_FILE#~/}"; \
		fi; \
		echo "  日志文件: $$LOG_FILE"; \
		mkdir -p "$$(dirname "$$LOG_FILE")"; \
		set -a; source backend/.env; set +a; \
		echo "[1/2] 启动后端服务 (air)..."; \
		cd backend && air --build.cmd "go build -o bin/taskmanager-server ./cmd/server" --build.bin "./bin/taskmanager-server" 2>&1 | tee -a "$$LOG_FILE" & \
		sleep 2; \
		echo "[2/2] 启动前端服务 (vite)..."; \
		cd frontend && pnpm run dev & \
		wait)

# 启动开发模式 (air 热重载)
dev-server:
	@command -v air >/dev/null 2>&1 || { echo "air 未安装，正在安装..."; go install github.com/air-verse/air@latest; }
	@echo "启动服务 - air 热重载..."
	@mkdir -p backend/logs
	@mkdir -p ~/.taskmanager
	@CONFIG_PATH=$${TASKMANAGER_CONFIG:-$$HOME/.taskmanager/config.yaml}; \
	if [ ! -f "$$CONFIG_PATH" ]; then \
		cd backend && go run ./cmd/cli config init >/dev/null; \
		cd ..; \
	fi; \
	LOG_FILE=$$(awk '\
		/^logging:/ {in_logging=1; next} \
		/^[^[:space:]]/ {in_logging=0} \
		in_logging && $$1=="server_log_path:" {print $$2; exit} \
	' "$$CONFIG_PATH"); \
	if [ -z "$$LOG_FILE" ]; then \
		echo "配置文件缺少 logging.server_log_path: $$CONFIG_PATH"; \
		exit 1; \
	fi; \
	LOG_FILE=$${LOG_FILE%\"}; \
	LOG_FILE=$${LOG_FILE#\"}; \
	LOG_FILE=$${LOG_FILE%\'}; \
	LOG_FILE=$${LOG_FILE#\'}; \
	if [[ "$$LOG_FILE" == "~/"* ]]; then \
		LOG_FILE="$$HOME/$${LOG_FILE#~/}"; \
	fi; \
	echo "日志文件: $$LOG_FILE"; \
	mkdir -p "$$(dirname "$$LOG_FILE")"; \
	set -a; source backend/.env; set +a; cd backend && air --build.cmd "go build -o bin/taskmanager-server ./cmd/server" --build.bin "./bin/taskmanager-server" 2>&1 | tee -a "$$LOG_FILE"

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
