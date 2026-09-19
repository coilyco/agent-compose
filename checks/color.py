"""Favorite-color math ported from internal/color/color.go.

Only the compose path is here: hex parsing, the legible band, and the joint
favorite solve. Shimmer, Nearest, ANSI, and Backgrounds are palette and theme
tooling that no bundle depends on, so they stay in Go.

Go's math.Round rounds half away from zero and Python's round() rounds half to
even, so delinearize floors x+0.5 instead. That one line is the whole reason
the two engines agree on a hex digit.
"""

from __future__ import annotations

import math
from dataclasses import dataclass

MIN_L = 0.60
MAX_L = 0.80
MIN_CHROMA = 0.05

DRIFT_CAP = 0.03
SPREAD_ROUNDS = 400
SPREAD_STEP = 0.004
SHARE_WEIGHT = 2


@dataclass(frozen=True)
class OKLab:
    L: float
    A: float
    B: float

    @property
    def chroma(self) -> float:
        result: float = math.hypot(self.A, self.B)
        return result


class ColorError(ValueError):
    """A hex color is malformed or falls outside the legible band."""


def parse_hex(value: str) -> tuple[int, int, int]:
    raw = value[1:] if value.startswith("#") else value
    if len(raw) != 6:
        raise ColorError(f"color {value!r} must be #rrggbb")
    try:
        return int(raw[0:2], 16), int(raw[2:4], 16), int(raw[4:6], 16)
    except ValueError as exc:
        raise ColorError(f"color {value!r} must be #rrggbb: {exc}") from exc


def _linearize(channel: int) -> float:
    c = channel / 255
    if c <= 0.04045:
        return c / 12.92
    linear: float = ((c + 0.055) / 1.055) ** 2.4
    return linear


def _delinearize(c: float) -> int:
    c = max(0.0, min(1.0, c))
    if c <= 0.0031308:
        c *= 12.92
    else:
        c = 1.055 * (c ** (1 / 2.4)) - 0.055
    return math.floor(c * 255 + 0.5)


def to_oklab(value: str) -> OKLab:
    ri, gi, bi = parse_hex(value)
    r, g, b = _linearize(ri), _linearize(gi), _linearize(bi)
    lc = math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b)
    mc = math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b)
    sc = math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b)
    return OKLab(
        L=0.2104542553 * lc + 0.7936177850 * mc - 0.0040720468 * sc,
        A=1.9779984951 * lc - 2.4285922050 * mc + 0.4505937099 * sc,
        B=0.0259040371 * lc + 0.7827717662 * mc - 0.8086757660 * sc,
    )


def _to_lms(lab: OKLab) -> tuple[float, float, float]:
    long = (lab.L + 0.3963377774 * lab.A + 0.2158037573 * lab.B) ** 3
    medium = (lab.L - 0.1055613458 * lab.A - 0.0638541728 * lab.B) ** 3
    short = (lab.L - 0.0894841775 * lab.A - 1.2914855480 * lab.B) ** 3
    return long, medium, short


def from_oklab(lab: OKLab) -> str:
    long, medium, short = _to_lms(lab)
    r = 4.0767416621 * long - 3.3077115913 * medium + 0.2309699292 * short
    g = -1.2684380046 * long + 2.6097574011 * medium - 0.3413193965 * short
    b = -0.0041960863 * long - 0.7034186147 * medium + 1.7076147010 * short
    return f"#{_delinearize(r):02x}{_delinearize(g):02x}{_delinearize(b):02x}"


def _in_gamut(lab: OKLab) -> bool:
    long, medium, short = _to_lms(lab)
    channels = (
        4.0767416621 * long - 3.3077115913 * medium + 0.2309699292 * short,
        -1.2684380046 * long + 2.6097574011 * medium - 0.3413193965 * short,
        -0.0041960863 * long - 0.7034186147 * medium + 1.7076147010 * short,
    )
    return all(-0.0001 <= channel <= 1.0001 for channel in channels)


def legible(value: str) -> None:
    """Raise ColorError unless the color sits inside the legible band."""
    lab = to_oklab(value)
    if lab.L < MIN_L or lab.L > MAX_L:
        raise ColorError(
            f"color {value} has OKLab lightness {lab.L:.2f} outside the "
            f"terminal-legible band [{MIN_L:.2f}, {MAX_L:.2f}]"
        )
    if lab.chroma < MIN_CHROMA:
        raise ColorError(
            f"color {value} is too gray (chroma {lab.chroma:.3f}, floor {MIN_CHROMA:.3f})"
        )


def _settle(lab: OKLab) -> OKLab:
    lightness = max(MIN_L, min(MAX_L, lab.L))
    a, b = lab.A, lab.B
    lab = OKLab(lightness, a, b)
    for _ in range(64):
        if _in_gamut(lab):
            break
        a *= 0.98
        b *= 0.98
        lab = OKLab(lightness, a, b)
    chroma = lab.chroma
    if 0 < chroma < MIN_CHROMA:
        scale = MIN_CHROMA / chroma
        lab = OKLab(lightness, a * scale, b * scale)
    return lab


def _weighted_blend(hexes: list[str], weights: list[float]) -> OKLab:
    sum_l = sum_a = sum_b = 0.0
    total = 0.0
    min_component_chroma = math.inf
    for value, weight in zip(hexes, weights, strict=True):
        lab = to_oklab(value)
        sum_l += weight * lab.L
        sum_a += weight * lab.A
        sum_b += weight * lab.B
        total += weight
        min_component_chroma = min(min_component_chroma, lab.chroma)
    blend = OKLab(sum_l / total, sum_a / total, sum_b / total)
    chroma = blend.chroma
    if 0 < chroma < min_component_chroma:
        scale = min_component_chroma / chroma
        blend = OKLab(blend.L, blend.A * scale, blend.B * scale)
    return _settle(blend)


def _separation(a: OKLab, b: OKLab) -> float:
    return math.sqrt((a.L - b.L) ** 2 + (a.A - b.A) ** 2 + (a.B - b.B) ** 2)


def _min_separation(labs: list[OKLab]) -> float:
    lowest = math.inf
    for i in range(len(labs)):
        for j in range(i + 1, len(labs)):
            lowest = min(lowest, _separation(labs[i], labs[j]))
    return lowest


def _step(from_lab: OKLab, away: OKLab) -> OKLab:
    dl, da, db = from_lab.L - away.L, from_lab.A - away.A, from_lab.B - away.B
    norm = math.sqrt(dl * dl + da * da + db * db)
    if norm < 1e-9:
        dl, da, db, norm = 1.0, 0.0, 0.0, 1.0
    return OKLab(
        from_lab.L + dl * SPREAD_STEP / norm,
        from_lab.A + da * SPREAD_STEP / norm,
        from_lab.B + db * SPREAD_STEP / norm,
    )


def _project(candidate: OKLab, anchor: OKLab) -> OKLab:
    dl, da, db = candidate.L - anchor.L, candidate.A - anchor.A, candidate.B - anchor.B
    drift = math.sqrt(dl * dl + da * da + db * db)
    if drift > DRIFT_CAP:
        scale = DRIFT_CAP / drift
        candidate = OKLab(anchor.L + dl * scale, anchor.A + da * scale, anchor.B + db * scale)
    return _settle(candidate)


def _spread_apart(anchors: list[OKLab]) -> list[OKLab]:
    current = list(anchors)
    best = list(anchors)
    best_separation = _min_separation(current)
    for _ in range(SPREAD_ROUNDS):
        for index in range(len(current)):
            nearest, distance = -1, math.inf
            for other in range(len(current)):
                if other == index:
                    continue
                d = _separation(current[index], current[other])
                if d < distance:
                    nearest, distance = other, d
            if nearest < 0:
                continue
            current[index] = _project(_step(current[index], current[nearest]), anchors[index])
        separation = _min_separation(current)
        if separation > best_separation:
            best_separation = separation
            best = list(current)
    return best


def favorites(groups: list[list[str]]) -> list[str]:
    """Derive every group's color together, then spread the results apart."""
    if not groups:
        raise ColorError("favorites needs at least one group")
    shared: dict[str, int] = {}
    for group in groups:
        for value in dict.fromkeys(group):
            shared[value] = shared.get(value, 0) + 1
    anchors = []
    for index, group in enumerate(groups):
        weights = [1 / (shared[value] ** SHARE_WEIGHT) for value in group]
        try:
            anchors.append(_weighted_blend(group, weights))
        except ColorError as exc:
            raise ColorError(f"group {index}: {exc}") from exc
    out = []
    for index, lab in enumerate(_spread_apart(anchors)):
        value = from_oklab(lab)
        try:
            legible(value)
        except ColorError as exc:
            raise ColorError(f"group {index}: {exc}") from exc
        out.append(value)
    return out
