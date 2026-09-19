"""One RosterError, so `except RosterError` around a load catches the whole load.

`validate` used to define its own class of the same name. Both subclassed
ValueError and neither imported the other, so catching the loader's caught the
colour derivation and the overlay checks and missed all seven validators.
"""

from __future__ import annotations

import ast
import pathlib

from checks import roster, validate

SOURCE = pathlib.Path(validate.__file__)


def test_validate_raises_the_loader_s_class() -> None:
    """`housecast/tests/test_validate.py` names it on all seven checks. This is the seam."""
    assert isinstance(validate._error("x"), roster.RosterError)


def test_every_raise_in_validate_goes_through_that_one_helper() -> None:
    """A direct `raise ValueError(...)` is how the two classes split the first time."""
    tree = ast.parse(SOURCE.read_text(encoding="utf-8"))
    raised = [
        node.exc for node in ast.walk(tree) if isinstance(node, ast.Raise) and node.exc is not None
    ]
    assert raised, "validate.py raises nothing, so this test proves nothing"
    for exc in raised:
        assert isinstance(exc, ast.Call) and isinstance(exc.func, ast.Name), ast.dump(exc)
        assert exc.func.id == "_error", f"raises {exc.func.id} rather than _error"
