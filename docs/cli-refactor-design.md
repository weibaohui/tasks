# CLI 重构设计文档

## 背景

当前 CLI 命令直接访问数据库，这违反了分层架构原则。需要重构为所有 CLI 命令通过 HTTP API 访问服务，经过 Token 认证。

## 目标

1. 所有 CLI 命令通过 HTTP API 调用服务端
2. 使用 Token 认证（Bearer Token）
3. 保持 CLI 接口不变（向后兼容）
4. 删除直接数据库访问代码

## 当前架构分析

### 直接访问数据库的命令（已重构）
- ✅ `requirement list` - 使用 HTTP API
- ✅ `requirement create` - 使用 HTTP API
- ✅ `requirement update` - 使用 HTTP API
- ✅ `requirement get` - 使用 HTTP API
- ✅ `requirement tasks` - 使用 HTTP API
- ✅ `requirement review` - 使用 HTTP API（创建需求）
- ✅ `requirement complete` - 使用 HTTP API
- ✅ `requirement reset` - 使用 HTTP API（新增 `/requirements/reset` 端点）
- ✅ `requirement redispatch` - 使用 HTTP API
- ✅ `agent list` - 使用 HTTP API
- ✅ `project list` - 使用 HTTP API
- ✅ `project heartbeat *` - 使用 HTTP API

### 原有的 HTTP API 命令
- `requirement dispatch` - 已使用 HTTP API

### 特殊命令（已废弃）
- `create-admin` - 改为在 server 端执行
- `delete-admin` - 改为在 server 端执行

## 新增的服务端 API 端点

### Requirement API
```
POST   /api/v1/requirements/reset          # 重置需求状态
```

## 重构方案

### 第一阶段：新增服务端 API

在 `router.go` 添加新的路由：
```go
mux.HandleFunc("/api/v1/requirements/reset", requireAuth(...))
```

在 `requirement_handler.go` 添加重置处理函数。

### 第二阶段：重构 CLI 命令

1. 创建 `backend/cmd/cli/client/client.go` - HTTP 客户端封装
2. 修改所有命令文件，使用 HTTP 客户端
3. 删除 `common.go` 中的数据库访问代码

### HTTP 客户端设计

```go
// client.go
package client

type Client struct {
    baseURL string
    token   string
    http    *http.Client
}

func New() *Client
func (c *Client) ListRequirements(ctx context.Context, projectID string) (*ListRequirementsResponse, error)
func (c *Client) CreateRequirement(ctx context.Context, req CreateRequirementRequest) (*Requirement, error)
func (c *Client) GetRequirement(ctx context.Context, id string) (*Requirement, error)
func (c *Client) UpdateRequirement(ctx context.Context, req UpdateRequirementRequest) (*Requirement, error)
func (c *Client) DispatchRequirement(ctx context.Context, req DispatchRequirementRequest) (*DispatchResult, error)
func (c *Client) CompleteRequirement(ctx context.Context, requirementID string) (*Requirement, error)
func (c *Client) ResetRequirement(ctx context.Context, requirementID string) (*Requirement, error)
func (c *Client) RedispatchRequirement(ctx context.Context, requirementID string) (*Requirement, error)
func (c *Client) CopyAndDispatchRequirement(ctx context.Context, requirementID string) (*Requirement, error)
func (c *Client) ListAgents(ctx context.Context) ([]Agent, error)
func (c *Client) ListProjects(ctx context.Context) ([]Project, error)
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error)
func (c *Client) UpdateProject(ctx context.Context, req UpdateProjectRequest) (*Project, error)
func (c *Client) UpdateProjectHeartbeat(ctx context.Context, projectID string, enabled bool, intervalMinutes int, mdContent, agentCode string) (*Project, error)
```

## 使用说明

### 1. 初始化配置文件

```bash
cd backend
go run cmd/cli/main.go config init
```

### 2. 创建管理员用户（在 server 端执行）

```bash
cd backend
go run cmd/server/main.go create-admin
```

### 3. 获取 API Token

1. 启动 server:
```bash
cd backend && go run cmd/server/main.go
```

2. 登录 Web UI (http://localhost:8888)
3. 进入 Personal Access Token 页面
4. 生成 Token 并复制

### 4. 配置 Token

编辑 `~/.taskmanager/config.yaml`:

```yaml
server:
  port: 8888

database:
  path: ~/.taskmanager/data.db

api:
  base_url: http://localhost:8888/api/v1
  token: your-api-token-here    # 从 Web UI 获取

logging:
  level: info
```

### 5. 使用 CLI

```bash
# 列出需求
go run cmd/cli/main.go requirement list

# 创建需求
go run cmd/cli/main.go requirement create -p <project_id> -t "需求标题" -d "需求描述"

# 派发需求
go run cmd/cli/main.go requirement dispatch <requirement_id>

# 重置需求
go run cmd/cli/main.go requirement reset --id <requirement_id>

# 列出项目
go run cmd/cli/main.go project list

# 查看心跳状态
go run cmd/cli/main.go project heartbeat status

# 列出 Agent
go run cmd/cli/main.go agent list
```

## 配置文件示例

```yaml
# ~/.taskmanager/config.yaml
server:
  port: 8888

database:
  path: ~/.taskmanager/data.db

api:
  base_url: http://localhost:8888/api/v1
  token: your-api-token-here    # 从 Web UI 获取

logging:
  level: info
```

## 错误处理

CLI 命令统一返回 JSON 格式错误：
```json
{"error": "error message"}
```

HTTP 客户端处理：
- 401 Unauthorized - Token 无效或过期
- 403 Forbidden - 权限不足
- 404 Not Found - 资源不存在
- 500 Internal Server Error - 服务器错误
- 网络连接错误

## 实现文件清单

### 新增文件
- `backend/cmd/cli/client/client.go` - HTTP 客户端封装

### 修改文件
- `backend/cmd/cli/cmd/common.go` - 删除数据库访问代码
- `backend/cmd/cli/cmd/requirementList.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/requirementCreate.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/requirementUpdate.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/requirementGet.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/requirementTasks.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/requirementReset.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/requirementDispatch.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/agent.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/project.go` - 使用 HTTP Client
- `backend/cmd/cli/cmd/createAdmin.go` - 标记为废弃
- `backend/cmd/cli/cmd/deleteAdmin.go` - 标记为废弃
- `backend/cmd/cli/cmd/config.go` - 显示 Token 配置
- `backend/interfaces/http/requirement_handler.go` - 添加 ResetRequirement 方法
- `backend/interfaces/http/router.go` - 添加 /requirements/reset 路由

## 测试验证

```bash
# 编译测试
cd backend
go build ./cmd/cli/...
go build ./cmd/server/...

# 单元测试
go test ./interfaces/http/...
go test ./cmd/cli/...
```
