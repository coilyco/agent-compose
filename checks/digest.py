"""Content digests, the one primitive both halves of housecast share.

It lives here rather than in `compose` because the grading half needs it and
must not import the composition engine to get it. The `sha256:` prefix is wire
format rather than decoration: Go emits the same string, and a pin taken before
this module existed still has to verify against one taken after.
"""

from __future__ import annotations

import hashlib


def digest(raw: bytes | str) -> str:
    if isinstance(raw, str):
        raw = raw.encode()
    return "sha256:" + hashlib.sha256(raw).hexdigest()
