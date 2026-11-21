"""Greet someone: `bar-cost-demo hello <name>`."""

NAME = "hello"


def run(args: list[str]) -> str:
    """The greeting for the first argument, or for the world."""
    who = args[0] if args else "World"
    return f"Hello, {who}!"
