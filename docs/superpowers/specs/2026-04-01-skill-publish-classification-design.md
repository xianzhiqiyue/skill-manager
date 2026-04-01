# Skill Home 技能发布分类与标签强制化设计

## 1. 目标

为 Skill Home 建立一套稳定、可执行、可被代理工作流消费的“官方分类元数据”机制，确保每个发布到注册中心的 skill 都具备最基本的可发现性信息。

本次设计要同时解决两个问题：

- 从产品和目录角度，保证每个 skill 至少有一个明确的主分类和一组官方标签
- 从 OpenClaw 代理体验角度，让 `skill-home-manager` 能在推送前主动帮用户补齐这些元数据，而不是把失败留到发布最后一步

期望的用户可见结果是：

- 新建 skill 时就能看到 `category` 和 `tags` 这两类正式元数据
- 使用 `skill-home-manager` 让 OpenClaw 帮忙发布 skill 时，代理会先补齐元数据，再执行 `validate/pack/push`
- 手工调用 CLI 时，交互模式可被引导补齐；非交互模式缺失则直接失败
- 注册中心不再接受缺少官方分类元数据的新增 skill

本设计聚焦“让每个 skill 至少拥有合格的官方分类信息”，不包含复杂的推荐算法、自动审核后台或社区标签治理系统。

## 2. 背景与现状

当前仓库已经具备部分相关能力，但没有形成闭环：

- `SKILL.md` 解析层已经支持 `tags`
- server 的 skill 创建与更新接口已经支持 `tags`，并会做归一化
- Web 发布页已经有 `Tags` 输入框
- 社区标签体系已经独立存在，适合发布后补充语义

但当前链路仍然有三个明显缺口：

### 2.1 缺少主分类

当前 skill 数据模型只有 `tags`，没有单值 `category`。这会导致：

- 浏览入口不稳定
- 目录结构只能依赖散乱标签
- 搜索和筛选难以建立清晰的一级导航

### 2.2 CLI `push` 没有真正上传 manifest tags

本地 manifest 可以解析 `tags`，但 CLI `PublishRequest` 目前没有把它们发送到服务端。这意味着：

- 手工 `push` 和 Web 发布页行为不一致
- 通过 OpenClaw 调 `skill-home push` 时，即使本地写了 `tags`，也可能不会进入注册中心

### 2.3 `skill-home-manager` 没有把分类元数据纳入默认发布流程

当前 `skill-home-manager` 更强调 `validate/pack/push`，但没有把“发布前必须补齐 `category/tags`”作为默认前置动作。结果是：

- OpenClaw 容易沿着旧心智直接发布
- 新规则一旦只在 CLI/server 侧落地，代理体验会变成“先失败，再补齐”

## 3. 核心设计决策

### 3.1 官方分类信息采用双层结构

每个 skill 都必须具备：

- 1 个必填 `category`
- 1 到 4 个必填 `official tags`

两者职责明确分工：

- `category` 回答“这个 skill 大体属于哪一类”
- `official tags` 回答“它最典型的几个使用场景或能力点是什么”

### 3.2 `SKILL.md` 是官方分类信息的唯一来源

`category` 和 `tags` 都定义在 `SKILL.md` frontmatter 中，而不是只存在于一次发布表单或一次 CLI 请求里。

这样做的原因是：

- skill 的分类属于 skill 定义本身，而不是某次发布的临时附注
- 代码审阅、版本管理、导入导出、IDE 同步都可以共享同一份元数据
- OpenClaw 可以直接围绕 `SKILL.md` 做检查、补齐和解释

### 3.3 社区标签继续与官方标签分层

本设计不改变既有的社区标签定位：

- `official tags` 由作者在发布时声明，承担目录和筛选职责
- `community tags` 由用户在发布后补充，承担社区语义共建职责

两者不混用，也不共享发布约束。

### 3.4 约束采用“三层递进”

为了既保证体验顺手，又真正做到不漏填，约束分三层实施：

- `skill-home-manager`：代理侧预检查与补齐，尽量减少撞墙
- CLI / Web / server：产品与接口层硬校验
- 数据库：存量迁移完成后的最终兜底约束

也就是说，交互层负责“好填”，服务端负责“卡住”，数据库负责“谁也绕不过”。

## 4. Taxonomy 初稿

### 4.1 一级分类

第一版 `category` 先控制在 8 个：

- `development`
- `testing`
- `docs`
- `automation`
- `integration`
- `ops`
- `research`
- `productivity`

它们的使用意图如下：

- `development`：代码生成、重构、调试、工程辅助
- `testing`：测试、验证、评审、质量保障
- `docs`：文档编写、知识整理、说明生成
- `automation`：流程编排、重复任务自动化
- `integration`：外部平台、API、第三方工具接入
- `ops`：部署、发布、运维、环境管理
- `research`：检索、分析、调研、总结
- `productivity`：协作推进、组织管理、个人效率

### 4.2 官方标签

第一版 `official tags` 建议从受控词表中选择，初稿如下：

- `codegen`
- `refactor`
- `debug`
- `testing`
- `review`
- `docs`
- `search`
- `analysis`
- `planning`
- `git`
- `ci-cd`
- `deployment`
- `monitoring`
- `security`
- `database`
- `frontend`
- `backend`
- `api`
- `mcp`
- `prompting`
- `automation`
- `workflow`
- `integration`

### 4.3 规范与别名

第一版约束：

- 全部使用小写 ASCII
- 优先 `kebab-case`
- 单个 tag 长度不超过 64
- 每个 skill 最少 1 个，最多 4 个

服务端负责做轻量别名折叠，例如：

- `ci`、`pipeline` 统一到 `ci-cd`
- `deploy` 统一到 `deployment`
- `doc` 统一到 `docs`

## 5. 单一真相来源

为了避免 CLI、server、Web、`skill-home-manager` 各维护一套不同词表，本次设计新增一份仓库级 taxonomy 配置，作为唯一真相来源。

建议位置：

- `config/skill-taxonomy.json`

该文件至少包含：

- `categories`
- `official_tags`
- `aliases`
- 每个分类与标签的简短说明

消费方式如下：

- CLI 读取该配置用于 `create/init/validate/push`
- server 读取该配置用于创建、更新校验与归一化
- Web 发布页用该配置渲染下拉选项和标签选择器
- `skills/skill-home-manager` 从该配置生成一份随 skill 一起发布的参考文件，例如 `references/publish-taxonomy.md`

这样 OpenClaw 在离开源码仓库、只拿到 skill 包时，仍然能读取同一套 taxonomy 描述，而不是依赖文档记忆。

## 6. 交互设计

### 6.1 新建 skill

`skill-home init` 与 `skill-home create` 需要把 `category` 与 `tags` 直接纳入模板和创建向导。

目标体验是：

- 新 skill 从第一天开始就知道自己需要正式分类元数据
- 作者不是在发布时临时想标签，而是在定义 skill 时顺手补齐

模板策略：

- `SKILL.md` 模板直接包含 `category:` 与 `tags:` 骨架
- 默认占位文案明确说明“发布前必须补齐”

### 6.2 CLI `push`

`skill-home push` 的行为分两种情况：

- 交互终端：
  - 先解析 `SKILL.md`
  - 如果缺失 `category` 或 `tags`，进入简短补齐向导
  - 用户确认后把结果写回 `SKILL.md`
  - 再继续 `validate/pack/push`
- 非交互终端：
  - 不进行提问
  - 缺失或非法时直接失败，并给出明确错误

这个设计保证：

- 代理和人工终端都能被引导完成
- CI、自动脚本和无头环境保持可预测失败

### 6.3 Web 发布页

Web 发布页需要同步升级为正式的官方分类录入界面：

- `category` 改为必填单选
- `tags` 改为受控多选，限制 1 到 4 个
- 未满足要求时提交按钮禁用
- 错误提示在表单内就地展示

第一版不要求 Web 提供自由输入官方标签，避免 taxonomy 失控。

### 6.4 `skill-home-manager` 代理行为

当用户通过 `skill-home-manager` 让 OpenClaw 帮忙创建、编辑或发布 skill 时，默认工作流改成：

1. 检查 `SKILL.md` 是否存在 `category` 和合格的 `tags`
2. 若缺失，则基于 `name`、`description`、正文和目录内容生成推荐值
3. 如果判断明确，代理直接补齐 `SKILL.md`
4. 如果判断存在歧义，只提一个短问题澄清主分类或核心场景
5. 元数据齐全后，继续执行 `validate/pack/push`

这使代理体验从“先撞校验墙，再返工”变成“先整理元数据，再发布”。

## 7. 强制化策略

### 7.1 CLI 校验

`skill-home validate` 需要新增规则：

- 缺少 `category` 失败
- `category` 不在官方 taxonomy 中失败
- `tags` 数量不在 1 到 4 之间失败
- 任一 tag 不在官方 taxonomy 中失败

### 7.2 发布接口校验

server 创建与更新 skill 时统一执行同样的硬校验：

- `category` 必填
- `tags` 必填
- taxonomy 非法值直接返回 `INVALID_INPUT`

这样即使有其他客户端绕过 CLI，也不能把非法 skill 写入注册中心。

### 7.3 数据库约束

数据库层约束不在第一天立即加死，而采用分阶段收紧：

- 第一阶段：代码层强校验，数据库允许旧数据继续存在
- 第二阶段：存量数据基本补齐后，增加 `category NOT NULL`
- 第三阶段：为 `tags` 增加最小长度约束或等效检查

## 8. 数据模型与协议变更

### 8.1 `SKILL.md`

frontmatter 新增：

- `category: ops`
- `tags: [deployment, ci-cd, automation]`

### 8.2 CLI

需要调整：

- manifest 结构增加 `Category`
- `init/create` 生成或采集 `category`
- `validate` 校验 taxonomy
- `push` 上传 `category` 与 `tags`
- 注册中心客户端请求结构补齐对应字段

### 8.3 server

需要调整：

- `Skill` 模型增加 `Category`
- 创建与更新请求结构增加 `category`
- 列表、详情、搜索结果返回 `category`
- taxonomy 校验与别名归一化集中在服务端帮助函数中

### 8.4 Web

需要调整：

- 发布页表单结构增加 `category`
- `tags` 从自由文本升级为受控选择
- 对象页与列表页可展示 `category`

## 9. 存量 skill 迁移

为了避免一次性卡死现有生态，本次采用两阶段迁移：

### 9.1 新增 skill 立即强制

从功能上线开始：

- 新建 skill 必须有 `category` 与 `tags`
- 新 namespace 下首次发布直接受新规则约束

### 9.2 存量 skill 延迟强制

已存在 skill 的处理原则是：

- 旧数据可以短期保留
- 一旦 owner 再次发布新版本，就必须先补齐元数据
- 平台可以在列表或后台标记 `metadata incomplete`

必要时可以准备一轮 backfill：

- 按现有 tags、描述和正文自动推荐 `category`
- 由 owner 在下次发布或编辑时确认

## 10. 错误处理与文案

错误文案应尽量指向动作，而不是只给抽象失败原因。

示例：

- `缺少 category，请在 SKILL.md 中设置 category`
- `tags 至少需要 1 个官方标签`
- `tag "pipeline" 不在官方词表中，可改用 "ci-cd"`
- `当前为非交互环境，无法自动补齐分类信息，请先更新 SKILL.md`

`skill-home-manager` 在代理模式下不应只复述错误，而应优先执行补齐或给出明确下一步。

## 11. 测试与验收

至少需要覆盖以下场景：

- `init/create` 生成的新模板包含 `category/tags` 骨架
- `validate` 对缺失、非法、超量 tag 的失败行为
- `push` 在交互终端的补齐分支
- `push` 在非交互环境下的直接失败分支
- server 创建、更新接口的 taxonomy 校验
- Web 发布页未填分类时不可提交
- `skill-home-manager` 文档和默认工作流明确要求先补齐分类元数据

## 12. 本阶段不包含

本次设计明确不包含：

- 社区标签治理后台
- 自动机器分类后直接无确认发布
- 基于分类的推荐算法
- 复杂多级 taxonomy
- 让社区标签进入发布门槛

## 13. 推荐实施顺序

推荐按以下顺序落地：

1. 固化 taxonomy 文件与字段规范
2. 补齐 CLI manifest、`validate`、`push`
3. 补 server 数据模型与接口校验
4. 升级 Web 发布页
5. 更新 `skills/skill-home-manager`
6. 处理存量 skill 迁移与最终数据库约束

这个顺序的好处是：

- 先建立单一真相来源
- 再让所有入口围绕同一约束收敛
- 最后再收数据库口子，避免上线初期直接打断旧数据
