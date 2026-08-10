# Skill Home 固定分类规范

## 结论

Skill Home 使用下面 12 个固定一级分类。创建、编辑、导入和发布 Skill 时，`category` 直接填写中文分类值，不接受自定义分类。

| 中文分类值 | 旧英文值 | 主要交付结果 | 典型例子 |
| --- | --- | --- | --- |
| `开发与编程` | `development` | 编写、重构和调试代码，构建网站、应用或 Agent | 前端开发、后端开发、代码重构 |
| `测试与质量` | `testing` | 测试、评审、质量检查、故障排查与合规审计 | E2E 测试、FMEA、代码评审 |
| `数据与分析` | `data` | 处理表格和数据库，完成统计分析、可视化与数据报告 | 数据清洗、Excel 分析、图表生成 |
| `搜索与研究` | `research` | 检索资料、开展调研，整理知识、论文、市场与行业情报 | 网页研究、论文追踪、知识库检索 |
| `文档与办公` | `docs` | 创建、编辑、翻译和排版文档、表格、演示文稿与 PDF | Word 排版、PDF 翻译、PPT 制作 |
| `设计与内容` | `creative` | 完成 UI、视觉、生图、写作、音视频与社交媒体内容创作 | UI 设计、生图、文章配图 |
| `业务与管理` | `business` | 支持销售、营销、财务、法务、人力、项目、生产与客户服务 | CRM、招聘、合同审核、订单管理 |
| `效率与协作` | `productivity` | 管理任务、笔记、会议、知识以及个人与团队协作 | 待办、会议纪要、Obsidian |
| `自动化与工作流` | `automation` | 执行浏览器、文件、定时任务和跨步骤重复流程 | 浏览器自动化、批处理、定时任务 |
| `平台与连接` | `integration` | 通过 API、MCP 或客户端连接通用外部平台和企业系统 | 钉钉连接器、邮箱客户端、通用 API 封装 |
| `运维与安全` | `ops` | 部署发布，管理服务器、环境、监控、权限与安全 | 服务器管理、CI/CD、环境诊断 |
| `Agent 与 Skill 工具` | `meta` | 创建、安装、管理或改进 Agent、Skill、提示词与上下文 | Skill 安装器、Skill 创建器、Agent 编排 |

## 选择规则

1. 先看用户最终拿到什么结果，再选一级分类。不要因为 Skill 调用了 API，就一律放进 `平台与连接`。
2. 面向明确业务部门或业务流程的 Skill，优先选择 `业务与管理`。例如“通过 CRM API 审核合同”的最终结果是业务审核，应选 `业务与管理`；只有可被多种场景复用的 CRM 通用连接器才选 `平台与连接`。
3. `自动化与工作流` 只表示跨场景的通用自动化能力。自动生成 Word 报告仍选 `文档与办公`，自动抓取论文并总结仍选 `搜索与研究`。
4. 每个 Skill 只选一个一级分类。API、MCP、自动化、数据库、前端等实现方式继续放在 1 到 4 个官方标签中。
5. 判断仍有歧义时，以 Skill 描述中的主要触发场景和主要输出作为依据。

## 为什么从 8 类扩展到 12 类

2026-08-08 使用 `skill-home list --remote --format json` 查询线上目录，共返回 95 个 Skill：83 个已有分类，12 个旧 Skill 仍为空分类。已有分类中 `productivity` 25 个、`integration` 20 个、`research` 16 个，而 `development` 和 `testing` 各只有 2 个。官方标签中 `automation` 出现 46 次、`analysis` 37 次、`workflow` 36 次、`integration` 35 次。

实际内容显示，生图和设计、CRM 和招聘、合同和订单、表格和数据分析等不同结果被集中放进原 `productivity` 或 `integration`。这会让用户点进一个分类后看到彼此无关的 Skill。因此固定为 12 个中文分类，并保留旧英文值的受控兼容映射：

- `数据与分析`：承接数据处理、统计分析与可视化。
- `设计与内容`：承接设计、生图和内容创作。
- `业务与管理`：承接销售、营销、财务、法务、HR 和企业业务流程。
- `Agent 与 Skill 工具`：承接 Agent 与 Skill 自身的创建、安装和治理工具。

同类产品也普遍同时提供技术类与业务结果类入口：[skills-hub.ai](https://skills-hub.ai/categories) 将工程能力与 Marketing、Creative、Data、Business、Research、Education 等领域分开；[SkillHub Marketplace](https://useskillhub.com/marketplace) 使用 Content、Research、Sales/CRM、Data/Sheets、Finance、HR、Operations、Automation、Development 等任务分类；[Anthropic Skills](https://github.com/anthropics/skills) 的公开集合区分 Creative & Design、Development & Technical、Enterprise & Communication 和 Document Skills。

## 旧数据兼容与迁移建议

旧英文值会由代码归一化为对应中文值，数据库迁移也会把已有英文分类改成中文。12 条空分类旧数据仍需要在下次编辑或发布时从固定列表选择；本次代码变更不直接修改线上数据。

| 旧 Skill | 建议分类 |
| --- | --- |
| `xiaohongshu-explore` | `业务与管理` |
| `designer-studio` | `设计与内容` |
| `business-system-skill` | `开发与编程` |
| `content-radar` | `搜索与研究` |
| `fastgpt-dataset-api` | `平台与连接` |
| `headhunter-search-report` | `业务与管理` |
| `headhunter-find-job` | `业务与管理` |
| `order-process-specialist` | `业务与管理` |
| `order-progress-analyze` | `业务与管理` |
| `app-development-skill` | `开发与编程` |
| `bom-splitter-api` | `业务与管理` |
| `ciqtek-services` | `平台与连接` |

迁移前应由 Skill owner 结合实际触发场景确认；以上映射是基于线上名称和描述的建议，不代表已经写入线上。

## 代码约束

- 唯一事实源：`config/skill-taxonomy.json`
- 生成脚本：`scripts/generate_skill_taxonomy.py`
- CLI：创建时只展示固定列表，`validate` 和 `push` 拒绝列表外分类。
- Web：发布页和设置页使用下拉选择，表单校验拒绝列表外分类。
- Server：创建和更新接口使用同一词表做硬校验，旧英文值会归一化为中文，绕过页面直接请求也不能写入自定义分类。

更新分类时必须先修改唯一事实源，再运行生成脚本并提交所有生成产物。不得在 CLI、Web 或 Server 中单独维护另一份分类列表。
