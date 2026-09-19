"""Roster validation ported from internal/person.

The Go tests are the specification, so the messages here quote theirs closely
enough that a reader can find the Go check from a Python failure. Word and
paragraph counting reproduce roleSkillBodyWordCount and briefingParagraphCount
exactly, including dropping a leading `# ` heading before counting.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

from checks import color
from checks.roster import RosterError

if TYPE_CHECKING:
    from checks.roster import Roster

MIN_ROLE_BODY_WORDS = 140
MAX_ROLE_BODY_WORDS = 1200
MIN_PERSONALITY_BODY_WORDS = 120
MAX_PERSONALITY_BODY_WORDS = 320
MIN_ROLE_PARAGRAPHS = 3
MIN_BOUNDARY_SIDE_WORDS = 80

BOUNDARY_OWN_HEADING = "## If you own this boundary"
BOUNDARY_DEFER_HEADING = "## If you defer this boundary"
BOUNDARY_SCOPED_HEADING = "## If you hold this boundary within a scope"


def word_count(body: str) -> int:
    content = body.replace("\r\n", "\n").strip()
    first, _, remainder = content.partition("\n")
    if remainder and first.strip().startswith("# "):
        content = remainder
    return len(content.split())


def paragraph_count(body: str) -> int:
    normalized = body.replace("\r\n", "\n")
    return sum(1 for para in normalized.split("\n\n") if para.strip())


def split_frontmatter(raw: str, skill: str) -> tuple[str, str]:
    text = raw.replace("\r\n", "\n")
    if not text.startswith("---\n"):
        raise _error(f"person skill {skill!r}: SKILL.md needs YAML frontmatter")
    end = text[4:].find("\n---\n")
    if end < 0:
        raise _error(f"person skill {skill!r}: SKILL.md has unterminated frontmatter")
    return text[4 : 4 + end], text[4 + end + 5 :]


def description(raw: str, skill: str) -> str:
    """The first sentence of the frontmatter description, as the card renders it."""
    frontmatter, _ = split_frontmatter(raw, skill)
    for line in frontmatter.split("\n"):
        if line.startswith("description: "):
            value = line[len("description: ") :]
            sentence, sep, _ = value.partition(". ")
            return sentence + "." if sep else value.strip()
    raise _error(f"person skill {skill!r}: missing description")


def _error(message: str) -> RosterError:
    return RosterError(message)


def check_boundary_ownership(roster: Roster) -> None:
    """An owner receives the body by owning it, never by declaring it."""
    for name in roster.boundary_order:
        boundary = roster.boundaries[name]
        if not boundary.owner:
            raise _error(f"boundary {name!r} has no owner")
        role = roster.roles.get(boundary.owner)
        if role is None:
            raise _error(f"boundary {name!r} names unknown owner {boundary.owner!r}")
        if name in role.defers:
            raise _error(f"boundary {name!r} owner {boundary.owner!r} also declares it")
        if any(entry.name == name for entry in role.scoped):
            raise _error(f"boundary {name!r} owner {boundary.owner!r} also scopes it")


def check_personality_bindings(roster: Roster) -> None:
    for role_name in roster.role_order:
        role = roster.roles[role_name]
        if not role.personalities:
            raise _error(f"role {role_name!r} activates no personalities")
        for personality in role.personalities:
            if personality not in roster.personalities:
                raise _error(
                    f"role {role_name!r}: personality {personality!r} has no catalog binding"
                )


def check_definition_set(roster: Roster) -> None:
    """Every declared name resolves, and every definition is reachable."""
    if set(roster.boundary_order) != set(roster.boundaries):
        raise _error("boundary_order and the boundary definitions are a mismatched set")
    if set(roster.role_order) != set(roster.roles):
        raise _error("role_order and the role definitions are a mismatched set")
    bound = {p for role in roster.roles.values() for p in role.personalities}
    orphans = sorted(set(roster.personalities) - bound)
    if orphans:
        raise _error(f"personalities defined but bound to no role: {', '.join(orphans)}")
    for role_name in roster.role_order:
        for name in roster.roles[role_name].defers:
            if name not in roster.boundaries:
                raise _error(f"role {role_name!r} defers unknown boundary {name!r}")
        for entry in roster.roles[role_name].scoped:
            if entry.name not in roster.boundaries:
                raise _error(f"role {role_name!r} scopes unknown boundary {entry.name!r}")


def check_personality_colors(roster: Roster) -> None:
    for name in sorted(roster.personalities):
        try:
            color.legible(roster.personalities[name].color)
        except color.ColorError as exc:
            raise _error(f"personality {name!r} color: {exc}") from exc


def check_creature_names(roster: Roster) -> None:
    """Named per role and unique, the way emblem names are.

    Uniqueness is the point rather than tidiness: the creature is one of the
    four parts a seat states as its identity, and two seats answering with the
    same one makes that sentence stop identifying anybody.
    """
    seen: dict[str, str] = {}
    for name in roster.role_order:
        creature = roster.roles[name].creature
        if not creature:
            raise _error(f"role {name!r} has no creature")
        if creature in seen:
            raise _error(f"creature {creature!r} is on both {seen[creature]!r} and {name!r}")
        seen[creature] = name
        element = roster.roles[name].element
        if not element:
            raise _error(f"role {name!r} has no element")
        head = creature.split("-")[0].lower()
        if head != element:
            raise _error(f"role {name!r}: creature opens {head!r}, element is {element!r}")


def check_grounding_lanes(roster: Roster) -> None:
    """Every role names the class of fact its function depends on.

    Required rather than optional because the board derives from the roster: a
    role with no lane silently contributes no grounding pair, and coverage
    reports a gap it cannot see. See docs/grading.md.
    """
    for name in roster.role_order:
        if not roster.roles[name].grounding:
            raise _error(f"role {name!r} has no grounding lane")


def check_guardrails(roster: Roster) -> None:
    """Shape and binding, not presence, because absence is a decision.

    Kai scoped the primitive to four roles on 2026-09-10: science, advocate,
    director and sysadmin. Frontend, platform and gamedev are deliberately
    without one, so presence stays unenforced and a role with no guardrail
    correctly derives no case.

    The binding is checked in both directions. A guardrail is a first-class
    element now, so the role names the slug and the guardrail names the role, and
    a half-written binding leaves either an element that reaches no bundle or a
    role pointing at nothing. Both used to be representable and neither was
    caught.
    """
    if set(roster.guardrail_order) != set(roster.guardrails):
        raise _error("guardrail_order and the guardrails map name different sets")

    for name in roster.guardrail_order:
        guardrail = roster.guardrails[name]
        if not guardrail.card.strip():
            raise _error(f"guardrail {name!r} has no card half")
        if not guardrail.body.strip():
            raise _error(f"guardrail {name!r} has no body half")
        authored = (guardrail.attests_in.strip(), guardrail.attests_out.strip())
        if guardrail.reproducible:
            if any(authored):
                raise _error(
                    f"guardrail {name!r} names a detector and authors attests targets, "
                    "which the deriver generates and would never read"
                )
        elif not all(authored):
            raise _error(
                f"guardrail {name!r} is attested without both attests targets, "
                "and there is no generic text to fall back on"
            )
        if guardrail.role not in roster.roles:
            raise _error(f"guardrail {name!r} names unknown role {guardrail.role!r}")
        if roster.roles[guardrail.role].guardrail != name:
            raise _error(
                f"guardrail {name!r} claims role {guardrail.role!r}, which does not "
                "name it back, so one side of the binding is unreachable"
            )

    for name in roster.role_order:
        slug = roster.roles[name].guardrail
        if slug and slug not in roster.guardrails:
            raise _error(f"role {name!r} names unknown guardrail {slug!r}")


def check_skill_frontmatter(roster: Roster) -> None:
    """Every skill body declares its own name in frontmatter."""
    entries = [(r.skill, r.body) for r in roster.roles.values()]
    entries += [(p.skill, p.body) for p in roster.personalities.values()]
    entries += [(b.skill, b.body) for b in roster.boundaries.values()]
    entries += [(g.skill, g.body) for g in roster.guardrails.values()]
    for skill, body in entries:
        frontmatter, _ = split_frontmatter(body, skill)
        if f"\nname: {skill}\n" not in "\n" + frontmatter + "\n":
            raise _error(f"person skill {skill!r}: frontmatter does not declare name {skill!r}")
        description(body, skill)


def check_copy_contract(roster: Roster) -> None:
    """Prose bounds: the role charter, the personality entries, both boundary sides."""
    for role_name in roster.role_order:
        role = roster.roles[role_name]
        _, body = split_frontmatter(role.body, role.skill)
        words = word_count(body)
        if words > MAX_ROLE_BODY_WORDS:
            raise _error(
                f"role {role_name!r} skill body has {words} words, maximum is {MAX_ROLE_BODY_WORDS}"
            )
        paragraphs = paragraph_count(body)
        if paragraphs < MIN_ROLE_PARAGRAPHS:
            raise _error(
                f"role {role_name!r} skill needs at least three paragraphs, got {paragraphs}"
            )
    for name in sorted(roster.personalities):
        personality = roster.personalities[name]
        _, body = split_frontmatter(personality.body, personality.skill)
        words = word_count(body)
        if words > MAX_PERSONALITY_BODY_WORDS:
            raise _error(
                f"personality {name!r} skill body has {words} words, "
                f"maximum is {MAX_PERSONALITY_BODY_WORDS}"
            )
    for name in roster.boundary_order:
        boundary = roster.boundaries[name]
        _, body = split_frontmatter(boundary.body, boundary.skill)
        own = body.find(BOUNDARY_OWN_HEADING)
        defer = body.find(BOUNDARY_DEFER_HEADING)
        if own < 0 or defer < 0:
            raise _error(
                f"boundary {name!r} skill body needs both "
                f"{BOUNDARY_OWN_HEADING!r} and {BOUNDARY_DEFER_HEADING!r} sections"
            )
        if own > defer:
            raise _error(f"boundary {name!r} skill body states the defer side before the own side")


def _boundary_sides(body: str) -> dict[str, str]:
    """Split a boundary body into its own/scoped/defer prose, as Go does."""
    own = body.find(BOUNDARY_OWN_HEADING)
    defer = body.find(BOUNDARY_DEFER_HEADING)
    if own < 0 or defer < 0:
        return {}
    sections = {"own": body[own:defer], "defer": body[defer:]}
    scoped = body.find(BOUNDARY_SCOPED_HEADING)
    if scoped >= 0:
        sections["own"] = body[own:scoped]
        sections["scoped"] = body[scoped:defer]
    return {label: section.partition("\n")[2] for label, section in sections.items()}


def check_prose_floors(roster: Roster) -> None:
    """Keep a shipped entry from thinning into a label.

    Ported from Go's validateRosterProseFloors, which the Go engine calls only
    from shipped_roster_test.go and never from its load path. The ceilings in
    check_copy_contract bind every package; these floors bind the roster this
    repo ships. So this is a test-called check: registering it in validate()
    rejects legitimate thin fixtures, which is how the port first went wrong.
    """
    for role_name in roster.role_order:
        role = roster.roles[role_name]
        _, body = split_frontmatter(role.body, role.skill)
        words = word_count(body)
        if words < MIN_ROLE_BODY_WORDS:
            raise _error(
                f"role {role_name!r} body has {words} words, minimum is {MIN_ROLE_BODY_WORDS}"
            )
    for name in sorted(roster.personalities):
        personality = roster.personalities[name]
        _, body = split_frontmatter(personality.body, personality.skill)
        words = word_count(body)
        if words < MIN_PERSONALITY_BODY_WORDS:
            raise _error(
                f"personality {name!r} body has {words} words, "
                f"minimum is {MIN_PERSONALITY_BODY_WORDS}"
            )
    for name in roster.boundary_order:
        boundary = roster.boundaries[name]
        _, body = split_frontmatter(boundary.body, boundary.skill)
        for label, prose in _boundary_sides(body).items():
            words = word_count(prose)
            if words < MIN_BOUNDARY_SIDE_WORDS:
                raise _error(
                    f"boundary {name!r} {label} side has {words} words, "
                    f"minimum is {MIN_BOUNDARY_SIDE_WORDS}"
                )
