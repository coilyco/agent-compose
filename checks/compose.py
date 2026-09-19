"""Bundle emission, ported from internal/bundle and internal/resolver.

Go's encoder escapes <, > and & even inside strings, and Python's does not, so
go_json below re-adds exactly those three. Without it two identical documents
differ on any summary containing an ampersand.
"""

from __future__ import annotations

import json
import pathlib
from typing import TYPE_CHECKING, Any

from checks import render, voiceprofile
from checks.digest import digest

if TYPE_CHECKING:
    from checks.roster import Role, Roster, Seat

SOURCE_SAFE = set("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.")


def source_segment(value: str) -> str:
    out = []
    for octet in value.encode():
        char = chr(octet)
        out.append(char if char in SOURCE_SAFE else f"%{octet:02X}")
    return "".join(out)


def _escape(text: str) -> str:
    return text.replace("<", "\\u003c").replace(">", "\\u003e").replace("&", "\\u0026")


def go_json(value: Any) -> str:
    return _escape(json.dumps(value, indent=2, ensure_ascii=False))


def go_json_compact(value: Any) -> str:
    """json.Marshal without indentation, which is what the identity digest covers."""
    return _escape(json.dumps(value, separators=(",", ":"), ensure_ascii=False))


def _identity(roster: Roster, role: Role) -> dict[str, Any]:
    return {
        "person": roster.person,
        "display_name": role.display_name,
        "purpose": role.purpose,
        "seats": [_seat(role, seat) for seat in role.seats],
        "personalities": [
            {
                "name": name,
                "color": roster.personalities[name].color,
                "emblem": {
                    "names": roster.personalities[name].emblem.names,
                    "emoji": roster.personalities[name].emblem.emoji,
                },
            }
            for name in role.personalities
        ],
    }


def _identity_digest_input(roster: Roster, role: Role) -> dict[str, Any]:
    """The anonymous struct manifestContent digests, which carries no json tags.

    Go falls back to field names when a struct has none, so these keys are
    capitalized and the nested types keep their own tags. Anything that reads
    like a typo here is load-bearing.
    """
    return {
        "Purpose": role.purpose,
        "Skill": role.skill,
        "Personalities": list(role.personalities),
        "PersonalityMetadata": [
            {
                "name": name,
                "color": roster.personalities[name].color,
                "emblem": {
                    "names": roster.personalities[name].emblem.names,
                    "emoji": roster.personalities[name].emblem.emoji,
                },
            }
            for name in role.personalities
        ],
        "Seats": [_seat(role, seat) for seat in role.seats],
        "SupportedModelTiers": list(role.supported_model_tiers),
        "FavoriteColor": role.favorite_color,
    }


def _seat(role: Role, seat: Seat) -> dict[str, Any]:
    out = {
        "key": seat.key,
        "harness": seat.harness,
        "name": role.identity_name,
        "pronouns": role.identity_pronouns,
    }
    if seat.tier:
        out["tier"] = seat.tier
    if seat.legal_name:
        out["legal_name"] = seat.legal_name
    return out


def compiled_document(roster: Roster, role_name: str) -> str:
    """Instructions then every selected body, which is joinInstructions plus a blank line."""
    out = render.instructions(roster, role_name)
    for _, body in _selected_bodies(roster, role_name):
        if out and not out.endswith("\n"):
            out += "\n"
        out += "\n" + body
    return out


def _selected_bodies(roster: Roster, role_name: str) -> list[tuple[str, str]]:
    role = roster.roles[role_name]
    bodies = [(role.skill, role.body)]
    bodies += [
        (roster.boundaries[n].skill, roster.boundaries[n].body)
        for n in role.active_boundaries(roster.boundaries)
    ]
    bodies += [
        (roster.personalities[n].skill, roster.personalities[n].body) for n in role.personalities
    ]
    if role.guardrail:
        guardrail = roster.guardrails[role.guardrail]
        bodies.append((guardrail.skill, guardrail.body))
    return bodies


def manifest(
    roster: Roster, role_name: str, model_tier: str, delivery: str = "native-skills"
) -> dict[str, Any]:
    role = roster.roles[role_name]
    active = role.active_boundaries(roster.boundaries)
    source = roster.source
    identity = _identity(roster, role)
    content = [
        {"id": f"{source}:invariant", "digest": digest(roster.invariant)},
        {"id": f"{source}:role:{role_name}", "digest": digest(role.body)},
        {
            "id": f"{source}:role:{role_name}:identity",
            "digest": digest(go_json_compact(_identity_digest_input(roster, role))),
        },
    ]
    for name in sorted(role.personalities):
        binding = roster.personalities[name]
        content.append({"id": f"{source}:skill:{binding.skill}", "digest": digest(binding.body)})
    if role.guardrail:
        guardrail = roster.guardrails[role.guardrail]
        content.append(
            {"id": f"{source}:skill:{guardrail.skill}", "digest": digest(guardrail.body)}
        )
    # The Go engine sorts, and its order is the contract the parity test holds
    # both engines to. Appending happened to be sorted until guardrail arrived.
    content.sort(key=lambda entry: entry["id"])
    return {
        "format": "agent-compose.bundle",
        "role": role_name,
        "role_skill": role.skill,
        "role_skill_source": role.skill_source,
        "role_skill_digest": digest(role.body),
        "model_tier": model_tier,
        "personalities": list(role.personalities),
        "boundaries": active,
        **({"guardrail": role.guardrail} if role.guardrail else {}),
        "color": role.favorite_color,
        "identity": identity,
        "sources": [source],
        "content": content,
        "delivery": _delivery(roster, role_name, delivery),
    }


def _delivery(roster: Roster, role_name: str, mode: str) -> dict[str, Any]:
    # Key order is the Go Delivery struct's, because parity is byte-identical.
    out: dict[str, Any]
    if mode == "native-skills":
        body = render.instructions(roster, role_name)
        out = {
            "mode": mode,
            "instructions": "content/instructions.md",
            "skills_root": "content/skills",
        }
    elif mode == "compiled":
        body = compiled_document(roster, role_name)
        out = {
            "mode": mode,
            "instructions": "content/instructions.md",
            "compiled_context": "delivery/compiled.md",
        }
    else:
        raise ValueError(f"unknown delivery mode {mode!r}")
    if voice_profile(roster, role_name) is not None:
        out["voice_profile"] = "content/voice-profile.json"
    out["body_bytes"] = len(body.encode())
    return out


def voice_profile(roster: Roster, role_name: str) -> dict[str, Any] | None:
    """The seat's merged linter profile. No source ships one here, so no carried half."""
    banks = voiceprofile.banks(roster, role_name)
    return voiceprofile.build(f"{roster.source}:{role_name}", [], voiceprofile.generated(banks))


def trace(roster: Roster, role_name: str, delivery: str = "native-skills") -> dict[str, Any]:
    role = roster.roles[role_name]
    source = roster.source
    active = role.active_boundaries(roster.boundaries)
    scoped = {entry.name for entry in role.scoped}
    meld = ", ".join(role.personalities)
    bound_skills = ", ".join(roster.personalities[n].skill for n in role.personalities)

    decisions = [
        {
            "subject": f"role:{role_name}",
            "kind": "profile",
            "source": source,
            "outcome": "selected",
            "reason": f"{source} defines this role: {role.purpose}",
        }
    ]
    for name in role.personalities:
        decisions.append(
            {
                "subject": f"personality:{name}",
                "kind": "profile",
                "source": source,
                "outcome": "selected",
                "reason": f'role "{role_name}" activates its full personality set: {meld}',
            }
        )
    for name in active:
        if roster.boundaries[name].owner == role_name:
            verb = "owns"
        elif name in scoped:
            verb = "holds within a scope"
        else:
            verb = "defers"
        decisions.append(
            {
                "subject": f"boundary:{name}",
                "kind": "profile",
                "source": source,
                "outcome": "selected",
                "reason": f'role "{role_name}" {verb} boundary "{name}", '
                "whose body is identical on every side",
            }
        )
    decisions.append(
        {
            "subject": f"source:{source}",
            "kind": "source",
            "source": source,
            "outcome": "selected",
            "reason": f'selected person package defines role "{role_name}" '
            "and its active personalities",
        }
    )
    decisions.append(
        {
            "subject": "instruction:personality-invariant",
            "kind": "instruction",
            "source": source,
            "outcome": "selected",
            "reason": "instructions from admitted sources are always selected",
        }
    )

    context_bytes = len(role.body.encode())
    for name in sorted(roster.personalities):
        binding = roster.personalities[name]
        if name in role.personalities:
            decisions.append(
                {
                    "subject": f"skill:{binding.skill}",
                    "kind": "skill",
                    "source": source,
                    "outcome": "selected",
                    "reason": f'active personality "{name}" binds this skill',
                }
            )
            context_bytes += len(binding.body.encode())
        else:
            decisions.append(
                {
                    "subject": f"skill:{binding.skill}",
                    "kind": "skill",
                    "source": source,
                    "outcome": "excluded",
                    "reason": f'role "{role_name}" activates personalities {meld}, '
                    f"which bind skills {bound_skills}",
                }
            )
    decisions.append(
        {
            "subject": f"skill:{role.skill}",
            "kind": "skill",
            "source": source,
            "outcome": "selected",
            "reason": f'role "{role_name}" composes this skill',
        }
    )
    for name in active:
        boundary = roster.boundaries[name]
        decisions.append(
            {
                "subject": f"skill:{boundary.skill}",
                "kind": "skill",
                "source": source,
                "outcome": "selected",
                "reason": f'role "{role_name}" composes this skill',
            }
        )
        context_bytes += len(boundary.body.encode())
    if role.guardrail:
        guardrail = roster.guardrails[role.guardrail]
        decisions.append(
            {
                "subject": f"skill:{guardrail.skill}",
                "kind": "skill",
                "source": source,
                "outcome": "selected",
                "reason": f'role "{role_name}" composes this skill',
            }
        )
        context_bytes += len(guardrail.body.encode())
    decisions.append(
        {
            "subject": "content/instructions.md",
            "kind": "delivery",
            "outcome": "delivered",
            "reason": "canonical selected instructions",
        }
    )
    decisions.append(
        {
            "subject": "content/skills",
            "kind": "delivery",
            "outcome": "delivered",
            "reason": "canonical selected skill trees",
        }
    )
    if delivery == "compiled":
        decisions.append(
            {
                "subject": "delivery/compiled.md",
                "kind": "delivery",
                "outcome": "delivered",
                "reason": "selected instructions and skill prose compiled "
                "into one context document",
            }
        )

    skills = 1 + len(active) + len(role.personalities) + bool(role.guardrail)
    return {
        "format": "agent-compose.trace",
        "decisions": decisions,
        "providers": [
            {
                "source": source,
                "category": "person-package",
                "scope": "person",
                "outcome": "selected",
                "reason": f'selected person package defines role "{role_name}" '
                "and its active personalities",
                "skills": skills,
                "context_bytes": context_bytes,
                "approximate_tokens": (context_bytes + 3) // 4,
            }
        ],
    }


def compose(
    roster: Roster,
    role_name: str,
    model_tier: str,
    out_dir: pathlib.Path | str,
    delivery: str = "native-skills",
) -> pathlib.Path:
    role = roster.roles[role_name]
    # The Go resolver refuses here too. Parsing may differ between the engines,
    # semantics may not.
    if role.archived:
        raise ValueError(f"role {role_name!r} is archived and cannot be composed")
    if model_tier not in role.supported_model_tiers:
        raise ValueError(
            f"role {role_name!r} does not support model tier {model_tier!r}; "
            f"supported: {', '.join(role.supported_model_tiers)}"
        )
    out_dir = pathlib.Path(out_dir)
    skills_root = out_dir / "content" / "skills" / source_segment(roster.source)
    for skill, body in _selected_bodies(roster, role_name):
        target = skills_root / skill / "SKILL.md"
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(body)
    (out_dir / "content" / "instructions.md").write_text(render.instructions(roster, role_name))
    if delivery == "compiled":
        target = out_dir / "delivery" / "compiled.md"
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(compiled_document(roster, role_name))
    (out_dir / "manifest.json").write_text(
        go_json(manifest(roster, role_name, model_tier, delivery)) + "\n"
    )
    (out_dir / "trace.json").write_text(go_json(trace(roster, role_name, delivery)) + "\n")
    profile = voice_profile(roster, role_name)
    if profile is not None:
        (out_dir / "content" / "voice-profile.json").write_text(go_json(profile) + "\n")
    return out_dir
