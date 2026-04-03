---
name: skill-home-manager
version: 0.2.10
description: 当用户想用本地 skill-home CLI 创建、编辑、验证、打包、导出、同步、安装或排查 skill 时使用，尤其适合把 skill 交付到 Codex。
category: productivity
namespace: "@skill-home"
author: Skill Home Team
license: MIT
tags:
  - workflow
  - automation
  - docs
  - integration
homepage: https://github.com/skill-home/skill-manager
ide_config:
  codex:
    globs: ["**/*"]
    auto_activate: false
    tools: [read, edit, bash, glob, grep]
---

# Skill Home 管理指南

用这个 skill 操作本机已安装的 `skill-home` CLI；缺 CLI 时优先从公开发布入口安装或刷新，把“管理 skill”的能力封装成 Codex 可重复执行的工作流。

## 触发场景

- 用户要在当前仓库里新建或修改一个 skill 子项目
- 用户要运行 `skill-home create/init/validate/scan/pack/preview/export/sync/install/doctor/delete/delete-version`
- 用户要把本地 skill 安装到 Codex
- 用户要把本地 skill 发布到 Skill Home 或更新已发布 skill
- 用户怀疑本机 `skill-home` 二进制过旧，需要重新安装或刷新已发布 CLI
- 用户要删除自己已发布的远程 skill 或某个远程版本

## 默认上下文

- 当前工作目录: 以用户当前任务目录为准，不默认假设存在特定仓库
- 本机配置文件: `~/.config/skill-home/config.yaml`
- 公共安装入口: `https://soulstore.ciqtek.com/skill-home/install.sh`
- GitHub Releases: `https://github.com/xianzhiqiyue/skill-manager/releases`
- Codex 全局 skills 目录不能写死。优先按宿主环境探测: `SKILL_HOME_CODEX_SKILLS_DIR` -> `skill-home` 配置里的 `ide.codex.global_path` -> `$CODEX_HOME/skills` -> 已存在的 `~/.agents/skills` 或 `~/.codex/skills`
- 运行 bundled scripts 之前，先按宿主环境解析 `skill_home_manager_root`。优先顺序是: `SKILL_HOME_MANAGER_ROOT` -> `skill-home` 配置里的 `ide.codex.global_path/skill-home-manager` -> `$CODEX_HOME/skills/skill-home-manager` -> 已安装位置里的 `~/.agents/skills/skill-home-manager` 或 `~/.codex/skills/skill-home-manager`

只有在用户明确说明自己正在 `skill-manager` 仓库里开发时，才把该仓库视作额外上下文；不要默认要求本机存在任何 `skill-home` 源码目录。
不要把 bundled scripts 路径写死成任何开发机仓库绝对路径，例如 `/home/zhuyue/code/skill-manager/...`。

## 默认工作流

1. 先查看目标 skill 目录、现有示例和 CLI 支持的真实命令，不靠猜测写操作流。
2. 先确保本机真的有可用的 `skill-home` CLI。缺 CLI 时，优先跑:
   `bash "$skill_home_manager_root/scripts/bootstrap-cli.sh"`
3. 对高频动作，优先跑 bundled scripts，而不是手写一长串命令。
4. 只要涉及 `validate`、`pack`、`push` 或更新远程 skill，先检查 `SKILL.md` 里的 `category` 和 `tags` 是否齐全且属于官方 taxonomy。
5. 如果缺失或不合法，优先根据 skill 名称、描述和正文内容推断推荐值，并参考 `references/publish-taxonomy.md` 补齐后写回 `SKILL.md`。
6. 如果分类存在歧义，只问一个短问题澄清主分类或核心场景；不要把一串 taxonomy 问题丢给用户。
7. 任何涉及“安装到 Codex”或“打包输出位置”的动作，都先让 agent 根据配置、环境变量、现有目录和当前工作目录判断真实路径；不要把 `~/.codex/skills`、`~/.agents/skills` 或开发机仓库绝对路径当成默认真值。
8. 新建本地 skill:
   `bash "$skill_home_manager_root/scripts/create-local-skill.sh" <name> [description] [output-dir]`
9. 把本地 skill 安装到 Codex:
   `bash "$skill_home_manager_root/scripts/install-to-codex.sh" <skill-path>`
10. 如果 CLI 版本过旧、缺子命令，或行为不像最新 release，先刷新已发布 CLI 再继续:
   `bash "$skill_home_manager_root/scripts/rebuild-cli.sh"`
11. 如果问题和路径、配置、注册中心或 API Key 有关，先跑 `skill-home doctor`。
12. 涉及仓库发布时，先遵循仓库统一发布口径：先判断影响面，再更新版本、提交、推远端，最后发布制品；如果仓库文档与当前 skill 约束冲突，以仓库里的 `docs/release-process.md` 为准。

## 命令选择规则

- 本地新建 skill: `skill-home init`
- 交互式创建 skill: `skill-home create`
- 格式校验: `skill-home validate`
- 安全扫描: `skill-home scan`
- 打包发布产物: `skill-home pack`
- 预览平台导出结果: `skill-home preview`
- 生成平台格式或直接落盘: `skill-home export`
- 把本地目录同步进 IDE: `skill-home sync`
- 浏览远程公开目录: `skill-home list --remote`
- 搜索远程公开目录: `skill-home search`
- 从 registry 安装远程 skill 引用: `skill-home install`
- 删除远程 skill: `skill-home delete`
- 删除远程 skill 版本: `skill-home delete-version`
- 排查环境与配置: `skill-home doctor`

`install` 适合远程 skill 引用，不适合本地 skill 目录。对本地 skill 目录，优先用 `sync` 或 `export --install`。
`pack` 默认输出到当前工作目录；如果用户没指定 `--output`，agent 应先根据当前 skill 所在目录和后续用途判断是否需要显式给出输出路径，避免把产物落到错误位置后再猜。
`push`、`delete`、`delete-version` 需要先登录；公开 skill 的 `pull/install/update/search/info/list --remote` 不需要登录。
`validate` 现在会把 `category` 和 `official tags` 一起作为硬校验；不要把刚 `init` 出来的空骨架误判成“已经可发布”。
`push` 在交互终端里会尝试补齐缺失的官方分类元数据，但默认仍应由代理先整理好 `SKILL.md`，再进入发布动作。
`list --remote` 与 `search` 会先检查远程目录版本；版本未变化时优先复用本地目录缓存。如果注册中心临时失败且本地已有对应缓存，会自动回退并提示“结果可能过期”。
这个目录缓存只覆盖公开目录结构，不保证 `download_count`、`rating`、`rating_count` 这类动态统计字段实时。对 `search` 来说，缓存结果只保证来自同一份公开目录快照，不保证与当前服务端搜索排序或召回逻辑完全一致。

## 用户请求到动作的映射

- “帮我把 skill-home CLI 装上 / 补齐本机环境”: 先用 `scripts/bootstrap-cli.sh`
- “帮我新建一个 skill 子项目”: 先用 `scripts/create-local-skill.sh`
- “帮我发布一个 skill”: 先补齐 `category/tags`，再跑 `validate -> pack -> push`
- “帮我统一发布流程 / 规范发布口径”: 优先更新仓库里的 `docs/release-process.md`，再同步 `README.md`、`DEPLOYMENT.md` 和相关 skill 文档
- “帮我把这个 skill 装到 Codex”: 先用 `scripts/install-to-codex.sh`
- “帮我把本地 skill-home 更新到最新发布版本”: 先用 `scripts/rebuild-cli.sh`
- “帮我排查为什么没生效”: 先看 `skill-home doctor`，必要时再开 `--debug` 重跑相关命令
- “帮我看看远程有哪些公开 skill”: 用 `skill-home list --remote`
- “帮我搜索远程公开 skill”: 用 `skill-home search <keyword>`
- “帮我删除我刚发布的 skill”: 用 `skill-home delete @namespace/name`
- “帮我删除我刚发布的某个版本”: 用 `skill-home delete-version @namespace/name@version`

## Codex 特别约束

- 这台机器是 WSL + Windows 混合环境，向 Codex 安装 skill 时优先 `--mode mirror`，避免生成指向 Linux 路径的符号链接。
- `skill_home_manager_root` 必须来自当前宿主环境里的已安装 skill 根目录，不能默认继承开发机仓库路径。
- 无论安装还是打包，都不要把目标路径写死给用户或写死在代理动作里；先探测配置与环境，再决定具体路径。
- `bash "$skill_home_manager_root/scripts/bootstrap-cli.sh"` 默认从已部署安装页拉取公开安装脚本；如果部署页不可用，再回退到 GitHub 上的安装脚本。它安装的是已发布的 CLI，不依赖本地源码目录。
- `bash "$skill_home_manager_root/scripts/rebuild-cli.sh"` 在这个公共 skill 里表示“重新安装最新发布版 CLI”，不是从源码重建。
- 不要默认要求本机存在任何 `skill-home` 源码仓库；只有当用户明确在维护该仓库时，才允许走源码工作流。
- 涉及 registry 写操作时，先检查是否已登录；未登录时提示用户先执行 `skill-home login`。
- 遇到 skill 发布时，默认要把“补齐官方分类元数据”视为发布前置步骤，而不是等 `push` 失败后再补救。
- 如果可以明确推断 `category/tags`，直接写回 `SKILL.md`；如果存在歧义，只问一个短问题澄清主分类或核心场景。
- skill 发布的默认顺序是：递增版本号 -> `validate` -> `scan` -> `pack` -> `push`，发布成功后再把版本变更提交到 Git。
- registry 读操作默认可匿名；如果远程接口异常，或私有 skill 因未登录/无权限被拒绝，直接说明原因并给出下一步。
- `list --remote` 与 `search` 的目录缓存是读优化层，不要把它表述成“下载量、评分一定最新”的强保证。

## 参考资料

需要具体命令模板、典型操作顺序或排障清单时，再读取 [references/cli-workflows.md](references/cli-workflows.md)。
需要为 skill 选择 `category` 和 `official tags` 时，再读取 [references/publish-taxonomy.md](references/publish-taxonomy.md)。
