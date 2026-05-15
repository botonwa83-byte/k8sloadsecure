# K8sLoadSecure

> Kubernetes 集群访问控制网关

[![Version](https://img.shields.io/badge/version-v1.0.0-blue.svg)](CHANGELOG.md)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## 功能特性

- **细粒度权限控制**：基于角色的访问控制（RBAC）
- **多角色支持**：管理员、全局只读、开发者
- **项目隔离**：命名空间级别的资源隔离
- **K8s Dashboard 集成**：安全的 Dashboard 访问代理
- **审计追踪**：完整的操作日志记录
- **权限审批**：灵活的权限申请与审批流程

## 技术栈

- **后端**：Go 1.21 + Gin
- **前端**：Vue 3 + Element Plus
- **数据库**：MySQL 8.0+
- **K8s 客户端**：client-go

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- Kubernetes 集群

### 启动服务

```bash
# 启动后端服务
cd backend
cp .env.example .env
# 修改 .env 配置数据库连接
go run main.go

# 启动前端服务
cd frontend
npm install
npm run dev
```

### 访问地址

- 前端：http://localhost:5173
- 后端 API：http://localhost:8080
- 健康检查：http://localhost:8080/healthz

### 默认账户

```
用户名：admin
密码：Admin@123
角色：管理员
```

## 角色说明

| 角色 | 权限范围 | 说明 |
|------|----------|------|
| admin | 全部 | 集群管理员，拥有所有权限 |
| global_viewer | 全局只读 | 可查看所有资源，不能修改 |
| developer | 指定项目 | 只能访问分配的项目命名空间 |

## 权限流程

```
创建用户 → 创建项目 → 分配用户到项目 → 自动同步K8s权限 → 访问Dashboard
```

### 开发者权限申请

1. 登录开发者账户
2. 提交写权限申请
3. 管理员审批
4. 获得 24 小时写权限

## 项目结构

```
├── backend