# Skill 发布分类与标签词表

发布到 Skill Home 时，每个 skill 都需要：

- 1 个 `category`
- 1 到 4 个 `official tags`

## 一级分类

- `开发与编程`: 编写、重构和调试代码，构建网站、应用或 Agent
- `测试与质量`: 测试、评审、质量检查、故障排查与合规审计
- `数据与分析`: 处理表格和数据库，完成统计分析、可视化与数据报告
- `搜索与研究`: 检索资料、开展调研，整理知识、论文、市场与行业情报
- `文档与办公`: 创建、编辑、翻译和排版文档、表格、演示文稿与 PDF
- `设计与内容`: 完成 UI、视觉、生图、写作、音视频与社交媒体内容创作
- `业务与管理`: 支持销售、营销、财务、法务、人力、项目、生产与客户服务
- `效率与协作`: 管理任务、笔记、会议、知识以及个人与团队协作
- `自动化与工作流`: 执行浏览器、文件、定时任务和跨步骤重复流程
- `平台与连接`: 通过 API、MCP 或客户端连接通用外部平台和企业系统
- `运维与安全`: 部署发布，管理服务器、环境、监控、权限与安全
- `Agent 与 Skill 工具`: 创建、安装、管理或改进 Agent、Skill、提示词与上下文

## 旧英文分类兼容

- `development` -> `开发与编程`
- `testing` -> `测试与质量`
- `data` -> `数据与分析`
- `research` -> `搜索与研究`
- `docs` -> `文档与办公`
- `creative` -> `设计与内容`
- `business` -> `业务与管理`
- `productivity` -> `效率与协作`
- `automation` -> `自动化与工作流`
- `integration` -> `平台与连接`
- `ops` -> `运维与安全`
- `meta` -> `Agent 与 Skill 工具`

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

- 分类只能从上面的固定列表中选择，不接受自定义分类。
- 一级分类按 skill 的主要交付结果选择，而不是按调用的技术或平台选择。
- 面向特定业务结果的 skill 优先选择 `业务与管理`；只有通用连接器才选择 `平台与连接`。
- `自动化与工作流` 只用于跨场景的通用自动化能力；文档、研究或业务流程仍选择对应结果分类。
- 再从官方标签里挑 1 到 4 个最典型的场景词；API、MCP、自动化等实现方式放在标签中。
- 优先使用受控词表，不要临时发明新的官方标签。

## 示例

- `deploy-buddy`: `category: 运维与安全`，`tags: [deployment, ci-cd, automation]`
- `doc-helper`: `category: 文档与办公`，`tags: [docs, analysis, workflow]`
- `crm-contract-audit`: `category: 业务与管理`，`tags: [analysis, api, workflow]`
- `image-generator`: `category: 设计与内容`，`tags: [automation, integration]`
