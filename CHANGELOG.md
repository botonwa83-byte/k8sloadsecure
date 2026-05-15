# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-05-15

### Added

- **用户管理模块**
  - 用户创建、编辑、删除功能
  - 密码重置与过期管理
  - 三角色体系：admin、global_viewer、developer

- **项目管理模块**
  - 项目创建与命名空间关联
  - 用户-项目权限分配
  - 默认只读权限配置

- **K8s Dashboard 集成**
  - ServiceAccount 自动创建
  - 基于角色的 RBAC 权限同步
  - 跳过登录配置优化

- **审计日志系统**
  - 操作日志记录与查询
  - 登录日志追踪
  - 统计分析接口

- **权限审批流程**
  - 开发者写权限申请
  - 管理员审批机制
  - 24小时权限自动过期

### Fixed

- 角色验证失败问题（统一 viewer → global_viewer）
- 开发者无法访问 K8s Dashboard
- Dashboard 需要 token 登录
- 集群概览页面空白
- 统计接口路由配置错误

### Security

- 密码强度验证
- Token 过期管理
- 权限边界控制

## [0.1.0] - 2026-05-10

### Added

- 项目初始化
- 基础架构搭建
- 数据库设计
- API 接口设计文档