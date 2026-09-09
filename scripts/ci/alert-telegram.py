#!/usr/bin/env python3
"""Send the CI failure alert for a main-branch run.

Extracted verbatim from the inline `python3 -c 'exec(...)'` body that lived in
.forgejo/workflows/ci.yml, which the actions-run-one-line hook rejects. Behavior
is unchanged: same fields, same endpoint, same proxy handling, same exit codes.

The workflow supplies REPO, WORKFLOW, JOB, REF, SHA, and RUN_URL, plus the
BOT_TOKEN and CHAT_ID secrets. FORGEJO_EGRESS_PROXY, API_BASE, and NOTE are
optional, NOTE being a trailing line the caller adds to say what red means.

Migrating this to the in-cluster Ward mapper, so no repository carries a bot
token, is tracked separately in coilyco-flight-deck/agentic-os#975.
"""

from __future__ import annotations

import os
import sys
import urllib.error
import urllib.parse
import urllib.request


def build_message() -> str:
    lines = [
        "CI failed on main",
        f"repo: {os.environ['REPO']}",
        f"workflow: {os.environ['WORKFLOW']}",
        f"job: {os.environ['JOB']}",
        f"ref: {os.environ['REF']}",
        f"sha: {os.environ['SHA']}",
        f"run: {os.environ['RUN_URL']}",
    ]
    # A red release can still have shipped, so the caller says what red means.
    note = os.environ.get("NOTE", "").strip()
    if note:
        lines.append(note)
    return "\n".join(lines)


def main() -> int:
    bot_token = os.environ.get("BOT_TOKEN", "")
    chat_id = os.environ.get("CHAT_ID", "")
    if not bot_token or not chat_id:
        print("telegram alert missing required secret", file=sys.stderr)
        return 2
    proxy_url = os.environ.get("FORGEJO_EGRESS_PROXY", "").strip()
    payload = urllib.parse.urlencode(
        {
            "chat_id": chat_id,
            "text": build_message(),
            "disable_web_page_preview": "true",
        }
    ).encode("utf-8")
    api_base = os.environ.get("API_BASE", "https://api.telegram.org").rstrip("/")
    request = urllib.request.Request(
        f"{api_base}/bot{bot_token}/sendMessage",
        data=payload,
        method="POST",
    )
    opener = urllib.request.build_opener(
        urllib.request.ProxyHandler({"https": proxy_url})
        if proxy_url
        else urllib.request.ProxyHandler()
    )
    try:
        with opener.open(request, timeout=15) as response:
            response.read()
    except urllib.error.HTTPError as exc:
        # The status separates the fixes: 400 is a malformed chat id, 401 a bad
        # token, 5xx wait and retry. The body is not printed, it echoes payload.
        print(f"telegram alert failed (HTTP {exc.code})", file=sys.stderr)
        return 1
    except urllib.error.URLError as exc:
        print(f"telegram alert failed ({type(exc).__name__})", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
