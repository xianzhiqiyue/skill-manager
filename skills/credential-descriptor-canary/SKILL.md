---
name: credential-descriptor-canary
version: 0.1.0
description: 用于验证 skill-home 凭证描述符链路的最小 canary skill
namespace: "@user"
description_zh: 用于验证 metadata.openclaw.credentials 解析、存储和 API 返回的最小技能
author: Skill Home Team
license: MIT
tags:
  - canary
  - credentials
  - skill-home
metadata:
  openclaw:
    credentials:
      - id: openai_api_key
        env: OPENAI_API_KEY
        label: OpenAI API Key
        description: 用于验证 credentials 描述符会被 skill-home 正确解析并返回
        secret: true
        required: true
        input: password
        help_url: https://platform.openai.com/api-keys
---

# Credential Descriptor Canary

这个 skill 只用于验证 `metadata.openclaw.credentials` 的端到端链路。

## 目标

- 验证 skill 包中的凭证描述符会被 skill-home 解析并持久化
- 验证详情接口会在顶层返回 `credentials`
- 验证兼容层会从 `credentials[].env` 自动派生 `manifest.requires`

## 使用说明

- 将它视为一个最小验收样本，而不是正式的生产技能
- 如果运行环境要求实际调用 OpenAI，请先配置 `OPENAI_API_KEY`

## 验收要点

1. `GET /api/v1/skills/user/credential-descriptor-canary` 顶层返回 `credentials`
2. 最新版本的 `manifest.metadata.openclaw.credentials` 存在
3. 最新版本的 `manifest.requires` 包含 `OPENAI_API_KEY`
