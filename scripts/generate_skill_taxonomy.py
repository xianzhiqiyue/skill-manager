#!/usr/bin/env python3

from __future__ import annotations

import json
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / "config" / "skill-taxonomy.json"
CLI_TARGET = ROOT / "skill-home-cli" / "internal" / "taxonomy" / "taxonomy.json"
SERVER_TARGET = ROOT / "skill-home-server" / "internal" / "taxonomy" / "taxonomy.json"
WEB_TARGET = ROOT / "skill-home-web" / "src" / "generated" / "skillTaxonomy.json"
SKILL_REFERENCE = ROOT / "skills" / "skill-home-manager" / "references" / "publish-taxonomy.md"


def load_taxonomy() -> dict:
    return json.loads(SOURCE.read_text(encoding="utf-8"))


def write_json(target: Path, payload: dict) -> None:
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(
        json.dumps(payload, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


def render_skill_reference(payload: dict) -> str:
    categories = payload["categories"]
    category_aliases = payload["category_aliases"]
    tags = payload["official_tags"]
    aliases = payload["aliases"]

    lines = [
        "# Skill 发布分类与标签词表",
        "",
        "发布到 Skill Home 时，每个 skill 都需要：",
        "",
        "- 1 个 `category`",
        "- 1 到 4 个 `official tags`",
        "",
        "## 一级分类",
        "",
    ]

    for category in categories:
        lines.append(f"- `{category['id']}`: {category['description']}")

    lines.extend(["", "## 旧英文分类兼容", ""])
    for source, target in category_aliases.items():
        lines.append(f"- `{source}` -> `{target}`")

    lines.extend(["", "## 官方标签", ""])
    for tag in tags:
        lines.append(f"- `{tag['id']}`: {tag['description']}")

    lines.extend(["", "## 别名归一化", ""])
    for source, target in sorted(aliases.items()):
        lines.append(f"- `{source}` -> `{target}`")

    lines.extend(
        [
            "",
            "## 使用建议",
            "",
            "- 分类只能从上面的固定列表中选择，不接受自定义分类。",
            "- 一级分类按 skill 的主要交付结果选择，而不是按调用的技术或平台选择。",
            "- 面向特定业务结果的 skill 优先选择 `业务与管理`；只有通用连接器才选择 `平台与连接`。",
            "- `自动化与工作流` 只用于跨场景的通用自动化能力；文档、研究或业务流程仍选择对应结果分类。",
            "- 再从官方标签里挑 1 到 4 个最典型的场景词；API、MCP、自动化等实现方式放在标签中。",
            "- 优先使用受控词表，不要临时发明新的官方标签。",
            "",
            "## 示例",
            "",
            "- `deploy-buddy`: `category: 运维与安全`，`tags: [deployment, ci-cd, automation]`",
            "- `doc-helper`: `category: 文档与办公`，`tags: [docs, analysis, workflow]`",
            "- `crm-contract-audit`: `category: 业务与管理`，`tags: [analysis, api, workflow]`",
            "- `image-generator`: `category: 设计与内容`，`tags: [automation, integration]`",
        ]
    )

    return "\n".join(lines) + "\n"


def main() -> None:
    payload = load_taxonomy()
    write_json(CLI_TARGET, payload)
    write_json(SERVER_TARGET, payload)
    write_json(WEB_TARGET, payload)
    SKILL_REFERENCE.parent.mkdir(parents=True, exist_ok=True)
    SKILL_REFERENCE.write_text(render_skill_reference(payload), encoding="utf-8")


if __name__ == "__main__":
    main()
