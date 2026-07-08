#!/usr/bin/env python3

"""Create a lightweight HTML architecture review report in TMPDIR."""

from __future__ import annotations

import argparse
import datetime as dt
import html
import json
import os
from pathlib import Path
from typing import Any


def text(value: Any, fallback: str = "TBD") -> str:
    if value is None:
        return fallback
    if isinstance(value, list):
        return "\n".join(str(item) for item in value if str(item).strip()) or fallback
    rendered = str(value).strip()
    return rendered if rendered else fallback


def escaped_lines(value: Any, fallback: str = "TBD") -> str:
    rendered = html.escape(text(value, fallback))
    return "<br>".join(rendered.splitlines())


def parse_files(raw: str | None) -> list[str]:
    if not raw:
        return []
    return [part.strip() for part in raw.split(",") if part.strip()]


def normalize_candidates(payload: dict[str, Any], args: argparse.Namespace) -> list[dict[str, Any]]:
    candidates = payload.get("candidates")
    if isinstance(candidates, list) and candidates:
        return [candidate for candidate in candidates if isinstance(candidate, dict)]

    return [
        {
            "axis": args.axis,
            "severity": args.severity,
            "strength": args.strength,
            "files": parse_files(args.files),
            "problem": args.problem,
            "proposal": args.proposal,
            "before": args.before,
            "after": args.after,
            "test_impact": args.test_impact,
            "risk": args.risk,
            "rollback": args.rollback,
            "verification": args.verification,
        }
    ]


def load_payload(path: str | None) -> dict[str, Any]:
    if not path:
        return {}
    with open(path, "r", encoding="utf-8") as handle:
        payload = json.load(handle)
    if not isinstance(payload, dict):
        raise SystemExit("--input must be a JSON object")
    return payload


def output_path(args: argparse.Namespace, title: str) -> Path:
    if args.output:
        return Path(args.output).expanduser().resolve()

    safe_title = "".join(ch.lower() if ch.isalnum() else "-" for ch in title).strip("-")
    safe_title = "-".join(part for part in safe_title.split("-") if part) or "luas-architecture-review"
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%d-%H%M%S")
    return Path(os.environ.get("TMPDIR", "/tmp")) / f"{safe_title}-{stamp}.html"


def render_candidate(candidate: dict[str, Any], index: int) -> str:
    files = candidate.get("files", [])
    if isinstance(files, str):
        files = parse_files(files)
    file_items = "\n".join(f"<li><code>{html.escape(str(item))}</code></li>" for item in files)
    if not file_items:
        file_items = "<li>TBD</li>"

    return f"""
      <section class="candidate">
        <div class="candidate-header">
          <h2>Candidate {index}: {html.escape(text(candidate.get("axis"), "Unscored axis"))}</h2>
          <div class="badges">
            <span>Severity: {html.escape(text(candidate.get("severity"), "P2"))}</span>
            <span>Recommendation Strength: {html.escape(text(candidate.get("strength"), "Medium"))}</span>
          </div>
        </div>
        <div class="grid">
          <article>
            <h3>Files</h3>
            <ul>{file_items}</ul>
          </article>
          <article>
            <h3>Problem</h3>
            <p>{escaped_lines(candidate.get("problem"))}</p>
          </article>
          <article>
            <h3>Proposed Deeper Seam</h3>
            <p>{escaped_lines(candidate.get("proposal"))}</p>
          </article>
          <article>
            <h3>Test Impact</h3>
            <p>{escaped_lines(candidate.get("test_impact"))}</p>
          </article>
          <article>
            <h3>Risk</h3>
            <p>{escaped_lines(candidate.get("risk"))}</p>
          </article>
          <article>
            <h3>Rollback</h3>
            <p>{escaped_lines(candidate.get("rollback"))}</p>
          </article>
        </div>
        <div class="flow">
          <div>
            <h3>Before</h3>
            <pre>{html.escape(text(candidate.get("before")))}</pre>
          </div>
          <div>
            <h3>After</h3>
            <pre>{html.escape(text(candidate.get("after")))}</pre>
          </div>
        </div>
        <article class="verification">
          <h3>Verification</h3>
          <p>{escaped_lines(candidate.get("verification"))}</p>
        </article>
      </section>
    """


def render_report(title: str, context: str, candidates: list[dict[str, Any]]) -> str:
    generated_at = dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds")
    rendered_candidates = "\n".join(
        render_candidate(candidate, index) for index, candidate in enumerate(candidates, start=1)
    )
    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{html.escape(title)}</title>
  <style>
    :root {{
      color-scheme: light;
      --ink: #17202a;
      --muted: #5f6b7a;
      --line: #d8dee8;
      --panel: #f7f8fb;
      --accent: #0f766e;
    }}
    body {{
      margin: 0;
      color: var(--ink);
      font: 15px/1.55 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: white;
    }}
    main {{
      max-width: 1120px;
      margin: 0 auto;
      padding: 40px 24px 64px;
    }}
    header {{
      border-bottom: 1px solid var(--line);
      margin-bottom: 28px;
      padding-bottom: 20px;
    }}
    h1, h2, h3 {{ margin: 0; line-height: 1.2; }}
    h1 {{ font-size: 30px; }}
    h2 {{ font-size: 21px; }}
    h3 {{ color: var(--muted); font-size: 13px; text-transform: uppercase; }}
    p {{ margin: 8px 0 0; }}
    code, pre {{
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 13px;
    }}
    pre {{
      min-height: 120px;
      margin: 10px 0 0;
      padding: 14px;
      overflow: auto;
      white-space: pre-wrap;
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
    }}
    .meta {{ color: var(--muted); margin-top: 8px; }}
    .candidate {{
      border: 1px solid var(--line);
      border-radius: 8px;
      margin-top: 20px;
      padding: 20px;
    }}
    .candidate-header {{
      align-items: center;
      display: flex;
      gap: 16px;
      justify-content: space-between;
      margin-bottom: 18px;
    }}
    .badges {{ display: flex; gap: 8px; }}
    .badges span {{
      background: #e7f5f3;
      border: 1px solid #b7ded8;
      border-radius: 999px;
      color: var(--accent);
      padding: 3px 10px;
      white-space: nowrap;
    }}
    .grid {{
      display: grid;
      gap: 16px;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    }}
    article {{
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 14px;
    }}
    ul {{ margin: 8px 0 0; padding-left: 20px; }}
    .flow {{
      display: grid;
      gap: 16px;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      margin-top: 16px;
    }}
    .verification {{ margin-top: 16px; }}
  </style>
</head>
<body>
  <main>
    <header>
      <h1>{html.escape(title)}</h1>
      <p class="meta">Generated at {html.escape(generated_at)} UTC</p>
      <p>{escaped_lines(context, "No context provided.")}</p>
    </header>
    {rendered_candidates}
  </main>
</body>
</html>
"""


def main() -> None:
    parser = argparse.ArgumentParser(description="Create a Luas architecture review HTML report.")
    parser.add_argument("--input", help="Optional JSON report payload.")
    parser.add_argument("--output", help="Output HTML path. Defaults to TMPDIR.")
    parser.add_argument("--title", default="Luas Architecture Review")
    parser.add_argument("--context", default="")
    parser.add_argument("--axis", default="Architecture depth")
    parser.add_argument("--severity", default="P2")
    parser.add_argument("--strength", default="Medium")
    parser.add_argument("--files", default="")
    parser.add_argument("--problem", default="")
    parser.add_argument("--proposal", default="")
    parser.add_argument("--before", default="")
    parser.add_argument("--after", default="")
    parser.add_argument("--test-impact", default="")
    parser.add_argument("--risk", default="")
    parser.add_argument("--rollback", default="")
    parser.add_argument("--verification", default="")
    args = parser.parse_args()

    payload = load_payload(args.input)
    title = text(payload.get("title", args.title), args.title)
    context = text(payload.get("context", args.context), args.context)
    candidates = normalize_candidates(payload, args)

    destination = output_path(args, title)
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(render_report(title, context, candidates), encoding="utf-8")
    print(destination)


if __name__ == "__main__":
    main()
