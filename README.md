# TaskManager

AI 原生软件开发平台任务调度系统。

## 项目概述

- **后端**：Go + DDD 架构
- **前端**：React + TypeScript + Ant Design
- **数据库**：SQLite

## 技术栈

### 后端
- Go 1.21+
- SQLite
- DDD 架构（domain → application → infrastructure → interfaces）

### 前端
- React 18
- TypeScript
- Ant Design 5
- Vite

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- pnpm 8+

### 安装依赖

```bash
make install
```

### 启动服务

```bash
taskmanager server start
```

### 前端开发

```bash
cd frontend && pnpm run dev
```

### 数据库

```bash
sqlite3 ~/.taskmanager/data.db
```

## 开发规范

### DDD 原则

- domain 层不引用其他层
- infrastructure 不引用 interfaces
- 应用服务是贫血模型（不含业务逻辑）
- 跨聚合直接修改必须通过聚合根方法

### 代码质量

- 必须通过 `go vet ./...`
- 核心逻辑必须有测试覆盖
- 遵循 Git 工作流规范

## 文档

- [CLAUDE.md](./CLAUDE.md) - Claude Code 开发规范
- [DDD.md](./DDD.md) - DDD 架构约束与最佳实践
- [docs](./docs/) - 详细设计文档

## 登录凭证

- 用户名：`test`
- 密码：`test`

## 调试

### 查看日志

```bash
tail -f ~/.taskmanager/server.log
```

### 远程调试

使用 Cloudflare Tunnel 创建临时公共 URL：

```bash
taskmanager tunnel
```
