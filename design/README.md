# 多任务管理系统设计文档 (DDD模式)

## 概述

构建一个支持并发执行、链路追踪、状态广播、Hook扩展的多任务管理系统，采用**领域驱动设计(DDD)**模式进行架构设计。

## 核心特性

- **并发执行**: 支持多任务并行执行，资源可控
- **链路追踪**: TraceID + SpanID + ParentSpanID 形成完整调用链
- **状态管理**: 任务全生命周期状态可见
- **广播机制**: 任务状态、进度、结果实时广播
- **Hook系统**: 任务各阶段可插入自定义逻辑
- **任务控制**: 支持强制停止、超时控制
- **任务树**: 通过 TraceID 查看完整任务链路

## 文档结构

| 文档 | 内容 |
|------|------|
| [architecture.md](./architecture.md) | DDD分层架构、整体架构图 |
| [domain.md](./domain.md) | 领域层：实体、值对象、领域事件、仓储接口、领域服务 |
| [application.md](./application.md) | 应用层：应用服务、DTO、工作池 |
| [infrastructure.md](./infrastructure.md) | 基础设施层：SQLite仓储、事件总线、ID生成器 |
| [interface-layer.md](./interface-layer.md) | 接口层：HTTP API、WebSocket适配器 |
| [examples.md](./examples.md) | 使用示例 |
| [directory-structure.md](./directory-structure.md) | 项目目录结构 |

## 快速开始

```go
// 1. 创建管理器
tm, _ := application.NewTaskApplicationService(...)

// 2. 创建任务
task, _ := tm.CreateTask(ctx, application.CreateTaskCommand{
    Name: "数据处理",
    Type: "processing",
})

// 3. 启动任务
tm.StartTask(ctx, task.ID(), handler, hooks)

// 4. 查询任务树
tree, _ := tm.GetTaskTree(ctx, task.TraceID())
```

## 技术选型

| 功能 | 选型 |
|------|------|
| ID生成 | nanoID |
| 日志 | zap |
| 存储 | SQLite (modernc.org/sqlite) |
| 架构 | DDD分层 |

## DDD概念对照表

| DDD概念 | 本项目实现 |
|---------|-----------|
| 实体 (Entity) | Task |
| 值对象 (Value Object) | TaskID, TraceID, Progress, Result |
| 聚合根 (Aggregate Root) | Task |
| 领域事件 (Domain Event) | TaskCreatedEvent, TaskStartedEvent... |
| 仓储 (Repository) | TaskRepository接口 |
| 领域服务 (Domain Service) | TaskTreeBuilder, TaskExecutor |
| 应用服务 (App Service) | TaskApplicationService |
| DTO | CreateTaskCommand, GetTaskDTO |
