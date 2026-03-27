---
name: skill-home-manager
version: 0.2.3
description: 当用户想用本地 skill-home CLI 创建、编辑、验证、打包、导出、同步、安装或排查 skill 时使用，尤其适合把 skill 交付到 Codex。
namespace: "@skill-home"
author: Skill Home Team
license: MIT
tags:
  - skill-home
  - codex
  - cli
  - skill-management
homepage: https://github.com/skill-home/skill-manager
ide_config:
  codex:
    globs: ["**/*"]
    auto_activate: false
    tools: [read, edit, bash, glob, grep]
---

# Skill Home Manager

用这个 skill 操作本机已安装的 `skill-home` CLI；缺 CLI 时优先从公开发布入口安装或刷新，把“管理 skill”的能力封装成 Codex 可重复执行的工作流。

## 触发场景

- 用户要在当前仓库里新建或修改一个 skill 子项目
- 用户要运行 `skill-home create/init/validate/scan/pack/preview/export/sync/install/doctor/delete/delete-version`
- 用户要把本地 skill 安装到 Codex
- 用户怀疑本机 `skill-home` 二进制过旧，需要重新构建并覆盖安装
- 用户要删除自己已发布的远程 skill 或某个远程版本

## 默认上下文

- 当前工作目录: 以用户当前任务目录为准，不默认假设存在特定仓库
- 本机配置文件: `~/.config/skill-home/config.yaml`
- 公共安装入口: `http://47.122.112.210:8080/install.sh`
- GitHub Releases: `https://github.com/xianzhiqiyue/skill-manager/releases`
- Codex 全局 skills 目录: 优先使用 `$CODEX_HOME/skills`，常见值是 `~/.codex/skills`

只有在用户明确说明自己正在 `skill-manager` 仓库里开发时，才把该仓库视作额外上下文；不要默认要求本机存在任何 `skill-home` 源码目录。

## 默认工作流

1. 先查看目标 skill 目录、现有示例和 CLI 支持的真实命令，不靠猜测写操作流。
2. 先确保本机真的有可用的 `skill-home` CLI。缺 CLI 时，优先跑:
   `scripts/bootstrap-cli.sh`
3. 对高频动作，优先跑 bundled scripts，而不是手写一长串命令。
4. 新建本地 skill:
   `scripts/create-local-skill.sh <name> [description] [output-dir]`
5. 把本地 skill 安装到 Codex:
   `scripts/install-to-codex.sh <skill-path>`
6. 如果 CLI 版本过旧、缺子命令，或行为不像最新 release，先刷新已发布 CLI 再继续:
   `scripts/rebuild-cli.sh`
7. 如果问题和路径、配置、注册中心或 API Key 有关，先跑 `skill-home doctor`。

## 命令选择规则

- 本地新建 skill: `skill-home init`
- 交互式创建 skill: `skill-home create`
- 格式校验: `skill-home validate`
- 安全扫描: `skill-home scan`
- 打包发布产物: `skill-home pack`
- 预览平台导出结果: `skill-home preview`
- 生成平台格式或直接落盘: `skill-home export`
- 把本地目录同步进 IDE: `skill-home sync`
- 从 registry 安装远程 skill 引用: `skill-home install`
- 删除远程 skill: `skill-home delete`
- 删除远程 skill 版本: `skill-home delete-version`
- 排查环境与配置: `skill-home doctor`

`install` 适合远程 skill 引用，不适合本地 skill 目录。对本地 skill 目录，优先用 `sync` 或 `export --install`。
`push`、`delete`、`delete-version` 需要先登录；公开 skill 的 `pull/install/update/search/info` 不需要登录。
`skill-home login` 默认支持邮箱/密码登录，并会自动为 CLI 创建并保存 API Key；如果用户手上已经有 Key，也可以直接用 `skill-home login --api-key ...`。

## 用户请求到动作的映射

- “帮我把 skill-home CLI 装上 / 补齐本机环境”: 先用 `scripts/bootstrap-cli.sh`
- “帮我新建一个 skill 子项目”: 先用 `scripts/create-local-skill.sh`
- “帮我把这个 skill 装到 Codex”: 先用 `scripts/install-to-codex.sh`
- “帮我把本地 skill-home 更新到最新发布版本”: 先用 `scripts/rebuild-cli.sh`
- “帮我排查为什么没生效”: 先看 `skill-home doctor`，必要时再开 `--debug` 重跑相关命令
- “帮我删除我刚发布的 skill”: 用 `skill-home delete @namespace/name`
- “帮我删除我刚发布的某个版本”: 用 `skill-home delete-version @namespace/name@version`

## Codex 特别约束

- 这台机器是 WSL + Windows 混合环境，向 Codex 安装 skill 时优先 `--mode mirror`，避免生成指向 Linux 路径的符号链接。
- `scripts/bootstrap-cli.sh` 默认从已部署安装页拉取公开安装脚本；如果部署页不可用，再回退到 GitHub 上的安装脚本。它安装的是已发布的 CLI，不依赖本地源码目录。
- `scripts/rebuild-cli.sh` 在这个公共 skill 里表示“重新安装最新发布版 CLI”，不是从源码重建。
- 不要默认要求本机存在任何 `skill-home` 源码仓库；只有当用户明确在维护该仓库时，才允许走源码工作流。
- 涉及 registry 写操作时，先检查是否已登录；未登录时优先提示用户直接执行 `skill-home login` 走邮箱/密码登录流，或在已有 Key 时使用 `skill-home login --api-key ...`。
- registry 读操作默认可匿名；如果远程接口异常，或私有 skill 因未登录/无权限被拒绝，直接说明原因并给出下一步。

## 参考资料

需要具体命令模板、典型操作顺序或排障清单时，再读取 [references/cli-workflows.md](references/cli-workflows.md)。
