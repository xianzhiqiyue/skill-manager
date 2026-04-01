# Skill 发布分类与标签词表

发布到 Skill Home 时，每个 skill 都需要：

- 1 个 `category`
- 1 到 4 个 `official tags`

## 一级分类

- `development`: 代码生成、重构、调试、工程辅助
- `testing`: 测试、验证、评审、质量保障
- `docs`: 文档编写、知识整理、说明生成
- `automation`: 流程编排、重复任务自动化
- `integration`: 外部平台、API、第三方工具接入
- `ops`: 部署、发布、运维、环境管理
- `research`: 检索、分析、调研、总结
- `productivity`: 协作推进、组织管理、个人效率

## 官方标签

- `codegen`: 代码生成
- `refactor`: 重构与结构调整
- `debug`: 调试与问题定位
- `testing`: 测试设计与执行
- `review`: 评审与审查
- `docs`: 文档写作与整理
- `search`: 检索与查找
- `analysis`: 分析与总结
- `planning`: 规划与方案输出
- `git`: Git 工作流
- `ci-cd`: CI/CD 与 pipeline 场景
- `deployment`: 部署与发布
- `monitoring`: 观测、告警与监控
- `security`: 安全与合规
- `database`: 数据库与存储
- `frontend`: 前端与界面实现
- `backend`: 后端与服务实现
- `api`: API 设计与集成
- `mcp`: MCP 与工具连接
- `prompting`: 提示词与代理编排
- `automation`: 自动化流程
- `workflow`: 工作流组织
- `integration`: 第三方集成

## 别名归一化

- `ci` -> `ci-cd`
- `deploy` -> `deployment`
- `doc` -> `docs`
- `documentation` -> `docs`
- `pipeline` -> `ci-cd`

## 使用建议

- 先判断 skill 的一级能力域，填写 `category`。
- 再从官方标签里挑 1 到 4 个最典型的场景词。
- 优先使用受控词表，不要临时发明新的官方标签。

## 示例

- `deploy-buddy`: `category: ops`，`tags: [deployment, ci-cd, automation]`
- `doc-helper`: `category: docs`，`tags: [docs, analysis, workflow]`
