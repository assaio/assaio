#!/usr/bin/env python3
"""Checks the harness under .claude/ is well-formed: frontmatter parses, every rule is
path-scoped and its globs hit a tracked file, every path a harness file cites exists, the
always-loaded AGENTS.md stays inside its line budget, and no Fable/Mythos id is assigned
anywhere. Exit 1 on the first class of problem found; every problem is printed.

Usage: setup.py [--root DIR]
"""
import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

AGENTS_LINE_BUDGET = 150
DESCRIPTION_RANGE = (60, 1024)
AGENT_MODELS = {"haiku", "sonnet", "opus"}
FABLE = re.compile(r'(model|--model|--judge|--candidates|-m)["\']?\s*[:=\s]\s*["\']?(claude-)?(fable|mythos)', re.I)
CITED = re.compile(r"`([^`\s]+)`")
PATH_EXT = (".md", ".go", ".sh", ".py", ".json", ".yml", ".yaml", ".toml", ".sql", ".html", ".cff", ".txt", ".tmpl")


def tracked(root):
    out = subprocess.run(["git", "-C", str(root), "ls-files"], capture_output=True, text=True, check=False)
    return set(out.stdout.split("\n")) - {""}


def frontmatter(text, path, problems):
    lines = text.split("\n")
    if not lines or lines[0].strip() != "---":
        problems.append("%s: no frontmatter" % path)
        return {}
    try:
        end = lines[1:].index("---") + 1
    except ValueError:
        problems.append("%s: frontmatter never closes" % path)
        return {}
    data, key = {}, None
    for n, line in enumerate(lines[1:end], start=2):
        if not line.strip():
            continue
        if "\t" in line:
            problems.append("%s:%d: tab in frontmatter" % (path, n))
        if re.match(r"^\s+-\s", line) and key is not None:
            data.setdefault(key, []).append(line.strip()[2:].strip().strip("\"'"))
            continue
        m = re.match(r"^([A-Za-z][A-Za-z0-9_-]*):(?:\s*(.*))?$", line)
        if not m:
            problems.append("%s:%d: not `key: value`: %s" % (path, n, line.strip()))
            continue
        key, value = m.group(1), (m.group(2) or "").strip()
        if value == "":
            data[key] = []
            continue
        if value[0] in "\"'":
            if value[-1] != value[0]:
                problems.append("%s:%d: unbalanced quote in %s" % (path, n, key))
            data[key] = value[1:-1]
        elif value.startswith("["):
            if not value.endswith("]"):
                problems.append("%s:%d: unclosed flow list in %s" % (path, n, key))
            if key in ("argument-hint", "description") or "[" in value[1:]:
                problems.append("%s:%d: %s starting with '[' parses as a YAML list; quote it" % (path, n, key))
            data[key] = [v.strip().strip("\"'") for v in value[1:-1].split(",") if v.strip()]
        else:
            if ": " in value or value.endswith(":"):
                problems.append("%s:%d: unquoted value with ': ' breaks YAML (%s)" % (path, n, key))
            if "[" in value:
                problems.append("%s:%d: unquoted value with '[' parses as a list (%s)" % (path, n, key))
            data[key] = value
    return data


def check_description(data, path, problems):
    desc = data.get("description")
    if not isinstance(desc, str):
        problems.append("%s: description missing" % path)
        return
    lo, hi = DESCRIPTION_RANGE
    if not lo <= len(desc) <= hi:
        problems.append("%s: description is %d chars, want %d..%d" % (path, len(desc), lo, hi))


def glob_regex(glob):
    out = ""
    i = 0
    while i < len(glob):
        c = glob[i]
        if glob.startswith("**/", i):
            out += "(?:.*/)?"
            i += 3
            continue
        if glob.startswith("**", i):
            out += ".*"
            i += 2
            continue
        if c == "*":
            out += "[^/]*"
        elif c == "?":
            out += "[^/]"
        elif c == "{":
            j = glob.index("}", i)
            out += "(?:" + "|".join(re.escape(a) for a in glob[i + 1:j].split(",")) + ")"
            i = j
        else:
            out += re.escape(c)
        i += 1
    return re.compile("^" + out + "$")


def check_agents(root, problems):
    for path in sorted((root / ".claude/agents").glob("*.md")):
        data = frontmatter(path.read_text(encoding="utf-8"), path, problems)
        if data.get("name") != path.stem:
            problems.append("%s: name must be %s" % (path, path.stem))
        check_description(data, path, problems)
        if data.get("model") not in AGENT_MODELS:
            problems.append("%s: model must be one of %s (never inherit: the session may run Fable)" % (path, sorted(AGENT_MODELS)))
        if "tools" not in data:
            problems.append("%s: tools must be listed" % path)


def check_skills(root, problems):
    for path in sorted((root / ".claude/skills").glob("*/SKILL.md")):
        data = frontmatter(path.read_text(encoding="utf-8"), path, problems)
        if data.get("name") != path.parent.name:
            problems.append("%s: name must be %s" % (path, path.parent.name))
        check_description(data, path, problems)
    for d in sorted(p for p in (root / ".claude/skills").iterdir() if p.is_dir()):
        if not (d / "SKILL.md").exists():
            problems.append("%s: skill directory without SKILL.md" % d)
    if (root / ".claude/commands").exists():
        problems.append(".claude/commands/ still exists; commands were migrated to skills")


def check_rules(root, files, problems):
    for path in sorted((root / ".claude/rules").glob("*.md")):
        data = frontmatter(path.read_text(encoding="utf-8"), path, problems)
        paths = data.get("paths")
        if not paths:
            problems.append("%s: a rule without paths loads into every session; scope it" % path)
            continue
        for g in paths:
            rx = glob_regex(g)
            if not any(rx.match(f) for f in files):
                problems.append("%s: glob %s matches no tracked file" % (path, g))


def looks_like_path(token):
    """A repository-relative file: has a directory part and a known extension. Bare basenames,
    directories, absolute paths, URLs and skill names are not checked."""
    if any(ch in token for ch in "<>*{}$~@:") or token.startswith(("-", "/", "./", "../")):
        return False
    return "/" in token and token.endswith(PATH_EXT)


def check_citations(root, files, problems):
    sources = [root / "AGENTS.md", root / "CLAUDE.md"]
    sources += sorted((root / ".claude").rglob("*.md"))
    if (root / "docs/operations.md").exists():
        sources.append(root / "docs/operations.md")
    for path in sources:
        for token in CITED.findall(path.read_text(encoding="utf-8")):
            token = token.rstrip(".,;:)")
            if not looks_like_path(token):
                continue
            if token in files or (root / token).exists() or (path.parent / token).exists():
                continue
            problems.append("%s cites `%s`, which does not exist" % (path.relative_to(root), token))


def check_budget_and_models(root, problems):
    agents_md = root / "AGENTS.md"
    if agents_md.exists():
        n = len(agents_md.read_text(encoding="utf-8").split("\n"))
        if n > AGENTS_LINE_BUDGET:
            problems.append("AGENTS.md is %d lines; it loads into every session and subagent, budget is %d" % (n, AGENTS_LINE_BUDGET))
    for path in sorted((root / ".claude").rglob("*")):
        if path.is_file() and path.suffix in (".md", ".json", ".sh", ".py"):
            for n, line in enumerate(path.read_text(encoding="utf-8", errors="replace").split("\n"), start=1):
                if FABLE.search(line) and "fable_assignment" not in line and "FABLE" not in line and not path.name.endswith("_test.sh"):
                    problems.append("%s:%d assigns a Fable/Mythos model" % (path.relative_to(root), n))


def check_settings(root, problems):
    path = root / ".claude/settings.json"
    try:
        settings = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError) as err:
        problems.append("%s: %s" % (path, err))
        return
    for event, entries in settings.get("hooks", {}).items():
        for entry in entries:
            for hook in entry.get("hooks", []):
                for token in re.findall(r"\.claude/[A-Za-z0-9_./-]+", hook.get("command", "")):
                    if not (root / token).exists():
                        problems.append("settings.json %s hook names %s, which does not exist" % (event, token))


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args(argv)
    root = args.root.resolve()
    files = tracked(root)
    problems = []
    check_agents(root, problems)
    check_skills(root, problems)
    check_rules(root, files, problems)
    check_citations(root, files, problems)
    check_budget_and_models(root, problems)
    check_settings(root, problems)
    for p in problems:
        print("setup: %s" % p)
    if problems:
        return 1
    print("setup: ok (%d agents, %d skills, %d rules)" % (
        len(list((root / ".claude/agents").glob("*.md"))),
        len(list((root / ".claude/skills").glob("*/SKILL.md"))),
        len(list((root / ".claude/rules").glob("*.md")))))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
