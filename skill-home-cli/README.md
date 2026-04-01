# skill-home CLI

`skill-home` 是 Skill Home 的本地命令行工具，负责把 skill 的创作、校验、发布、拉取、安装和多 IDE 同步串成一条统一工作流。

## 能力概览

| 能力域 | 代表命令 | 说明 |
|--------|----------|------|
| 本地创作 | `init` `create` `import` | 新建 skill、交互式生成模板、从 GitHub/Claude/Codex/Cursor 导入 |
| 质量检查 | `validate` `scan` `preview` | 校验 `SKILL.md`、做安全扫描、预览导出效果 |
| 打包与导出 | `pack` `export` | 生成 zip 发布包或导出为 IDE 平台格式 |
| IDE 同步 | `sync` `install` `uninstall` | 同步到 Claude、Copilot、Cursor、Codex、OpenClaw |
| 注册中心交互 | `push` `pull` `search` `list --remote` `info` | 发布、拉取、搜索和查看远程 skill |
| 生命周期管理 | `update` `delete` `delete-version` `rate` | 更新本地缓存、删除远程版本、评分 |
| 环境治理 | `login` `logout` `whoami` `doctor` `self-update` | 认证、诊断、本机 CLI 自更新 |

## 安装

### 使用已部署安装页

```bash
curl -fsSL https://soulstore.ciqtek.com/skill-home/install.sh -o /tmp/skill-home-install.sh
bash /tmp/skill-home-install.sh
```

指定版本：

```bash
bash /tmp/skill-home-install.sh v0.2.16
```

升级已安装 CLI：

```bash
skill-home self-update
skill-home self-update v0.2.16
```

安装脚本和 `self-update` 都从当前 Skill Home 服务托管的 `/releases` 读取版本元数据、校验文件和平台包。

### 从源码安装

```bash
git clone https://github.com/xianzhiqiyue/skill-manager.git
cd skill-manager/skill-home-cli
make install
```

## 典型工作流

### 1. 新建并校验一个本地 skill

```bash
skill-home init my-skill
cd my-skill

# 先补齐 SKILL.md 里的 category 和 official tags
skill-home validate .
skill-home scan .
skill-home preview . -p codex
```

### 2. 打包并发布到 Skill Home

```bash
skill-home login
# 交互终端下，push 会在缺少 category/tags 时提示补齐并写回 SKILL.md
skill-home pack .
skill-home push .
```

## 发布前分类元数据

每个发布到 Skill Home 的 skill 都必须具备：

- 1 个 `category`
- 1 到 4 个 `official tags`

`validate` 和 `push` 都会校验这两项。非交互环境下，如果缺失或不合法会直接失败；交互终端下，`push` 会尝试提示补齐并写回 `SKILL.md`。

### 3. 浏览远程公开目录并安装到 IDE

```bash
skill-home list --remote --format json
skill-home search codex
skill-home install @skill-home/skill-home-manager --ide codex --global --mode mirror
```

### 4. 刷新本地缓存 skill 到最新版本

```bash
skill-home update --ide codex --global --mode mirror
```

## 远程目录缓存

`list --remote` 与 `search` 已接入目录版本缓存，用来减少重复拉取整份公开目录：

- CLI 会先请求 `GET /api/v1/catalog/version`
- 如果 `catalog_version` 未变化，优先复用本地缓存目录
- 如果服务暂时失败但本地已有缓存，会回退到旧缓存，并在 `stderr` 提示“结果可能过期”
- 这层缓存只覆盖公开目录结构，不保证 `download_count`、`rating`、`rating_count` 这类动态统计字段实时

## 命令参考

| 命令 | 说明 |
|------|------|
| `skill-home init <name>` | 创建新技能模板（包含 `category/tags` 骨架） |
| `skill-home create [name]` | 交互式创建技能（会选择分类与官方标签） |
| `skill-home import <source>` | 从外部源导入技能 |
| `skill-home validate [path]` | 验证 `SKILL.md` 格式与官方分类元数据 |
| `skill-home scan [path]` | 扫描技能安全 |
| `skill-home preview [path] -p <platform>` | 预览导出效果 |
| `skill-home export [path] -p <platform>` | 导出到指定平台格式 |
| `skill-home pack [path]` | 打包技能 |
| `skill-home sync [path]` | 同步技能到 IDE |
| `skill-home list` | 列出本地已安装技能 |
| `skill-home list --remote` | 列出远程公开 skill |
| `skill-home search <keyword>` | 搜索远程公开 skill |
| `skill-home info <skill-ref>` | 查看技能详情与版本 |
| `skill-home pull <skill-ref>` | 从注册中心拉取技能 |
| `skill-home install <skill-ref>` | 拉取并同步到 IDE |
| `skill-home uninstall <skill-ref>` | 从本地 IDE 卸载技能 |
| `skill-home update` | 更新本地缓存技能到最新版 |
| `skill-home push [path]` | 推送技能到注册中心，必要时交互补齐分类元数据 |
| `skill-home delete <skill-ref>` | 删除远程技能 |
| `skill-home delete-version <skill-ref>` | 删除远程技能版本 |
| `skill-home rate <skill-ref> --score 5` | 为技能评分 |
| `skill-home activity` | 查看最近活动 |
| `skill-home doctor` | 诊断本地环境与 registry 配置 |
| `skill-home self-update [version]` | 更新当前 CLI 到最新或指定版本 |
| `skill-home version` | 显示 CLI 版本 |

技能引用格式：

- `my-skill`：默认命名空间，最新版本
- `@user/my-skill`：指定命名空间
- `my-skill@1.0.0`：指定版本
- `@user/my-skill@1.0.0`：完整格式

## 注册中心认证边界

- `push`、`delete`、`delete-version`、`rate`、`activity`、`whoami` 需要先执行 `skill-home login`
- `pull`、`install`、`update`、`search`、`info`、`list --remote` 对公开 skill 不需要登录
- 访问私有 skill 时，CLI 会提示先执行 `skill-home login`

## 登录工作流

### 邮箱/密码登录

```bash
skill-home login
```

CLI 会登录 Skill Home 服务，并自动创建一把 CLI 可复用的 API Key 保存到本地配置。

### 直接使用 API Key

```bash
skill-home login --api-key "sk_xxx"
```

或者：

```bash
export SKILL_HOME_API_KEY="sk_xxx"
```

## 配置

配置文件位于 `~/.config/skill-home/config.yaml`：

```yaml
registry:
  endpoint: "https://soulstore.ciqtek.com/skill-home"
  api_key: "your-api-key"

ide:
  claude:
    enabled: true
    project_path: ".claude/skills"
    global_path: "~/.claude/skills"
  copilot:
    enabled: false
    project_path: ".github/skills"
    global_path: "~/.copilot/skills"
  cursor:
    enabled: true
    project_path: ".cursor/rules"
  codex:
    enabled: true
    project_path: ".agents/skills"
    global_path: "~/.agents/skills"
  openclaw:
    enabled: true
    project_path: "skills"
    global_path: "~/.openclaw/skills"

sync:
  mode: "auto"
  conflict_strategy: "project_wins"
```

如果你希望 Codex 全局目录落到 `~/.codex/skills`，可以显式改写 `ide.codex.global_path`。

## 开发

```bash
# 安装依赖
make deps

# 构建
make build

# 运行测试
make test

# 代码格式化
make fmt

# 跨平台构建
make build-all
```

## 环境变量

| 变量名 | 说明 |
|--------|------|
| `SKILL_HOME_API_KEY` | API Key，优先于配置文件 |
| `SKILL_HOME_CONFIG` | 配置文件路径 |
| `SKILL_HOME_RELEASES_BASE_URL` | 覆盖 CLI 自更新下载源 |

## 调试模式

```bash
skill-home --debug sync
skill-home --debug search codex
```

## 项目结构

```
skill-home-cli/
├── cmd/skill-home/         # 程序入口
├── internal/
│   ├── cmd/                # CLI 命令实现
│   ├── config/             # 配置管理
│   ├── errors/             # 错误处理
│   ├── import/             # 技能导入器
│   ├── logger/             # 日志系统
│   ├── registry/           # 注册中心客户端
│   ├── skill/              # 技能解析
│   ├── ide/                # IDE 适配器
│   └── sync/               # 同步引擎
├── pkg/
│   ├── archive/            # 归档工具
│   └── validator/          # 安全扫描器
└── Makefile
```

## 许可证

MIT License
