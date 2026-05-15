# K8sGate - K8s 集群登录与审计系统

轻量级 K8s 集群登录认证、权限控制和操作审计网关。作为现有 Kubernetes Dashboard 的前置代理层，提供统一登录、分级权限和操作审计能力。

## 功能

- 账号密码登录，强制定期更换密码
- 三级权限控制：Viewer / Developer / Admin
- 项目与命名空间多对多映射
- 操作审计日志自动记录
- 个人操作报告与导出

## 技术栈

- 后端：Go 1.22+ (Gin + GORM + client-go)
- 前端：Vue 3 + Element Plus
- 数据库：MySQL 5.7
- 部署：K8s + Istio

## 快速开始

详见 [docs/](docs/) 目录下的设计文档。
