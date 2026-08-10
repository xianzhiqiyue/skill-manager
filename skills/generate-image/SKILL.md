---
name: generate-image
version: 0.1.3
description: Generate images for “生成汽车图片”, “画一只猫”, “做张海报”, illustrations, avatars, and product visuals. Prefer a native image tool, with an OpenAI-compatible fallback.
category: 设计与内容
tags:
  - automation
  - integration
license: MIT
---

# Generate Image

Create a new raster image from an ordinary user request. Treat short requests as complete enough to act on: a user should not need to know model names, prompt syntax, sizes, or API terminology.

## Workflow

1. Convert the request into one self-contained image prompt. Preserve any requested subject, text, composition, style, colors, aspect ratio, and exclusions. Add sensible visual detail when the request is short.
2. Generate immediately unless a missing choice would materially change the result. For “给我生成一张汽车图片”, choose a polished default composition and create one image instead of asking for model, style, size, or quality.
3. Use the first available route:
   - **Native route:** If the host Agent exposes a raster image generation tool, use it according to that tool's contract. Tool names differ by host and may include `imagegen`, `image_gen`, or `GenerateImage`.
   - **Script fallback:** If no native image tool is available, run `scripts/generate_image.py` against an OpenAI-compatible Images API.
4. Verify that generation succeeded. Return the generated image through the host's image/file result mechanism when available; otherwise give the user the saved file path. Never claim success based only on the request being accepted.

## Script Fallback

The fallback has no third-party Python dependency. It uses Python 3's standard library and calls `POST <base-url>/images/generations`.

Configuration is read from these environment variables, in priority order:

| Purpose | Preferred | Compatible fallback | Default |
| --- | --- | --- | --- |
| API key | `IMAGE_GENERATION_API_KEY` | `OPENAI_API_KEY` | required |
| API base URL | `IMAGE_GENERATION_BASE_URL` | `OPENAI_BASE_URL` | `https://api.openai.com/v1` |
| Model | `IMAGE_GENERATION_MODEL` | — | `gpt-image-1` |
| Timeout seconds | `IMAGE_GENERATION_TIMEOUT_SECONDS` | — | `180` |

For a private OpenAI-compatible gateway, set the base URL including its API version prefix and set its image model ID. For example, a gateway that provides `gpt-image-2-codex` should configure `IMAGE_GENERATION_MODEL=gpt-image-2-codex`; do not hard-code a private model or credential into this public skill.

Pass prompts through standard input so user text is not interpreted as shell syntax. Replace `<skill-root>` with this skill's directory and choose a heredoc delimiter that does not occur in the prompt:

```bash
python3 "<skill-root>/scripts/generate_image.py" --prompt-file - <<'IMAGE_PROMPT_EOF'
<complete image prompt>
IMAGE_PROMPT_EOF
```

Useful optional arguments:

- `--size auto|WIDTHxHEIGHT`: unsupported dimensions are mapped to the nearest native aspect ratio.
- `--quality auto|low|medium|high`: omit unless the user expresses a quality preference.
- `--count 1..4`: generate one image by default.
- `--output-format png|jpeg|webp`: omit to use the provider default.
- `--output-dir <path>`: defaults to `generated-images` under the current working directory.
- `--model <id>` and `--base-url <url>`: explicit non-secret overrides for the environment settings.

Read the JSON result from stdout. A successful result contains `ok: true` and a `files` list with absolute and relative paths. A failed result is written to stderr with `ok: false` plus structured `type`, `message`, and, when available, HTTP `status` and `requestId` metadata.

## Error Handling

- Do not ask the user to paste an API key into chat. Explain which environment variable their Agent runtime must configure.
- Preserve the returned HTTP status and request ID. A `502` proves the gateway failed to complete the request, but does not by itself prove whether routing, model availability, or the upstream image service caused it.
- Do not silently switch to a text/chat model when image generation is unavailable.
- Do not paste base64 image data into the conversation.

## Scope

- This version creates new images only.
- Do not claim source-image editing, inpainting, or variation support.
- Generate at most four images per invocation.
