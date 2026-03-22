---
name: skill-home-manager
version: 0.1.0
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

用这个 skill 操作当前仓库里的 `skill-home` CLI，把“管理 skill”的能力封装成 Codex 可重复执行的工作流。

## 触发场景

- 用户要在当前仓库里新建或修改一个 skill 子项目
- 用户要运行 `skill-home create/init/validate/scan/pack/preview/export/sync/install/doctor`
- 用户要把本地 skill 安装到 Codex
- 用户怀疑本机 `skill-home` 二进制过旧，需要重新构建并覆盖安装

## 默认上下文

- 仓库根目录: `/home/zhuyue/code/skill-manager`
- CLI 源码目录: `/home/zhuyue/code/skill-manager/skill-home-cli`
- 本机配置文件: `/home/zhuyue/.config/skill-home/config.yaml`
- Codex 全局 skills 目录: `/mnt/c/Users/zhuyu/.codex/skills`

## 默认工作流

1. 先查看目标 skill 目录、现有示例和 CLI 支持的真实命令，不靠猜测写操作流。
2. 对高频动作，优先跑 bundled scripts，而不是手写一长串命令。
3. 新建本地 skill:
   `scripts/create-local-skill.sh <name> [description] [output-dir]`
4. 把本地 skill 安装到 Codex:
   `scripts/install-to-codex.sh <skill-path>`
5. 如果 CLI 行为不像当前源码，先重建再继续:
   `scripts/rebuild-cli.sh`
6. 如果问题和路径、配置、注册中心或 API Key 有关，先跑 `skill-home doctor`。

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
- 排查环境与配置: `skill-home doctor`

`install` 适合远程 skill 引用，不适合本地 skill 目录。对本地 skill 目录，优先用 `sync` 或 `export --install`。

## 用户请求到动作的映射

- “帮我新建一个 skill 子项目”: 先用 `scripts/create-local-skill.sh`
- “帮我把这个 skill 装到 Codex”: 先用 `scripts/install-to-codex.sh`
- “帮我把本地 skill-home 更新到最新源码版本”: 先用 `scripts/rebuild-cli.sh`
- “帮我排查为什么没生效”: 先看 `skill-home doctor`，必要时再开 `--debug` 重跑相关命令

## Codex 特别约束

- 这台机器是 WSL + Windows 混合环境，向 Codex 安装 skill 时优先 `--mode mirror`，避免生成指向 Linux 路径的符号链接。
- 如需修改 `/usr/local/bin` 或用户配置文件，先在执行前明确说明你要这么做。
- registry 相关命令不要默认可用；如果远程接口异常或未配置 API Key，直接说明并切回本地工作流。

## 参考资料

需要具体命令模板、典型操作顺序或排障清单时，再读取 [references/cli-workflows.md](references/cli-workflows.md)。
