"""The roster language: what YAML housecast reads, and what it refuses.

Ported from internal/person. The Go side parses KDL and this parses YAML, so
the loaders differ on purpose; what must not differ is the semantics either one
applies once the file is in memory.
"""

from __future__ import annotations

import pathlib
from dataclasses import dataclass, field
from typing import Any

import yaml

from checks import color

DATA = pathlib.Path(__file__).parent / "data" / "roster.yaml"


class RosterError(ValueError):
    """The roster is malformed, or a role asks for something incoherent."""


@dataclass(frozen=True)
class Emblem:
    names: list[str]
    emoji: str


@dataclass(frozen=True)
class Voice:
    """Register. Role authors the baseline, personalities modulate it."""

    summary: str
    cadence: str = ""
    person: str = ""
    tell: str = ""
    prefer: list[str] = field(default_factory=list)
    avoid: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class Guardrail:
    """One strong course correction, a first-class element beside role and boundary.

    It is an element rather than a field on a role because a rule the seat is
    supposed to follow belongs on the same layer as the rules it sits among. As a
    field it had no skill, no body in the composed tree, and reached no bundle at
    all: the board derived cases against an instruction nobody was ever given.

    `card` renders eager into the identity card, so the correction fires without
    the seat choosing to load anything, and `body` is the lazy SKILL.md carrying
    the procedure. A lazy-only guardrail is a guardrail that fires after the
    mistake.

    `detector` carries the whole reproducible-attested split. A guardrail naming
    a detector can be re-run by a grader, so its evidence is independent of the
    seat under test. One without a detector can only be checked for presence and
    adjacency, because the evidence was produced by the seat being graded.

    The kind is declared by the key's presence rather than by a `kind:` string,
    so it cannot drift from the content it labels.

    A reproducible guardrail derives both targets from the detector, because the
    detector is the evidence. An attested one cannot: what counts as evidence
    differs per role, so `attests` carries the two targets and the deriver has no
    generic text to fall back on. Deriving them generically instead reads every
    attested guardrail in whichever role's vocabulary was written first.
    """

    name: str
    skill: str
    role: str
    card: str
    body: str
    detector: str = ""
    attests_in: str = ""
    attests_out: str = ""

    @property
    def reproducible(self) -> bool:
        return bool(self.detector)


@dataclass(frozen=True)
class Outro:
    """What a session says as it closes. Role only, and not melded.

    Voice melds because it is read once as doctrine. An outro is read in half a
    second by somebody already leaving, and role plus two personalities runs to
    three sentences, which is a paragraph rather than a banner.
    """

    clean: str = ""
    failure: str = ""


@dataclass(frozen=True)
class Act:
    """One thing an attribute requires a seat to actually run.

    `tool` is carried separately from `text` so the coverage check is exact
    rather than a grep over prose. `side` is set on boundary acts only, because
    owning, scoping, and deferring are three different acts.
    """

    tool: str
    text: str
    side: str = ""


@dataclass(frozen=True)
class Personality:
    name: str
    skill: str
    color: str
    motif: str
    emblem: Emblem
    body: str
    voice: Voice | None = None
    acts: list[Act] = field(default_factory=list)


@dataclass(frozen=True)
class Boundary:
    name: str
    skill: str
    owner: str
    summary: str
    body: str
    acts: list[Act] = field(default_factory=list)

    def acts_for_side(self, side: str) -> list[Act]:
        """A deferred boundary is a different act, never the owner's withheld."""
        return [act for act in self.acts if act.side == side]


@dataclass(frozen=True)
class Seat:
    key: str
    harness: str
    tier: str | None = None
    channel: str | None = None
    # The seat's own product name, authored rather than derived. Absent stays
    # absent: identity describes and grants nothing (agent-compose#396).
    legal_name: str | None = None


@dataclass(frozen=True)
class Scoped:
    name: str
    scope: str


@dataclass(frozen=True)
class Adjacent:
    """The role most easily confused with this one, and why. evalkit reads it."""

    role: str
    reason: str


@dataclass
class Role:
    name: str
    display_name: str
    purpose: str
    skill: str
    skill_source: str
    stance: str
    supported_model_tiers: list[str]
    defers: list[str]
    scoped: list[Scoped]
    personalities: list[str]
    identity_name: str
    identity_pronouns: str
    seats: list[Seat]
    adjacents: list[Adjacent]
    body: str
    methods: list[str] = field(default_factory=list)
    favorite_color: str = ""
    # A role naming color_twin copies another role's favorite_color instead
    # of deriving its own; see resolve_favorite_colors. agent-compose#7847.
    color_twin: str = ""
    # Per-role rather than per-meld (agent-compose#396): a meld is not a unique
    # key, since two roles sharing one fail the favorite-colour floor.
    creature: str = ""
    # Selects the animal clade a seat is drawn from, never its colour. Kai's
    # assignment, mirrored by the renderer (agentic-os-xxx#60).
    element: str = ""
    # The class of fact this role's function depends on, per-role because the
    # evidence differs.
    grounding: str = ""
    # The slug of this role's guardrail, or empty. Four roles carry one by Kai's
    # scope decision, so absence is a decision rather than a gap.
    guardrail: str = ""
    outro: Outro | None = None
    voice: Voice | None = None
    acts: list[Act] = field(default_factory=list)
    # Retired from selection, kept whole and still a valid adjacency target.
    archived: bool = False

    def active_boundaries(self, boundaries: dict[str, Boundary]) -> list[str]:
        """Deferred, then scoped, then the single boundary this role owns."""
        active = list(self.defers)
        active += [entry.name for entry in self.scoped]
        active += [name for name, b in boundaries.items() if b.owner == self.name]
        return active


@dataclass
class Roster:
    person: str
    source: str
    invariant: str
    boundary_order: list[str]
    boundaries: dict[str, Boundary]
    personalities: dict[str, Personality]
    guardrail_order: list[str]
    guardrails: dict[str, Guardrail]
    role_order: list[str]
    roles: dict[str, Role]
    raw: bytes = field(default=b"", repr=False)

    def resolve_favorite_colors(self) -> None:
        """Derive every untwinned role's favorite together, in role order.

        A role naming color_twin sits out of the solve and copies its named
        role's result once every untwinned role is resolved. agent-compose#7847.
        """
        solved = [name for name in self.role_order if not self.roles[name].color_twin]
        groups = []
        for name in solved:
            role = self.roles[name]
            components = []
            for personality in role.personalities:
                if personality not in self.personalities:
                    raise RosterError(
                        f"role {name!r}: personality {personality!r} has no catalog binding"
                    )
                components.append(self.personalities[personality].color)
            if not components:
                raise RosterError(f"role {name!r} has no personalities to derive a favorite from")
            groups.append(components)
        try:
            derived = color.favorites(groups)
        except color.ColorError as exc:
            raise RosterError(f"derive role favorite colors: {exc}") from exc
        for name, value in zip(solved, derived, strict=True):
            self.roles[name].favorite_color = value
        for name in self.role_order:
            twin = self.roles[name].color_twin
            if not twin:
                continue
            if twin not in self.roles:
                raise RosterError(f"role {name!r}: color_twin {twin!r} is not defined")
            if self.roles[twin].color_twin:
                raise RosterError(
                    f"role {name!r}: color_twin {twin!r} is itself a twin, no chaining"
                )
            self.roles[name].favorite_color = self.roles[twin].favorite_color


def _voice(spec: dict[str, Any] | None) -> Voice | None:
    if not spec:
        return None
    return Voice(
        summary=str(spec["summary"]),
        cadence=str(spec.get("cadence", "")),
        person=str(spec.get("person", "")),
        tell=str(spec.get("tell", "")),
        prefer=[str(x) for x in spec.get("prefer", []) or []],
        avoid=[str(x) for x in spec.get("avoid", []) or []],
    )


def _acts(spec: list[dict[str, Any]] | None) -> list[Act]:
    return [
        Act(tool=str(a["tool"]), text=str(a["text"]), side=str(a.get("side", "")))
        for a in spec or []
    ]


def _outro(spec: dict[str, Any] | None) -> Outro | None:
    if not spec:
        return None
    return Outro(clean=str(spec.get("clean", "")), failure=str(spec.get("failure", "")))


def _guardrail(name: str, spec: dict[str, Any]) -> Guardrail:
    attests = spec.get("attests") or {}
    return Guardrail(
        name=name,
        skill=spec["skill"],
        role=spec["role"],
        card=str(spec.get("card", "")),
        body=str(spec.get("body", "")),
        detector=str(spec.get("detector", "")),
        attests_in=str(attests.get("in", "")),
        attests_out=str(attests.get("out", "")),
    )


def load(path: pathlib.Path | str = DATA) -> Roster:
    path = pathlib.Path(path)
    raw = path.read_bytes()
    doc = yaml.safe_load(raw)

    boundaries = {
        name: Boundary(
            name=name,
            skill=spec["skill"],
            owner=spec["owner"],
            summary=spec["summary"],
            body=spec["body"],
            acts=_acts(spec.get("acts")),
        )
        for name, spec in doc["boundaries"].items()
    }
    personalities = {
        name: Personality(
            name=name,
            skill=spec["skill"],
            color=spec["color"],
            motif=spec["motif"],
            emblem=Emblem(spec["emblem"]["names"], spec["emblem"]["emoji"]),
            body=spec["body"],
            voice=_voice(spec.get("voice")),
            acts=_acts(spec.get("acts")),
        )
        for name, spec in doc["personalities"].items()
    }
    guardrails = {
        name: _guardrail(name, spec) for name, spec in (doc.get("guardrails") or {}).items()
    }
    roles = {}
    for name, spec in doc["roles"].items():
        roles[name] = Role(
            name=name,
            display_name=spec["display_name"],
            purpose=spec["purpose"],
            skill=spec["skill"],
            skill_source=spec["skill_source"],
            stance=spec["stance"],
            voice=_voice(spec.get("voice")),
            supported_model_tiers=list(spec["supported_model_tiers"]),
            defers=list(spec["defers"]),
            scoped=[Scoped(s["name"], s["scope"]) for s in spec.get("scoped") or []],
            personalities=list(spec["personalities"]),
            identity_name=spec["identity"]["name"],
            identity_pronouns=spec["identity"]["pronouns"],
            seats=[
                Seat(s["key"], s["harness"], s.get("tier"), s.get("channel"), s.get("legal_name"))
                for s in spec["seats"]
            ],
            adjacents=[Adjacent(a["role"], a["reason"]) for a in spec.get("adjacents") or []],
            body=spec["body"],
            methods=list(spec.get("methods") or []),
            creature=str(spec.get("creature", "")),
            element=str(spec.get("element", "")),
            grounding=str(spec.get("grounding", "")),
            guardrail=str(spec.get("guardrail", "")),
            color_twin=str(spec.get("color_twin", "")),
            outro=_outro(spec.get("outro")),
            acts=_acts(spec.get("acts")),
            archived=bool(spec.get("archived", False)),
        )
    roster = Roster(
        person=doc["person"],
        source=doc["source"],
        invariant=doc["invariant"],
        boundary_order=list(doc["boundary_order"]),
        boundaries=boundaries,
        personalities=personalities,
        guardrail_order=list(doc.get("guardrail_order") or []),
        guardrails=guardrails,
        role_order=list(doc["role_order"]),
        roles=roles,
        raw=raw,
    )
    validate(roster)
    roster.resolve_favorite_colors()
    return roster


def validate(roster: Roster) -> None:
    """Every rule the Go loader applies, in the order it applies them."""
    from checks import validate as rules

    rules.check_boundary_ownership(roster)
    rules.check_personality_bindings(roster)
    rules.check_definition_set(roster)
    rules.check_personality_colors(roster)
    rules.check_creature_names(roster)
    rules.check_grounding_lanes(roster)
    rules.check_guardrails(roster)
    rules.check_skill_frontmatter(roster)
    rules.check_copy_contract(roster)
