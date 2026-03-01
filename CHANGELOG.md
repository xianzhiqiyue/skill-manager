# Changelog

所有显著变更都会记录在此文件。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [Unreleased]

## [1.0.0] - 2026-03-01

### Added

#### 服务端 (skill-home-server)
- 用户认证系统 (JWT + 密码登录)
- API Key 管理 (生成、撤销)
- 技能 CRUD 操作
- 技能版本管理
- 技能搜索功能
- 对象存储集成 (MinIO)
- 安全扫描 (基础规则)
- 数据库模型与迁移

#### CLI (skill-home-cli)
- 技能初始化命令
- 技能格式验证
- 安全扫描功能
- 技能打包
- 注册中心客户端
- 技能搜索

#### 部署
- Docker Compose 配置
- 生产环境部署脚本
- Systemd 服务配置
- 防火墙规则配置

#### 文档
- API 文档
- 部署文档
- 数据库设计文档
- 贡献指南

### Fixed

- JSON 字段类型扫描问题
- PostgreSQL 数组类型扫描
- URL 查询参数编码问题
- API Key 生成安全问题

### Security

- 数据库端口仅限本地访问
- Redis 端口仅限本地访问
- MinIO 默认密码已更新
- API Key 使用安全随机数生成

## [0.1.0] - 2026-02-28

### Added

- 项目初始架构
- 基础目录结构
- 技术规格文档
- 需求梳理文档

---

## 版本说明

### 版本号格式

`主版本号.次版本号.修订号`

- **主版本号**: 不兼容的 API 变更
- **次版本号**: 向下兼容的功能新增
- **修订号**: 向下兼容的问题修复

### 变更类型

- `Added`: 新功能
- `Changed`: 现有功能的变更
- `Deprecated`: 即将移除的功能
- `Removed`: 已移除的功能
- `Fixed`: 问题修复
- `Security`: 安全相关的修复
