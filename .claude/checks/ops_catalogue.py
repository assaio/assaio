#!/usr/bin/env python3
"""Checks docs/operations.md against the tree in both directions: every operator entry point
(Makefile target, workflow, script, hook, check) has a row, and every row names something that
exists. A catalogue that lists what is gone, or misses what was added, is worse than none.

Usage: ops_catalogue.py [--root DIR]
"""
import argparse
import re
import sys
from pathlib import Path

CATALOGUE = "docs/operations.md"
SCRIPT_DIRS = (".claude/hooks", ".claude/checks", ".claude/scripts", ".lefthook", "docs/assets")
ROW = re.compile(r"^\|\s*`([^`]+)`")


def entry_points(root):
    found = set()
    makefile = (root / "Makefile").read_text(encoding="utf-8")
    for m in re.finditer(r"^([A-Za-z][A-Za-z0-9_-]*):", makefile, re.M):
        found.add("make " + m.group(1))
    for wf in sorted((root / ".github/workflows").glob("*.yml")):
        found.add(".github/workflows/" + wf.name)
    for d in SCRIPT_DIRS:
        for p in sorted((root / d).rglob("*")):
            if p.is_file() and p.suffix in (".sh", ".py"):
                found.add(str(p.relative_to(root)))
    return found


def catalogued(root):
    path = root / CATALOGUE
    if not path.exists():
        return None
    rows = set()
    for line in path.read_text(encoding="utf-8").split("\n"):
        m = ROW.match(line)
        if m:
            rows.add(m.group(1).strip())
    return rows


def exists(root, name):
    if name.startswith("make "):
        target = name[5:]
        return re.search(r"^%s:" % re.escape(target), (root / "Makefile").read_text(encoding="utf-8"), re.M) is not None
    if name.startswith("assaio-agent "):
        return True
    return (root / name).exists()


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args(argv)
    root = args.root.resolve()
    rows = catalogued(root)
    if rows is None:
        print("ops-catalogue: %s is missing" % CATALOGUE)
        return 1
    points = entry_points(root)
    problems = ["%s has no row in %s" % (p, CATALOGUE) for p in sorted(points - rows)]
    problems += ["%s row `%s` names nothing in the tree" % (CATALOGUE, r) for r in sorted(rows) if not exists(root, r)]
    for p in problems:
        print("ops-catalogue: " + p)
    if problems:
        return 1
    print("ops-catalogue: ok (%d entry points, %d rows)" % (len(points), len(rows)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
