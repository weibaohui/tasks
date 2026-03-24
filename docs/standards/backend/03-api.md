# HTTP API 设计规范

## 1. URL 规范

### 1.1 路径格式
```
/api/v1/{resource}
/api/v1/{resource}/{id}
/api/v1/{resource}/{id}/{sub-resource}
```

**示例**：
```
GET    /api/v1/agents              # 列表
POST   /api/v1/agents               # 创建
GET    /api/v1/agents/{id}          # 获取单个
PUT    /api/v1/agents/{id}          # 更新
DELETE /api/v1/agents/{id}          # 删除
GET    /api/v1/agents/{id}/bindings # 获取子资源
```

### 1.2 命名
- **小写** + **中划线**
- 示例：`conversation-records`（不是 `conversationRecords`）

## 2. HTTP 方法

| 方法 | 用途 | 幂等 |
|------|------|------|
| GET | 查询 | ✅ |
| POST | 创建 | ❌ |
| PUT | 全量更新 | ✅ |
| PATCH | 部分更新 | ❌ |
| DELETE | 删除 | ✅ |

## 3. 请求格式

### 3.1 Header
```
Content-Type: application/json
Authorization: Bearer {token}
```

### 3.2 Body
```json
{
  "name": "Agent 名称",
  "model": "gpt-4"
}
```

## 4. 响应格式

### 4.1 成功响应
```json
// 单个资源
{
  "id": "abc123",
  "name": "Agent 名称",
  "model": "gpt-4"
}

// 资源列表
[
  {"id": "abc123", "name": "Agent 1"},
  {"id": "def456", "name": "Agent 2"}
]
```

### 4.2 错误响应
```json
{
  "code": 400,
  "message": "参数错误",
  "details": "name 不能为空"
}
```

### 4.3 HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 204 | 删除成功（无内容） |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务器错误 |

## 5. 分页

### 5.1 请求参数
```
GET /api/v1/agents?limit=20&offset=0
```

### 5.2 响应格式
```json
{
  "data": [...],
  "total": 100,
  "limit": 20,
  "offset": 0
}
```

## 6. 排序

```
GET /api/v1/agents?sort=created_at:desc,name:asc
```

## 7. 过滤

```
GET /api/v1/conversation-records?user_code=usr_xxx&event_type=llm_call
```

## 8. 认证

### 8.1 Bearer Token
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### 8.2 响应
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": 1699999999999,
  "user": {
    "id": "usr_xxx",
    "username": "admin"
  }
}
```

## 9. 版本控制

- URL 路径：`/api/v1/`
- 只在重大变更时升级版本
- 旧版本至少保留 6 个月

## 10. CORS

```go
// CORS 头
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```
