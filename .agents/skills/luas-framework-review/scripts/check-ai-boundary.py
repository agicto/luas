#!/usr/bin/env python3

"""Keep AI execution bounded, private, deterministic, and product-neutral."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def read(relative_path: str) -> str:
    path = ROOT / relative_path
    if not path.exists():
        raise FileNotFoundError(relative_path)
    return path.read_text(encoding="utf-8")


def require_all(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    try:
        content = read(relative_path)
    except FileNotFoundError:
        failures.append(f"{relative_path} is missing")
        return
    for marker in markers:
        if marker not in content:
            failures.append(f"{relative_path} must contain {marker!r}")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "api/internal/infra/config/config.go",
        (
            "DefaultAIRequestTimeout = 120 * time.Second",
            "DefaultAIMaxInputBytes = 1024 * 1024",
            "DefaultAIMaxResponseBytes int64 = 4 * 1024 * 1024",
            "DefaultAIMaxStreamEventBytes = 1024 * 1024",
            'env.GetBool("AI_ENABLED", false)',
            'env.Get("AI_DEFAULT_MODEL", "")',
            'env.GetInt("AI_MAX_INPUT_BYTES", DefaultAIMaxInputBytes)',
            "validateAIConfig(cfg.AI, isProd)",
            "OPENAI_BASE_URL must use https in production",
        ),
    )
    require_all(
        failures,
        "api/internal/capabilities/ai/ai.go",
        (
            "type RequestLimits struct",
            "ErrProviderNotConfigured",
            "ErrProviderRequestFailed",
            "ErrProviderResponseTooLarge",
            "type ProviderError struct",
            "ProviderResponseID string",
            "sort.Strings(names)",
            "utf8.ValidString(req.Input)",
            "len(req.Instructions) > maxInputBytes-len(req.Input)",
        ),
    )
    require_all(
        failures,
        "api/internal/capabilities/ai/openai.go",
        (
            "context.WithTimeout(ctx, p.timeout)",
            "io.LimitReader(body, maxBytes+1)",
            "io.LimitReader(body, maxProviderErrorDrainBytes)",
            "MaxResponseHeaderBytes: maxProviderResponseHeaderBytes",
            "MinVersion: tls.VersionTLS12",
            "Proxy: http.ProxyFromEnvironment",
            "CheckRedirect:",
            "scanner.Buffer(",
            "ProviderError{Provider: ProviderOpenAI}",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/ai.go",
        (
            "MaxInputBytes:       cfg.MaxInputBytes",
            "MaxResponseBytes:    cfg.MaxResponseBytes",
            "MaxStreamEventBytes: cfg.MaxStreamEventBytes",
        ),
    )
    require_all(
        failures,
        "api/docs/AI.md",
        (
            "not an AI product starter",
            "Luas deliberately has no fast-moving default model",
            "Provider response bodies and provider-supplied error messages never enter returned errors",
            "Luas does not automatically retry generation",
            "prompt templates and version history",
        ),
    )
    require_all(
        failures,
        "api/.env.example",
        (
            "AI_ENABLED=false",
            "AI_DEFAULT_MODEL=",
            "AI_MAX_INPUT_BYTES=1048576",
            "AI_MAX_RESPONSE_BYTES=4194304",
            "AI_MAX_STREAM_EVENT_BYTES=1048576",
        ),
    )

    try:
        source = read("api/internal/capabilities/ai/openai.go")
    except FileNotFoundError:
        source = ""
    forbidden = (
        (r"io\.ReadAll\(httpResp\.Body\)", "unbounded provider response read"),
        (r"string\((?:respBody|errBody|responseBody)\)", "provider body exposure"),
        (r"(?:responsePayload|ev)\.Error\.Message", "provider error message exposure"),
        (r"client:\s*&http\.Client\{\}", "bare provider HTTP client"),
        (r"http\.NewRequest\(", "provider request without caller context"),
    )
    for pattern, description in forbidden:
        if re.search(pattern, source):
            failures.append(
                f"api/internal/capabilities/ai/openai.go contains forbidden {description}"
            )

    if source.count("context.WithTimeout(ctx, p.timeout)") != 2:
        failures.append(
            "api/internal/capabilities/ai/openai.go must bound both one-shot and streaming calls"
        )

    env_example = read("api/.env.example")
    if re.search(r"^AI_ENABLED=true$", env_example, re.MULTILINE):
        failures.append("api/.env.example must keep the optional AI capability disabled")
    if re.search(r"^AI_DEFAULT_MODEL=.+$", env_example, re.MULTILINE):
        failures.append(
            "api/.env.example must not freeze a fast-moving provider model default"
        )

    if failures:
        print("AI boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        "AI boundary check passed "
        "(disabled default, explicit model, bounded I/O, private errors, timed streams)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
