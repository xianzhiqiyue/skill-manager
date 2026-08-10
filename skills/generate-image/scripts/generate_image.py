#!/usr/bin/env python3
"""Generate images through an OpenAI-compatible Images API."""

from __future__ import annotations

import argparse
import base64
import binascii
import json
import mimetypes
import os
import re
import sys
import tempfile
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlparse, urlunparse
from urllib.request import Request, urlopen

MAX_IMAGE_BYTES = 50 * 1024 * 1024
MAX_RESPONSE_BYTES = 100 * 1024 * 1024
MAX_ERROR_BYTES = 64 * 1024
DEFAULT_BASE_URL = "https://api.openai.com/v1"
DEFAULT_MODEL = "gpt-image-1"
DEFAULT_TIMEOUT_SECONDS = 180
NATIVE_SIZES = ("1024x1024", "1536x1024", "1024x1536")
NATIVE_ASPECT_RATIOS = {
    "1024x1024": 1.0,
    "1536x1024": 1.5,
    "1024x1536": 2 / 3,
}


class GenerateImageError(RuntimeError):
    """Structured error safe to return to an Agent."""

    def __init__(
        self,
        message: str,
        *,
        error_type: str,
        status: int | None = None,
        request_id: str | None = None,
    ) -> None:
        super().__init__(message)
        self.error_type = error_type
        self.status = status
        self.request_id = request_id

    def as_dict(self) -> dict[str, Any]:
        result: dict[str, Any] = {
            "type": self.error_type,
            "message": str(self),
        }
        if self.status is not None:
            result["status"] = self.status
        if self.request_id:
            result["requestId"] = self.request_id
        return result


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    prompt_group = parser.add_mutually_exclusive_group(required=True)
    prompt_group.add_argument("--prompt", help="Complete image prompt")
    prompt_group.add_argument(
        "--prompt-file",
        help="UTF-8 prompt file, or '-' to read from standard input",
    )
    parser.add_argument("--size", help="auto or requested WIDTHxHEIGHT")
    parser.add_argument("--quality", choices=("auto", "low", "medium", "high"))
    parser.add_argument("--count", type=int, default=1)
    parser.add_argument("--output-format", choices=("png", "jpeg", "webp"))
    parser.add_argument("--output-dir", default="generated-images")
    parser.add_argument("--model", help="Non-secret model ID override")
    parser.add_argument("--base-url", help="Non-secret API base URL override")
    parser.add_argument("--timeout", type=int, help="Request timeout in seconds")
    args = parser.parse_args()
    if not 1 <= args.count <= 4:
        parser.error("--count must be between 1 and 4")
    if args.timeout is not None and not 1 <= args.timeout <= 3600:
        parser.error("--timeout must be between 1 and 3600")
    return args


def _read_prompt(args: argparse.Namespace) -> str:
    if args.prompt is not None:
        prompt = args.prompt
    elif args.prompt_file == "-":
        prompt = sys.stdin.read()
    else:
        try:
            prompt = Path(args.prompt_file).read_text(encoding="utf-8")
        except OSError as exc:
            raise GenerateImageError(
                f"cannot read prompt file: {exc}", error_type="invalid_input"
            ) from exc
    prompt = prompt.strip()
    if not prompt:
        raise GenerateImageError(
            "image prompt cannot be empty", error_type="invalid_input"
        )
    return prompt


def _first_env(*names: str) -> str:
    for name in names:
        value = os.environ.get(name, "").strip()
        if value:
            return value
    return ""


def _resolve_timeout(cli_timeout: int | None) -> int:
    if cli_timeout is not None:
        return cli_timeout
    raw = _first_env("IMAGE_GENERATION_TIMEOUT_SECONDS")
    if not raw:
        return DEFAULT_TIMEOUT_SECONDS
    try:
        value = int(raw)
    except ValueError as exc:
        raise GenerateImageError(
            "IMAGE_GENERATION_TIMEOUT_SECONDS must be an integer",
            error_type="configuration_error",
        ) from exc
    if not 1 <= value <= 3600:
        raise GenerateImageError(
            "IMAGE_GENERATION_TIMEOUT_SECONDS must be between 1 and 3600",
            error_type="configuration_error",
        )
    return value


def _resolve_endpoint(cli_base_url: str | None) -> str:
    base_url = (
        (cli_base_url or "").strip()
        or _first_env("IMAGE_GENERATION_BASE_URL", "OPENAI_BASE_URL")
        or DEFAULT_BASE_URL
    )
    parsed = urlparse(base_url)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise GenerateImageError(
            "image API base URL must be an absolute HTTP(S) URL",
            error_type="configuration_error",
        )
    if parsed.query or parsed.fragment:
        raise GenerateImageError(
            "image API base URL must not contain a query string or fragment",
            error_type="configuration_error",
        )
    path = parsed.path.rstrip("/")
    if not path.endswith("/images/generations"):
        path += "/images/generations"
    return urlunparse(parsed._replace(path=path))


def _resolve_size(raw_size: str | None) -> tuple[str | None, str | None]:
    if raw_size is None:
        return None, None
    requested = raw_size.strip().lower().replace("×", "x").replace(" ", "")
    if requested == "auto" or requested in NATIVE_SIZES:
        return requested, requested
    match = re.fullmatch(r"(\d+)x(\d+)", requested)
    if not match or int(match.group(1)) == 0 or int(match.group(2)) == 0:
        raise GenerateImageError(
            "image size must be auto or WIDTHxHEIGHT",
            error_type="invalid_input",
        )
    aspect_ratio = int(match.group(1)) / int(match.group(2))
    resolved = min(
        NATIVE_SIZES,
        key=lambda size: abs(aspect_ratio - NATIVE_ASPECT_RATIOS[size]),
    )
    return requested, resolved


def _request_id(headers: Any) -> str | None:
    for name in ("x-request-id", "request-id", "x-correlation-id"):
        value = headers.get(name) if headers is not None else None
        if value:
            return str(value)
    return None


def _error_detail(body: bytes, fallback: str) -> str:
    text = body.decode("utf-8", errors="replace").strip()
    if not text:
        return fallback
    try:
        payload = json.loads(text)
    except json.JSONDecodeError:
        return text[:4096]
    if isinstance(payload, dict):
        error = payload.get("error")
        if isinstance(error, dict) and isinstance(error.get("message"), str):
            return error["message"][:4096]
        if isinstance(error, str):
            return error[:4096]
        if isinstance(payload.get("message"), str):
            return payload["message"][:4096]
    return text[:4096]


def _request_generation(
    endpoint: str,
    api_key: str,
    payload: dict[str, Any],
    timeout_seconds: int,
) -> dict[str, Any]:
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "Accept": "application/json",
    }
    organization = _first_env("OPENAI_ORG_ID", "OPENAI_ORGANIZATION")
    project = _first_env("OPENAI_PROJECT_ID", "OPENAI_PROJECT")
    if organization:
        headers["OpenAI-Organization"] = organization
    if project:
        headers["OpenAI-Project"] = project

    request = Request(
        endpoint,
        data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
        headers=headers,
        method="POST",
    )
    try:
        with urlopen(request, timeout=timeout_seconds) as response:
            body = response.read(MAX_RESPONSE_BYTES + 1)
    except HTTPError as exc:
        detail = _error_detail(exc.read(MAX_ERROR_BYTES), str(exc.reason))
        raise GenerateImageError(
            f"image API returned HTTP {exc.code}: {detail}",
            error_type="http_error",
            status=exc.code,
            request_id=_request_id(exc.headers),
        ) from exc
    except (OSError, URLError) as exc:
        raise GenerateImageError(
            f"cannot reach image API: {exc}", error_type="connection_error"
        ) from exc
    if len(body) > MAX_RESPONSE_BYTES:
        raise GenerateImageError(
            "image API response exceeds the 100 MiB safety limit",
            error_type="invalid_response",
        )
    try:
        result = json.loads(body)
    except json.JSONDecodeError as exc:
        raise GenerateImageError(
            "image API returned invalid JSON", error_type="invalid_response"
        ) from exc
    if not isinstance(result, dict):
        raise GenerateImageError(
            "image API returned an invalid result", error_type="invalid_response"
        )
    return result


def _decode_base64_image(value: str) -> bytes:
    encoded = value.strip()
    if encoded.startswith("data:"):
        if "," not in encoded:
            raise GenerateImageError(
                "image API returned an invalid data URL",
                error_type="invalid_response",
            )
        encoded = encoded.split(",", 1)[1]
    if len(encoded) > (MAX_IMAGE_BYTES * 4 // 3) + 16:
        raise GenerateImageError(
            "generated image exceeds the 50 MiB safety limit",
            error_type="invalid_response",
        )
    try:
        data = base64.b64decode(encoded, validate=True)
    except (ValueError, binascii.Error) as exc:
        raise GenerateImageError(
            "image API returned invalid base64 data",
            error_type="invalid_response",
        ) from exc
    if not data:
        raise GenerateImageError(
            "image API returned an empty image", error_type="invalid_response"
        )
    if len(data) > MAX_IMAGE_BYTES:
        raise GenerateImageError(
            "generated image exceeds the 50 MiB safety limit",
            error_type="invalid_response",
        )
    return data


def _download_image(url: str, timeout_seconds: int) -> tuple[bytes, str | None]:
    parsed = urlparse(url)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise GenerateImageError(
            "image API returned an unsupported image URL",
            error_type="invalid_response",
        )
    request = Request(url, headers={"Accept": "image/*"})
    try:
        with urlopen(request, timeout=timeout_seconds) as response:
            content_type = response.headers.get_content_type()
            data = response.read(MAX_IMAGE_BYTES + 1)
    except HTTPError as exc:
        raise GenerateImageError(
            f"generated image download returned HTTP {exc.code}",
            error_type="download_error",
            status=exc.code,
            request_id=_request_id(exc.headers),
        ) from exc
    except (OSError, URLError) as exc:
        raise GenerateImageError(
            f"cannot download generated image: {exc}",
            error_type="download_error",
        ) from exc
    if not data:
        raise GenerateImageError(
            "generated image download was empty", error_type="download_error"
        )
    if len(data) > MAX_IMAGE_BYTES:
        raise GenerateImageError(
            "generated image exceeds the 50 MiB safety limit",
            error_type="download_error",
        )
    return data, content_type


def _image_extension(data: bytes, content_type: str | None) -> str:
    if data.startswith(b"\x89PNG\r\n\x1a\n"):
        return ".png"
    if data.startswith(b"\xff\xd8\xff"):
        return ".jpg"
    if data.startswith((b"GIF87a", b"GIF89a")):
        return ".gif"
    if data.startswith(b"RIFF") and data[8:12] == b"WEBP":
        return ".webp"
    return {
        "image/png": ".png",
        "image/jpeg": ".jpg",
        "image/gif": ".gif",
        "image/webp": ".webp",
    }.get(str(content_type or "").lower(), ".png")


def _save_image(
    output_dir: Path,
    data: bytes,
    index: int,
    content_type: str | None,
) -> Path:
    extension = _image_extension(data, content_type)
    with tempfile.NamedTemporaryFile(
        mode="wb",
        prefix=f"generated-{index + 1}-",
        suffix=extension,
        dir=output_dir,
        delete=False,
    ) as handle:
        handle.write(data)
        temporary_path = Path(handle.name)
    final_path = temporary_path.with_name(
        temporary_path.name.replace("generated-", "image-", 1)
    )
    temporary_path.replace(final_path)
    return final_path


def _extract_and_save_images(
    result: dict[str, Any],
    output_dir: Path,
    timeout_seconds: int,
) -> list[dict[str, Any]]:
    items = result.get("data")
    if not isinstance(items, list) or not items:
        raise GenerateImageError(
            "image API returned no images", error_type="invalid_response"
        )
    files: list[dict[str, Any]] = []
    for index, item in enumerate(items):
        if not isinstance(item, dict):
            raise GenerateImageError(
                "image API returned an invalid image item",
                error_type="invalid_response",
            )
        content_type: str | None = None
        if isinstance(item.get("b64_json"), str):
            data = _decode_base64_image(item["b64_json"])
        elif isinstance(item.get("url"), str):
            data, content_type = _download_image(item["url"], timeout_seconds)
        else:
            raise GenerateImageError(
                "image API returned neither b64_json nor url",
                error_type="invalid_response",
            )
        path = _save_image(output_dir, data, index, content_type)
        mime_type = content_type or mimetypes.guess_type(path.name)[0] or "image/png"
        files.append(
            {
                "path": str(path),
                "relativePath": os.path.relpath(path, Path.cwd()),
                "name": path.name,
                "bytes": len(data),
                "mimeType": mime_type,
            }
        )
    return files


def main() -> int:
    args = _parse_args()
    try:
        prompt = _read_prompt(args)
        api_key = _first_env("IMAGE_GENERATION_API_KEY", "OPENAI_API_KEY")
        if not api_key:
            raise GenerateImageError(
                "image API key is missing; configure IMAGE_GENERATION_API_KEY or OPENAI_API_KEY in the Agent runtime",
                error_type="configuration_error",
            )
        model = (
            (args.model or "").strip()
            or _first_env("IMAGE_GENERATION_MODEL")
            or DEFAULT_MODEL
        )
        endpoint = _resolve_endpoint(args.base_url)
        timeout_seconds = _resolve_timeout(args.timeout)
        requested_size, resolved_size = _resolve_size(args.size)
        output_dir = Path(args.output_dir).expanduser()
        if not output_dir.is_absolute():
            output_dir = Path.cwd() / output_dir
        output_dir = output_dir.resolve()
        output_dir.mkdir(parents=True, exist_ok=True)

        payload: dict[str, Any] = {
            "model": model,
            "prompt": prompt,
            "n": args.count,
        }
        if resolved_size:
            payload["size"] = resolved_size
        if args.quality:
            payload["quality"] = args.quality
        if args.output_format:
            payload["output_format"] = args.output_format

        result = _request_generation(endpoint, api_key, payload, timeout_seconds)
        files = _extract_and_save_images(result, output_dir, timeout_seconds)
        output: dict[str, Any] = {
            "ok": True,
            "model": model,
            "files": files,
        }
        if requested_size and requested_size != resolved_size:
            output["requestedSize"] = requested_size
            output["resolvedSize"] = resolved_size
        print(json.dumps(output, ensure_ascii=False))
        return 0
    except GenerateImageError as exc:
        print(
            json.dumps({"ok": False, "error": exc.as_dict()}, ensure_ascii=False),
            file=sys.stderr,
        )
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
