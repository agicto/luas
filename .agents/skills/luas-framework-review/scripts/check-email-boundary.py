#!/usr/bin/env python3

"""Keep outbound email configuration, safety limits, callers, and guidance aligned."""

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
            "DefaultEmailRequestTimeout = 10 * time.Second",
            "RequestTimeout time.Duration",
            'env.GetDuration("MAIL_REQUEST_TIMEOUT", DefaultEmailRequestTimeout)',
            "validateEmailConfig(cfg.Email)",
            "MAIL_FROM and RESEND_API_KEY must be configured together",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/email/email.go",
        (
            "context.WithTimeout(ctx, s.requestTimeout)",
            "http.NewRequestWithContext(",
            "io.LimitReader(response.Body, maxProviderResponseBytes+1)",
            "maxResendRecipients      = 50",
            "maxProviderResponseBytes = 64 * 1024",
            "ErrNotConfigured",
            "ProviderError",
            "html.EscapeString",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/service.go",
        (
            "IsConfigured() bool",
            "SendPasswordResetEmail(ctx context.Context",
            "context.WithoutCancel(ctx)",
            "user.password_reset_lookup_failed",
            '"error_type", fmt.Sprintf("%T", err)',
            "s.mailer.SendPasswordResetEmail(ctx",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/listener.go",
        (
            "user.welcome_email_delivery_failed",
            '"user_id", user.ID',
            "h.mailer.SendWelcomeEmail(ctx",
        ),
    )
    require_all(
        failures,
        "api/docs/EMAIL.md",
        (
            "optional outbound email capability",
            "not a notification starter",
            "reads at most 64 KiB",
            "direct and best-effort",
            "must persist their business state before attempting email delivery",
        ),
    )
    require_all(
        failures,
        "api/.env.example",
        ("MAIL_FROM=", "RESEND_API_KEY=", "MAIL_REQUEST_TIMEOUT=10s"),
    )

    try:
        email_source = read("api/internal/infra/email/email.go")
    except FileNotFoundError:
        email_source = ""
    forbidden = (
        (r"\bdefaultService\b", "process-global email service"),
        (r"(?m)^func SendEmail\(", "package-global SendEmail function"),
        (r"http\.NewRequest\(", "request without caller context"),
        (r"io\.ReadAll\(response\.Body\)", "unbounded provider body read"),
        (r'"to"\s*:\s*to', "raw recipient logging"),
        (r'"subject"\s*:\s*subject', "subject logging"),
        (r"string\(body\)", "provider response body exposure"),
    )
    for pattern, description in forbidden:
        if re.search(pattern, email_source):
            failures.append(
                f"api/internal/infra/email/email.go contains forbidden {description}"
            )

    if failures:
        print("Email boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        "Email boundary check passed "
        "(50-recipient cap, 10s timeout, 64 KiB response cap, no provider-body exposure)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
