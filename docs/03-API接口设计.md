# API 接口设计

基础路径：`/api/v1`

所有接口（除登录和修改密码外）均需在请求头中携带 JWT Token，通过 HttpOnly Cookie 自动传递。

---

## 1. 认证模块

### 1.1 用户登录

```
POST /api/v1/auth/login
```

**请求体：**
```json
{
  "username": "zhangsan",
  "password": "Abc123456"
}
```

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": {
    "user_id": 1,
    "username": "zhangsan",
    "display_name": "张三",
    "role": "developer",
    "password_expired": false
  }
}
```

**失败响应 (401)：**
```json
{
  "code": 40101,
  "message": "用户名或密码错误"
}
```

**业务逻辑：**
- 验证账号密码
- 连续失败 5 次锁定 30 分钟
- 密码过期时返回 `password_expired: true`，前端跳转改密页
- 成功后签发 JWT，写入 HttpOnly Cookie（有效期 8 小时）
- 记录 login_logs

### 1.2 用户登出

```
POST /api/v1/auth/logout
```

**成功响应 (200)：**
```json
{
  "code": 0,
  "message": "已登出"
}
```

### 1.3 修改密码

```
PUT /api/v1/auth/password
```

**请求体：**
```json
{
  "old_password": "Abc123456",
  "new_password": "Xyz789012"
}
```

**密码规则：**
- 最少 8 位
- 必须包含大写字母、小写字母和数字
- 不能与前一次密码相同

**成功响应 (200)：**
```json
{
  "code": 0,
  "message": "密码修改成功，请重新登录"
}
```

### 1.4 获取当前用户信息

```
GET /api/v1/auth/me
```

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": {
    "user_id": 1,
    "username": "zhangsan",
    "display_name": "张三",
    "email": "zhangsan@company.com",
    "role": "developer",
    "projects": [
      {
        "project_id": 1,
        "project_name": "万方数据",
        "permission": "readwrite",
        "namespaces": ["wfks", "for-wfks"]
      }
    ],
    "password_expires_at": "2026-08-15T00:00:00Z"
  }
}
```

---

## 2. 用户管理模块（仅 Admin）

### 2.1 获取用户列表

```
GET /api/v1/users?page=1&page_size=20&keyword=zhang&role=developer&status=active
```

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": {
    "total": 128,
    "list": [
      {
        "id": 1,
        "username": "zhangsan",
        "display_name": "张三",
        "email": "zhangsan@company.com",
        "role": "developer",
        "status": "active",
        "password_expires_at": "2026-08-15T00:00:00Z",
        "created_at": "2026-01-10T08:00:00Z"
      }
    ]
  }
}
```

### 2.2 创建用户

```
POST /api/v1/users
```

**请求体：**
```json
{
  "username": "lisi",
  "password": "InitPass123",
  "display_name": "李四",
  "email": "lisi@company.com",
  "role": "viewer"
}
```

### 2.3 更新用户信息

```
PUT /api/v1/users/:id
```

**请求体：**
```json
{
  "display_name": "李四（新）",
  "email": "lisi_new@company.com",
  "role": "developer",
  "status": "active"
}
```

### 2.4 重置用户密码

```
PUT /api/v1/users/:id/reset-password
```

**请求体：**
```json
{
  "new_password": "ResetPass123"
}
```

### 2.5 删除用户

```
DELETE /api/v1/users/:id
```

---

## 3. 项目管理模块（仅 Admin）

### 3.1 获取项目列表

```
GET /api/v1/projects?page=1&page_size=20&keyword=万方
```

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": {
    "total": 15,
    "list": [
      {
        "id": 1,
        "name": "万方数据主站",
        "description": "万方数据主站相关服务",
        "namespaces": ["wfks", "for-wfks", "new-wfks"],
        "user_count": 12,
        "created_at": "2026-01-10T08:00:00Z"
      }
    ]
  }
}
```

### 3.2 创建项目

```
POST /api/v1/projects
```

**请求体：**
```json
{
  "name": "万方数据主站",
  "description": "万方数据主站相关服务",
  "namespaces": ["wfks", "for-wfks"]
}
```

### 3.3 更新项目

```
PUT /api/v1/projects/:id
```

**请求体：**
```json
{
  "name": "万方数据主站",
  "description": "更新描述",
  "namespaces": ["wfks", "for-wfks", "new-wfks"]
}
```

### 3.4 删除项目

```
DELETE /api/v1/projects/:id
```

### 3.5 获取集群所有命名空间

```
GET /api/v1/namespaces
```

**说明：** 从 K8s API 实时获取，供创建/编辑项目时选择命名空间使用。

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": ["agentscope", "aiscope", "crm", "default", "dify", "wfks", "..."]
}
```

### 3.6 为项目分配用户

```
POST /api/v1/projects/:id/users
```

**请求体：**
```json
{
  "user_id": 5,
  "permission": "readwrite"
}
```

### 3.7 移除项目用户

```
DELETE /api/v1/projects/:id/users/:user_id
```

---

## 4. 审计日志模块

### 4.1 查询操作日志

```
GET /api/v1/audit/logs?page=1&page_size=50&user_id=1&action=DELETE&namespace=wfks&start_time=2026-05-01&end_time=2026-05-15
```

**权限说明：**
- Admin：可查看所有用户的日志
- 普通用户：只能查看自己的日志（自动过滤 user_id）

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": {
    "total": 320,
    "list": [
      {
        "id": 10086,
        "username": "zhangsan",
        "action": "DELETE",
        "resource_type": "Pod",
        "resource_name": "web-server-abc123",
        "namespace": "wfks",
        "request_path": "/api/v1/namespaces/wfks/pods/web-server-abc123",
        "status_code": 200,
        "client_ip": "192.168.1.100",
        "detail": "删除 Pod",
        "created_at": "2026-05-15T14:02:33Z"
      }
    ]
  }
}
```

### 4.2 获取个人操作报告

```
GET /api/v1/audit/report?user_id=1&start_time=2026-05-01&end_time=2026-05-15
```

**权限说明：**
- Admin：可查看任意用户报告（需传 user_id）
- 普通用户：只能查看自己的报告（忽略 user_id 参数）

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": {
    "user_id": 1,
    "username": "zhangsan",
    "display_name": "张三",
    "period": {
      "start": "2026-05-01",
      "end": "2026-05-15"
    },
    "summary": {
      "total_operations": 128,
      "by_action": {
        "GET": 43,
        "POST": 32,
        "PUT": 37,
        "PATCH": 8,
        "DELETE": 8
      },
      "by_result": {
        "success": 120,
        "failed": 5,
        "denied": 3
      },
      "active_namespaces": ["wfks", "for-wfks"],
      "active_days": 11
    },
    "sensitive_operations": [
      {
        "id": 10086,
        "action": "DELETE",
        "resource_type": "Deployment",
        "resource_name": "payment-service",
        "namespace": "wfks",
        "created_at": "2026-05-12T10:30:00Z"
      }
    ]
  }
}
```

### 4.3 导出操作日志（CSV）

```
GET /api/v1/audit/export?user_id=1&start_time=2026-05-01&end_time=2026-05-15
```

**响应：** 直接返回 CSV 文件流，Content-Type 为 `text/csv`。

### 4.4 获取全局统计（仅 Admin）

```
GET /api/v1/audit/stats?start_time=2026-05-01&end_time=2026-05-15
```

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": {
    "total_operations": 15680,
    "active_users": 89,
    "top_users": [
      {"username": "zhangsan", "count": 320},
      {"username": "lisi", "count": 280}
    ],
    "top_namespaces": [
      {"namespace": "wfks", "count": 5200},
      {"namespace": "dify", "count": 3100}
    ],
    "denied_operations": 42,
    "by_action": {
      "GET": 8500,
      "POST": 3200,
      "PUT": 2800,
      "PATCH": 680,
      "DELETE": 500
    }
  }
}
```

---

## 5. K8s Dashboard 代理

### 5.1 代理请求

```
ALL /dashboard/*
```

**说明：**
- 所有匹配 `/dashboard/` 前缀的请求转发到后端 K8s Dashboard 服务
- 代理层自动注入对应权限的 ServiceAccount Token
- 代理层自动记录审计日志（通过中间件）
- 非 GET 请求额外解析请求体，提取操作详情写入 audit_logs.detail

---

## 6. 登录日志（仅 Admin）

### 6.1 查询登录日志

```
GET /api/v1/login-logs?page=1&page_size=50&username=zhangsan&result=failed&start_time=2026-05-01&end_time=2026-05-15
```

**成功响应 (200)：**
```json
{
  "code": 0,
  "data": {
    "total": 56,
    "list": [
      {
        "id": 1001,
        "username": "zhangsan",
        "client_ip": "192.168.1.100",
        "result": "failed",
        "reason": "密码错误",
        "created_at": "2026-05-15T09:00:00Z"
      }
    ]
  }
}
```

---

## 7. 通用响应格式

**成功：**
```json
{
  "code": 0,
  "data": {},
  "message": "操作成功"
}
```

**失败：**
```json
{
  "code": 40001,
  "message": "错误描述"
}
```

**错误码规范：**

| 错误码 | 含义 |
|--------|------|
| 0 | 成功 |
| 40001 | 参数错误 |
| 40101 | 认证失败（用户名或密码错误） |
| 40102 | Token 过期 |
| 40103 | 密码已过期，需修改 |
| 40104 | 账号已锁定 |
| 40105 | 账号已禁用 |
| 40301 | 权限不足 |
| 50001 | 服务器内部错误 |
| 50002 | K8s API 调用失败 |
