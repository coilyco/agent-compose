"""Project a roster into the person.json shape evalkit already reads.

The Go engine produced this through `agent-compose roster --out`, and evalkit
consumed it without knowing which engine wrote it. Emitting the same shape here
is what takes Go out of the eval path without touching evalkit at all. Only the
fields evalkit reads are emitted; agent-compose#338 owns the wider question of
whether the board should read the roster directly instead.

`acts` is emitted because the annotator charter now renders it: a grader judging
a boundary case sees what the seat is supposed to run, not only what it owns and
defers. Decided by Kai on housecast#7184. `tool` and `side` ride along unread so
this stays a faithful projection rather than the three-key narrowing that dropped
`creature` in agent-compose#1848.
"""

from __future__ import annotations

import json
from typing import Any

from checks.roster import Roster, Voice


def _voice(voice: Voice | None) -> dict[str, Any] | None:
    """Empty fields are dropped, so a bank nobody authored is absent, not [].

    evalkit melds role voice with each personality's and skips a seat whose
    sources are all empty, so an emitted-but-empty voice would derive a case
    with no rule in its target.
    """
    if voice is None:
        return None
    out: dict[str, Any] = {"summary": voice.summary}
    for key, value in (
        ("cadence", voice.cadence),
        ("person", voice.person),
        ("tell", voice.tell),
        ("prefer", list(voice.prefer)),
        ("avoid", list(voice.avoid)),
    ):
        if value:
            out[key] = value
    return out


def person_snapshot(roster: Roster) -> dict[str, Any]:
    roles: dict[str, Any] = {}
    for name in roster.role_order:
        role = roster.roles[name]
        roles[name] = {
            "display_name": role.display_name,
            "purpose": role.purpose,
            "skill": role.skill,
            "skill_source": role.skill_source,
            "stance": role.stance,
            "grounding": role.grounding,
            **({"guardrail": role.guardrail} if role.guardrail else {}),
            "supported_model_tiers": list(role.supported_model_tiers),
            "boundaries": list(role.defers),
            "scoped_boundaries": [{"name": s.name, "scope": s.scope} for s in role.scoped],
            "adjacents": [{"role": a.role, "reason": a.reason} for a in role.adjacents],
            "acts": [
                {k: v for k, v in (("tool", a.tool), ("text", a.text), ("side", a.side)) if v}
                for a in role.acts
            ],
            "personalities": list(role.personalities),
            "favorite_color": role.favorite_color,
            "archived": role.archived,
            "identity": {"name": role.identity_name, "pronouns": role.identity_pronouns},
            **({"voice": v} if (v := _voice(role.voice)) else {}),
            "seats": [
                {
                    k: v
                    for k, v in (
                        ("key", s.key),
                        ("harness", s.harness),
                        ("name", role.identity_name),
                        ("pronouns", role.identity_pronouns),
                        ("tier", s.tier),
                        ("legal_name", s.legal_name),
                    )
                    if v is not None
                }
                for s in role.seats
            ],
        }
    return {
        "person": roster.person,
        "source": roster.source,
        "role_order": list(roster.role_order),
        "roles": roles,
        "boundary_order": list(roster.boundary_order),
        "boundaries": {
            name: {"skill": b.skill, "owner": b.owner, "summary": b.summary}
            for name, b in roster.boundaries.items()
        },
        "personalities": {
            name: {
                "skill": p.skill,
                "color": p.color,
                "motif": p.motif,
                "emblem": {"names": list(p.emblem.names), "emoji": p.emblem.emoji},
                **({"voice": v} if (v := _voice(p.voice)) else {}),
            }
            for name, p in roster.personalities.items()
        },
        "guardrail_order": list(roster.guardrail_order),
        "guardrails": {
            name: {
                "skill": g.skill,
                "role": g.role,
                "card": g.card,
                **({"detector": g.detector} if g.reproducible else {}),
                **(
                    {}
                    if g.reproducible
                    else {"attests": {"in": g.attests_in, "out": g.attests_out}}
                ),
            }
            for name, g in roster.guardrails.items()
        },
    }


def dumps(roster: Roster) -> str:
    return json.dumps(person_snapshot(roster), indent=2, ensure_ascii=False) + "\n"
