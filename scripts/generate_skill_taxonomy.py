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
            "- 先判断 skill 的一级能力域，填写 `category`。",
            "- 再从官方标签里挑 1 到 4 个最典型的场景词。",
            "- 优先使用受控词表，不要临时发明新的官方标签。",
            "",
            "## 示例",
            "",
            "- `deploy-buddy`: `category: ops`，`tags: [deployment, ci-cd, automation]`",
            "- `doc-helper`: `category: docs`，`tags: [docs, analysis, workflow]`",
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
