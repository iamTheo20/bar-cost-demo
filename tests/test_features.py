"""Every feature is discovered and runs from the command line."""

import pytest

from bar_cost_demo.__main__ import main
from bar_cost_demo.features import discover
from bar_cost_demo.features.hello import feature as hello


def test_hello_greets_by_name() -> None:
    assert hello.run(["Ada"]) == "Hello, Ada!"


def test_hello_defaults_to_the_world() -> None:
    assert hello.run([]) == "Hello, World!"


def test_features_are_discovered_by_folder() -> None:
    assert "hello" in discover()


def test_the_cli_runs_a_feature(capsys: pytest.CaptureFixture[str]) -> None:
    assert main(["hello", "Bo"]) == 0
    assert capsys.readouterr().out == "Hello, Bo!\n"


def test_an_unknown_feature_is_a_usage_error() -> None:
    assert main(["nope"]) == 2
